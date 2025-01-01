/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package groupcache

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	peerAddrs  = flag.String("test_peer_addrs", "", "Comma-separated list of peer addresses; used by TestHTTPPool")
	peerIndex  = flag.Int("test_peer_index", -1, "Index of which peer this child is; used by TestHTTPPool")
	peerChild  = flag.Bool("test_peer_child", false, "True if running as a child process; used by TestHTTPPool")
	serverAddr = flag.String("test_server_addr", "", "Address of the server Child Getters will hit ; used by TestHTTPPool")
)

func TestHTTPPool(t *testing.T) {
	if *peerChild {
		beChildForTestHTTPPool(t)
		os.Exit(0)
	}

	const (
		nChild = 4
		nGets  = 100
	)

	var serverHits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello")
		serverHits++
	}))
	defer ts.Close()

	var childAddr []string
	for i := 0; i < nChild; i++ {
		childAddr = append(childAddr, pickFreeAddr(t))
	}

	var cmds []*exec.Cmd
	var wg sync.WaitGroup
	for i := 0; i < nChild; i++ {
		cmd := exec.Command(os.Args[0],
			"--test.run=TestHTTPPool",
			"--test_peer_child",
			"--test_peer_addrs="+strings.Join(childAddr, ","),
			"--test_peer_index="+strconv.Itoa(i),
			"--test_server_addr="+ts.URL,
		)
		cmds = append(cmds, cmd)
		cmd.Stdout = os.Stdout
		wg.Add(1)
		if err := cmd.Start(); err != nil {
			t.Fatal("failed to start child process: ", err)
		}
		go awaitAddrReady(t, childAddr[i], &wg)
	}
	defer func() {
		for i := 0; i < nChild; i++ {
			if cmds[i].Process != nil {
				cmds[i].Process.Kill()
			}
		}
	}()
	wg.Wait()

	// Use a dummy self address so that we don't handle gets in-process.
	p, mux := newTestHTTPPool("should-be-ignored")
	defer mux.Close()

	p.Set(addrToURL(childAddr)...)

	// Dummy getter function. Gets should go to children only.
	// The only time this process will handle a get is when the
	// children can't be contacted for some reason.
	getter := GetterFunc(func(ctx context.Context, key string, dest Sink) error {
		return errors.New("parent getter called; something's wrong")
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	g := NewGroup("httpPoolTest", 1<<20, getter)

	for _, key := range testKeys(nGets) {
		var value string
		if err := g.Get(ctx, key, StringSink(&value)); err != nil {
			t.Fatal(err)
		}
		if suffix := ":" + key; !strings.HasSuffix(value, suffix) {
			t.Errorf("Get(%q) = %q, want value ending in %q", key, value, suffix)
		}
		t.Logf("Get key=%q, value=%q (peer:key)", key, value)
	}

	if serverHits != nGets {
		t.Error("expected serverHits to equal nGets")
	}
	serverHits = 0

	var value string
	var key = "removeTestKey"

	// Multiple gets on the same key
	for i := 0; i < 2; i++ {
		if err := g.Get(ctx, key, StringSink(&value)); err != nil {
			t.Fatal(err)
		}
	}

	// Should result in only 1 server get
	if serverHits != 1 {
		t.Error("expected serverHits to be '1'")
	}

	// Remove the key from the cache and we should see another server hit
	if err := g.Remove(ctx, key); err != nil {
		t.Fatal(err)
	}

	// Get the key again
	if err := g.Get(ctx, key, StringSink(&value)); err != nil {
		t.Fatal(err)
	}

	// Should register another server get
	if serverHits != 2 {
		t.Error("expected serverHits to be '2'")
	}

	key = "setMyTestKey"
	setValue := []byte("test set")
	// Add the key to the cache, optionally updating our local hot cache
	if err := g.Set(ctx, key, setValue, time.Time{}, false); err != nil {
		t.Fatal(err)
	}

	// Get the key
	var getValue ByteView
	if err := g.Get(ctx, key, ByteViewSink(&getValue)); err != nil {
		t.Fatal(err)
	}

	if serverHits != 2 {
		t.Errorf("expected serverHits to be '3' got '%d'", serverHits)
	}

	if !bytes.Equal(setValue, getValue.ByteSlice()) {
		t.Fatal(errors.New(fmt.Sprintf("incorrect value retrieved after set: %s", getValue)))
	}

	// Key with non-URL characters to test URL encoding roundtrip
	key = "a b/c,d"
	if err := g.Get(ctx, key, StringSink(&value)); err != nil {
		t.Fatal(err)
	}
	if suffix := ":" + key; !strings.HasSuffix(value, suffix) {
		t.Errorf("Get(%q) = %q, want value ending in %q", key, value, suffix)
	}

	// Get a key that does not exist
	err := g.Get(ctx, "IReturnErrNotFound", StringSink(&value))
	errNotFound := &ErrNotFound{}
	if !errors.As(err, &errNotFound) {
		t.Fatal(errors.New("expected error to be 'ErrNotFound'"))
	}
	assert.Equal(t, "I am a ErrNotFound error", errNotFound.Error())

	// Get a key that is guaranteed to return a remote error.
	err = g.Get(ctx, "IReturnErrRemoteCall", StringSink(&value))
	errRemoteCall := &ErrRemoteCall{}
	if !errors.As(err, &errRemoteCall) {
		t.Fatal(errors.New("expected error to be 'ErrRemoteCall'"))
	}
	assert.Equal(t, "I am a ErrRemoteCall error", errRemoteCall.Error())

	// Get a key that is guaranteed to return an internal (500) error
	err = g.Get(ctx, "IReturnInternalError", StringSink(&value))
	assert.Equal(t, "I am a errors.New() error", err.Error())

}

func testKeys(n int) (keys []string) {
	keys = make([]string, n)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	return
}

func beChildForTestHTTPPool(t *testing.T) {
	addrs := strings.Split(*peerAddrs, ",")

	p, mux := newTestHTTPPool("http://" + addrs[*peerIndex])
	defer mux.Close()
	p.Set(addrToURL(addrs)...)

	getter := GetterFunc(func(ctx context.Context, key string, dest Sink) error {
		if key == "IReturnErrNotFound" {
			return &ErrNotFound{Msg: "I am a ErrNotFound error"}
		}

		if key == "IReturnErrRemoteCall" {
			return &ErrRemoteCall{Msg: "I am a ErrRemoteCall error"}
		}

		if key == "IReturnInternalError" {
			return errors.New("I am a errors.New() error")
		}

		if _, err := http.Get(*serverAddr); err != nil {
			t.Logf("HTTP request from getter failed with '%s'", err)
		}

		dest.SetString(strconv.Itoa(*peerIndex)+":"+key, time.Time{})
		return nil
	})
	NewGroup("httpPoolTest", 1<<20, getter)

	log.Fatal(http.ListenAndServe(addrs[*peerIndex], p))
}

// This is racy. Another process could swoop in and steal the port between the
// call to this function and the next listen call. Should be okay though.
// The proper way would be to pass the l.File() as ExtraFiles to the child
// process, and then close your copy once the child starts.
func pickFreeAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func addrToURL(addr []string) []string {
	url := make([]string, len(addr))
	for i := range addr {
		url[i] = "http://" + addr[i]
	}
	return url
}

func awaitAddrReady(t *testing.T, addr string, wg *sync.WaitGroup) {
	defer wg.Done()
	const max = 1 * time.Second
	tries := 0
	for {
		tries++
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			return
		}
		delay := time.Duration(tries) * 25 * time.Millisecond
		if delay > max {
			delay = max
		}
		time.Sleep(delay)
	}
}

type serveMux struct {
	mux      *http.ServeMux
	handlers map[string]http.Handler
}

func newTestHTTPPool(self string) (*HTTPPool, *serveMux) {
	httpPoolMade, portPicker = false, nil // Testing only

	p := NewHTTPPoolOpts(self, nil)
	sm := &serveMux{
		mux:      http.NewServeMux(),
		handlers: make(map[string]http.Handler),
	}

	sm.handlers[p.opts.BasePath] = p

	return p, sm
}

func (s *serveMux) Handle(pattern string, handler http.Handler) {
	s.handlers[pattern] = handler
	s.mux.Handle(pattern, handler)
}

func (s *serveMux) Close() {
	for pattern := range s.handlers {
		delete(s.handlers, pattern)
	}
}

func (s *serveMux) RemoveHandle(pattern string) {
	delete(s.handlers, pattern)
}

func (s *serveMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.handlers[r.URL.Path]; ok {
		s.mux.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}

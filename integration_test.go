package groupcache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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
	"github.com/stretchr/testify/require"
)

func TestManualSet(t *testing.T) {
	if *peerChild {
		beChildForIntegrationTest(t)
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
			"--test.run=TestManualSet",
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

	assertLocalGets := func(key, expectedValue string) {
		for _, addr := range childAddr {
			resp, err := http.Get("http://" + addr + "/local/" + key)
			assert.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			bytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Equal(t, expectedValue, string(bytes))
		}
	}
	g := NewGroup("integrationTest", 1<<20, getter)

	var got string
	err := g.Get(ctx, "key-0", StringSink(&got))
	assert.NoError(t, err)
	assert.Equal(t, "got:key-0", got)
	// Since nodes have hot caches, we assert that the localGets are returning the right data
	assertLocalGets("key-0", "got:key-0")

	// Manually set the value in the cache
	overwrite := "manual-set"
	err = g.Set(ctx, "key-0", []byte(overwrite), time.Time{}, true)
	assert.NoError(t, err)

	err = g.Get(ctx, "key-0", StringSink(&got))
	assert.NoError(t, err)
	assert.Equal(t, overwrite, got)
	assertLocalGets("key-0", overwrite)
}

type overwriteHttpPool struct {
	g *Group
	p *HTTPPool
}

// ServeHTTP implements http.Handler.
func (o overwriteHttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/local/") {
		fmt.Printf("peer %d group.Get(%s)\n", *peerIndex, r.URL.Path)
		key := strings.TrimPrefix(r.URL.Path, "/local/")
		// Custom logic here
		// For example, you can write the key to the response
		var got string
		err := o.g.Get(r.Context(), key, StringSink(&got))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _ = w.Write([]byte(got))
		return
	}

	// Call the original handler
	o.p.ServeHTTP(w, r)
}

var _ http.Handler = (*overwriteHttpPool)(nil)

func beChildForIntegrationTest(t *testing.T) {
	addrs := strings.Split(*peerAddrs, ",")

	p, mux := newTestHTTPPool("http://" + addrs[*peerIndex])
	defer mux.Close()

	hp := overwriteHttpPool{
		p: p,
	}
	hp.p.Set(addrToURL(addrs)...)

	getter := GetterFunc(func(ctx context.Context, key string, dest Sink) error {
		return dest.SetString("got:"+key, time.Time{})
	})
	hp.g = NewGroup("integrationTest", 1<<20, getter)

	log.Printf("Listening on %s\n", addrs[*peerIndex])
	log.Fatal(http.ListenAndServe(addrs[*peerIndex], hp))
}

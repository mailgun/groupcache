package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mailgun/groupcache/v2"
)

var store = map[string]string{}

var group = groupcache.NewGroup("cache1", 64<<20, groupcache.GetterFunc(
	func(ctx context.Context, key string, dest groupcache.Sink) error {
		fmt.Printf("Get Called\n")
		v, ok := store[key]
		if !ok {
			return fmt.Errorf("key not set")
		} else {
			if err := dest.SetBytes([]byte(v), time.Now().Add(10*time.Minute)); err != nil {
				log.Printf("Failed to set cache value for key '%s' - %v\n", key, err)
				return err
			}
		}

		return nil
	},
))

func main() {
	addr := flag.String("addr", ":8080", "server address")
	addr2 := flag.String("api-addr", ":8081", "api server address")
	peers := flag.String("pool", "http://localhost:8080", "server pool list")
	flag.Parse()

	p := strings.Split(*peers, ",")
	pool := groupcache.NewHTTPPoolOpts(fmt.Sprintf("http://%s", *addr), &groupcache.HTTPPoolOptions{})
	pool.Set(p...)

	http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		key := r.FormValue("key")
		value := r.FormValue("value")
		fmt.Printf("Set: [%s]%s\n", key, value)
		store[key] = value
	})

	http.HandleFunc("/cache", func(w http.ResponseWriter, r *http.Request) {
		key := r.FormValue("key")

		fmt.Printf("Fetching value for key '%s'\n", key)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var b []byte
		err := group.Get(ctx, key, groupcache.AllocatingByteSliceSink(&b))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Write(b)
		w.Write([]byte{'\n'})
	})

	server := http.Server{
		Addr:    *addr,
		Handler: pool,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start HTTP server - %v", err)
		}
	}()

	go func() {
		if err := http.ListenAndServe(*addr2, nil); err != nil {
			log.Fatalf("Failed to start API HTTP server - %v", err)
		}
	}()

	fmt.Printf("Running...\n")
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan
}

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/golang/groupcache"
)

func startGroupcache() *groupcache.Group {

	//
	// create groupcache pool
	//

	groupcachePort := ":5000"

	myURL := "http://127.0.0.1" + groupcachePort

	log.Printf("groupcache my URL: %s", myURL)

	pool := groupcache.NewHTTPPoolOpts(myURL, &groupcache.HTTPPoolOptions{})

	//
	// start groupcache server
	//

	serverGroupCache := &http.Server{Addr: groupcachePort, Handler: pool}

	go func() {
		log.Printf("groupcache server: listening on %s", groupcachePort)
		err := serverGroupCache.ListenAndServe()
		log.Printf("groupcache server: exited: %v", err)
	}()

	pool.Set(myURL)

	//
	// create cache
	//

	var groupcacheSizeBytes int64 = 1_000_000

	// https://talks.golang.org/2013/oscon-dl.slide#46
	//
	// 64 MB max per-node memory usage
	cache := groupcache.NewGroup("files", groupcacheSizeBytes, groupcache.GetterFunc(
		func(ctx context.Context, key string, dest groupcache.Sink) error {

			log.Printf("getter: loading: key:%s", key)

			data, errFile := os.ReadFile(key)
			if errFile != nil {
				return errFile
			}

			return dest.SetBytes(data)
		}))

	return cache
}

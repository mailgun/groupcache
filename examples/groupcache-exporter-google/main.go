// Package main implements the example.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/groupcache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/udhos/groupcache_exporter"
	"github.com/udhos/groupcache_exporter/groupcache/google"
)

func main() {

	appName := filepath.Base(os.Args[0])

	cache := startGroupcache()

	//
	// expose prometheus metrics
	//
	{
		metricsRoute := "/metrics"
		metricsPort := ":3000"

		log.Printf("starting metrics server at: %s %s", metricsPort, metricsRoute)

		google := google.New(cache)
		labels := map[string]string{
			"app": appName,
		}
		namespace := ""
		collector := groupcache_exporter.NewExporter(namespace, labels, google)

		prometheus.MustRegister(collector)

		go func() {
			http.Handle(metricsRoute, promhttp.Handler())
			log.Fatal(http.ListenAndServe(metricsPort, nil))
		}()
	}

	//
	// query cache periodically
	//

	const interval = 5 * time.Second

	for {
		var dst []byte
		cache.Get(context.TODO(), "/etc/passwd", groupcache.AllocatingByteSliceSink(&dst))
		log.Printf("cache answer: %d bytes, sleeping %v", len(dst), interval)
		time.Sleep(interval)
	}

}

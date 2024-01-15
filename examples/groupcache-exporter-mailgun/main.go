// Package main implements the example.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/grafana/groupcache_exporter"
	"github.com/grafana/groupcache_exporter/groupcache/mailgun"
	"github.com/mailgun/groupcache/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

		mailgun := mailgun.New(cache)
		labels := map[string]string{
			"app": appName,
		}
		namespace := ""
		collector := groupcache_exporter.NewExporter(namespace, labels, mailgun)

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

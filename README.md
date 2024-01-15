[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/groupcache_exporter/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/groupcache_exporter)](https://goreportcard.com/report/github.com/udhos/groupcache_exporter)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/groupcache_exporter.svg)](https://pkg.go.dev/github.com/udhos/groupcache_exporter)

# Prometheus Groupcache exporter

This exporter implements the prometheus.Collector interface in order to expose Prometheus metrics for [groupcache](https://github.com/golang/groupcache).

# Example for mailgun groupcache

```golang
import "github.com/udhos/groupcache_exporter/groupcache/mailgun"

// ...

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
```

Full example: [./examples/groupcache/mailgun](./examples/groupcache/mailgun)

# Example for modernprogram groupcache

```golang
import "github.com/udhos/groupcache_exporter/groupcache/modernprogram"

// ...

appName := filepath.Base(os.Args[0])

cache := startGroupcache()

//
// expose prometheus metrics
//
{
    metricsRoute := "/metrics"
    metricsPort := ":3000"

    log.Printf("starting metrics server at: %s %s", metricsPort, metricsRoute)

    modernprogram := modernprogram.New(cache)
    labels := map[string]string{
        "app": appName,
    }
    namespace := ""
    collector := groupcache_exporter.NewExporter(namespace, labels, modernprogram)

    prometheus.MustRegister(collector)

    go func() {
        http.Handle(metricsRoute, promhttp.Handler())
        log.Fatal(http.ListenAndServe(metricsPort, nil))
    }()
}
```

Full example: [./examples/groupcache/modernprogram](./examples/groupcache/modernprogram)

# Testing

## Build

    go install ./...

## Run example application

    groupcache-exporter-mailgun

## Query the metrics endpoint

```bash
curl -s localhost:3000/metrics | grep -E ^groupcache
groupcache_cache_bytes{app="groupcache-exporter-mailgun",group="files",type="hot"} 0
groupcache_cache_bytes{app="groupcache-exporter-mailgun",group="files",type="main"} 2954
groupcache_cache_evictions_total{app="groupcache-exporter-mailgun",group="files",type="hot"} 0
groupcache_cache_evictions_total{app="groupcache-exporter-mailgun",group="files",type="main"} 1
groupcache_cache_gets_total{app="groupcache-exporter-mailgun",group="files",type="hot"} 4
groupcache_cache_gets_total{app="groupcache-exporter-mailgun",group="files",type="main"} 16
groupcache_cache_hits_total{app="groupcache-exporter-mailgun",group="files",type="hot"} 0
groupcache_cache_hits_total{app="groupcache-exporter-mailgun",group="files",type="main"} 12
groupcache_cache_items{app="groupcache-exporter-mailgun",group="files",type="hot"} 0
groupcache_cache_items{app="groupcache-exporter-mailgun",group="files",type="main"} 1
groupcache_gets_total{app="groupcache-exporter-mailgun",group="files"} 14
groupcache_hits_total{app="groupcache-exporter-mailgun",group="files"} 12
groupcache_loads_deduped_total{app="groupcache-exporter-mailgun",group="files"} 2
groupcache_loads_total{app="groupcache-exporter-mailgun",group="files"} 2
groupcache_local_load_errs_total{app="groupcache-exporter-mailgun",group="files"} 0
groupcache_local_load_total{app="groupcache-exporter-mailgun",group="files"} 2
groupcache_peer_errors_total{app="groupcache-exporter-mailgun",group="files"} 0
groupcache_peer_loads_total{app="groupcache-exporter-mailgun",group="files"} 0
groupcache_server_requests_total{app="groupcache-exporter-mailgun",group="files"} 0
```

# Prometheus Groupcache Exporter

This exporter extracts statistics from groupcache instances and exports Prometheus metrics.

# Example

```go
import (
	"github.com/mailgun/groupcache/v2"
	"github.com/mailgun/groupcache/v2/promexporter"
)

// ...

collector := promexporter.NewExporter("", nil)
prometheus.MustRegister(collector)

// Collector will discover newly created group.
group := groupcache.NewGroup("mygroup", cacheSize, getter)
```

## Exported Metrics

- `groupcache_cache_bytes{group,type="main|hot"}`: Gauge of current bytes in use
- `groupcache_cache_evictions_nonexpired_total{group,type="main|hot"}`: Count of cache evictions for non-expired keys due to memory full
- `groupcache_cache_evictions_total{group,type="main|hot"}`: Count of cache evictions
- `groupcache_cache_gets_total{group,type="main|hot"}`: Count of cache gets
- `groupcache_cache_hits_total{group,type="main|hot"}`: Count of cache hits
- `groupcache_cache_items{group,type="main|hot"}`: Gauge of current items in use
- `groupcache_get_from_peers_latency_lower{group}`: Represent slowest duration to request value from peers
- `groupcache_gets_total{group}`: Count of cache gets (including from peers, from either main or hot caches)
- `groupcache_hits_total{group}`: Count of cache hits (from either main or hot caches)
- `groupcache_loads_deduped_total{group}`: Count of loads after singleflight
- `groupcache_loads_total{group}`: Count of (gets - hits)
- `groupcache_local_load_errs_total{group}`: Count of load errors from local cache
- `groupcache_local_loads_total{group}`: Count of loads from local cache
- `groupcache_peer_errors_total{group}`: Count of errors from peers
- `groupcache_peer_loads_total{group}`: Count of loads or cache hits from peers
- `groupcache_server_requests_total{group}`: Count of gets received from peers

# Attribution

This package source was copied from https://github.com/udhos/groupcache_exporter.  See LICENSE for MIT license details impacting the contents of this package directory in addition to the LICENSE at the root of this repo for co-existing Apache license details.

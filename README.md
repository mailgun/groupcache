[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/Baliedge/groupcache_exporter/blob/main/LICENSE)

# Prometheus Groupcache Exporter

This exporter extracts statistics from [groupcache](https://github.com/golang/groupcache) instances and converts to Prometheus metrics.

# Example

Full example: [examples/groupcache-exporter-mailgun](examples/groupcache-exporter-mailgun)

## Exported Metrics

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

// Package promexporter exports Prometheus metrics for groupcache.
package promexporter

import (
	"github.com/mailgun/groupcache/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// Exporter implements interface prometheus.Collector to extract metrics from groupcache.
type Exporter struct {
	namespace                     string
	labels                        map[string]string
	groupGets                     *prometheus.Desc
	groupHits                     *prometheus.Desc
	groupGetFromPeersLatencyLower *prometheus.Desc
	groupPeerLoads                *prometheus.Desc
	groupPeerErrors               *prometheus.Desc
	groupLoads                    *prometheus.Desc
	groupLoadsDeduped             *prometheus.Desc
	groupLocalLoads               *prometheus.Desc
	groupLocalLoadErrs            *prometheus.Desc
	groupServerRequests           *prometheus.Desc
	cacheBytes                    *prometheus.Desc
	cacheItems                    *prometheus.Desc
	cacheGets                     *prometheus.Desc
	cacheHits                     *prometheus.Desc
	cacheEvictions                *prometheus.Desc
	cacheEvictionsNonExpired      *prometheus.Desc
}

// GroupStatistics is a plugable interface to extract metrics from a groupcache implementation.
// GroupStatistics is used by Exporter to collect the group statistics.
// The user must provide a concrete implementation of this interface that knows how to
// extract group statistics from the actual groupcache implementation.
type GroupStatistics interface {
	// Name returns the group's name
	Name() string

	// Gets represents any Get request, including from peers
	Gets() int64

	// CacheHits represents either cache was good
	CacheHits() int64

	// GetFromPeersLatencyLower represents slowest duration to request value from peers
	GetFromPeersLatencyLower() float64

	// PeerLoads represents either remote load or remote cache hit (not an error)
	PeerLoads() int64

	// PeerErrors represents a count of errors from peers
	PeerErrors() int64

	// Loads represents (gets - cacheHits)
	Loads() int64

	// LoadsDeduped represents after singleflight
	LoadsDeduped() int64

	// LocalLoads represents total good local loads
	LocalLoads() int64

	// LocalLoadErrs represents total bad local loads
	LocalLoadErrs() int64

	// ServerRequests represents gets that came over the network from peers
	ServerRequests() int64

	MainCacheItems() int64
	MainCacheBytes() int64
	MainCacheGets() int64
	MainCacheHits() int64
	MainCacheEvictions() int64
	MainCacheEvictionsNonExpired() int64

	HotCacheItems() int64
	HotCacheBytes() int64
	HotCacheGets() int64
	HotCacheHits() int64
	HotCacheEvictions() int64
	HotCacheEvictionsNonExpired() int64
}

type Option interface {
	apply(*Exporter)
}

type namespaceOption struct {
	namespace string
}

type labelsOption struct {
	labels map[string]string
}

// NewExporter creates Exporter.
// namespace is usually the empty string.
func NewExporter(opts ...Option) *Exporter {
	const subsystem = "groupcache"

	e := new(Exporter)
	for _, opt := range opts {
		opt.apply(e)
	}
	e.groupGets = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "gets_total"),
		"Count of cache gets (including from peers, from either main or hot cache)",
		[]string{"group"},
		e.labels,
	)
	e.groupHits = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "hits_total"),
		"Count of cache hits (from either main or hot cache)",
		[]string{"group"},
		e.labels,
	)
	e.groupGetFromPeersLatencyLower = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "get_from_peers_latency_lower"),
		"Represent slowest duration to request value from peers",
		[]string{"group"},
		e.labels,
	)
	e.groupPeerLoads = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "peer_loads_total"),
		"Count of loads or cache hits from peers",
		[]string{"group"},
		e.labels,
	)
	e.groupPeerErrors = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "peer_errors_total"),
		"Count of errors from peers",
		[]string{"group"},
		e.labels,
	)
	e.groupLoads = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "loads_total"),
		"Count of (gets - hits)",
		[]string{"group"},
		e.labels,
	)
	e.groupLoadsDeduped = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "loads_deduped_total"),
		"Count of loads after singleflight",
		[]string{"group"},
		e.labels,
	)
	e.groupLocalLoads = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "local_loads_total"),
		"Count of loads from local cache",
		[]string{"group"},
		e.labels,
	)
	e.groupLocalLoadErrs = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "local_load_errs_total"),
		"Count of load errors from local cache",
		[]string{"group"},
		e.labels,
	)
	e.groupServerRequests = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "server_requests_total"),
		"Count of gets received from peers",
		[]string{"group"},
		e.labels,
	)
	e.cacheBytes = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "cache_bytes"),
		"Gauge of current bytes in use",
		[]string{"group", "type"},
		e.labels,
	)
	e.cacheItems = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "cache_items"),
		"Gauge of current items in use",
		[]string{"group", "type"},
		e.labels,
	)
	e.cacheGets = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "cache_gets_total"),
		"Count of cache gets",
		[]string{"group", "type"},
		e.labels,
	)
	e.cacheHits = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "cache_hits_total"),
		"Count of cache hits",
		[]string{"group", "type"},
		e.labels,
	)
	e.cacheEvictions = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "cache_evictions_total"),
		"Count of cache evictions",
		[]string{"group", "type"},
		e.labels,
	)
	e.cacheEvictionsNonExpired = prometheus.NewDesc(
		prometheus.BuildFQName(e.namespace, subsystem, "cache_evictions_nonexpired_total"),
		"Count of cache evictions for non-expired keys due to memory full",
		[]string{"group", "type"},
		e.labels,
	)
	return e
}

func (e *Exporter) getStatGroups() []GroupStatistics {
	groups := groupcache.GetGroups()
	statgroups := make([]GroupStatistics, 0, len(groups))
	for _, g := range groups {
		statgroups = append(statgroups, newStatsAdapter(g))
	}
	return statgroups
}

// Describe sends metrics descriptors.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.groupGets
	ch <- e.groupHits
	ch <- e.groupGetFromPeersLatencyLower
	ch <- e.groupPeerLoads
	ch <- e.groupPeerErrors
	ch <- e.groupLoads
	ch <- e.groupLoadsDeduped
	ch <- e.groupLocalLoads
	ch <- e.groupLocalLoadErrs
	ch <- e.groupServerRequests
	ch <- e.cacheBytes
	ch <- e.cacheItems
	ch <- e.cacheGets
	ch <- e.cacheHits
	ch <- e.cacheEvictions
	ch <- e.cacheEvictionsNonExpired
}

// Collect is called by the Prometheus registry when collecting metrics.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, group := range e.getStatGroups() {
		e.collectFromGroup(ch, group)
	}
}

func (e *Exporter) collectFromGroup(ch chan<- prometheus.Metric, stats GroupStatistics) {
	e.collectStats(ch, stats)
	e.collectCacheStats(ch, stats)
}

func (e *Exporter) collectStats(ch chan<- prometheus.Metric, stats GroupStatistics) {
	ch <- prometheus.MustNewConstMetric(e.groupGets, prometheus.CounterValue, float64(stats.Gets()), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupHits, prometheus.CounterValue, float64(stats.CacheHits()), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupGetFromPeersLatencyLower, prometheus.GaugeValue, stats.GetFromPeersLatencyLower(), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupPeerLoads, prometheus.CounterValue, float64(stats.PeerLoads()), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupPeerErrors, prometheus.CounterValue, float64(stats.PeerErrors()), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupLoads, prometheus.CounterValue, float64(stats.Loads()), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupLoadsDeduped, prometheus.CounterValue, float64(stats.LoadsDeduped()), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupLocalLoads, prometheus.CounterValue, float64(stats.LocalLoads()), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupLocalLoadErrs, prometheus.CounterValue, float64(stats.LocalLoadErrs()), stats.Name())
	ch <- prometheus.MustNewConstMetric(e.groupServerRequests, prometheus.CounterValue, float64(stats.ServerRequests()), stats.Name())
}

func (e *Exporter) collectCacheStats(ch chan<- prometheus.Metric, stats GroupStatistics) {
	ch <- prometheus.MustNewConstMetric(e.cacheItems, prometheus.GaugeValue, float64(stats.MainCacheItems()), stats.Name(), "main")
	ch <- prometheus.MustNewConstMetric(e.cacheBytes, prometheus.GaugeValue, float64(stats.MainCacheBytes()), stats.Name(), "main")
	ch <- prometheus.MustNewConstMetric(e.cacheGets, prometheus.CounterValue, float64(stats.MainCacheGets()), stats.Name(), "main")
	ch <- prometheus.MustNewConstMetric(e.cacheHits, prometheus.CounterValue, float64(stats.MainCacheHits()), stats.Name(), "main")
	ch <- prometheus.MustNewConstMetric(e.cacheEvictions, prometheus.CounterValue, float64(stats.MainCacheEvictions()), stats.Name(), "main")
	ch <- prometheus.MustNewConstMetric(e.cacheEvictionsNonExpired, prometheus.CounterValue, float64(stats.MainCacheEvictionsNonExpired()), stats.Name(), "main")

	ch <- prometheus.MustNewConstMetric(e.cacheItems, prometheus.GaugeValue, float64(stats.HotCacheItems()), stats.Name(), "hot")
	ch <- prometheus.MustNewConstMetric(e.cacheBytes, prometheus.GaugeValue, float64(stats.HotCacheBytes()), stats.Name(), "hot")
	ch <- prometheus.MustNewConstMetric(e.cacheGets, prometheus.CounterValue, float64(stats.HotCacheGets()), stats.Name(), "hot")
	ch <- prometheus.MustNewConstMetric(e.cacheHits, prometheus.CounterValue, float64(stats.HotCacheHits()), stats.Name(), "hot")
	ch <- prometheus.MustNewConstMetric(e.cacheEvictions, prometheus.CounterValue, float64(stats.HotCacheEvictions()), stats.Name(), "hot")
	ch <- prometheus.MustNewConstMetric(e.cacheEvictionsNonExpired, prometheus.CounterValue, float64(stats.HotCacheEvictionsNonExpired()), stats.Name(), "hot")
}

func WithNamspace(namespace string) Option {
	return &namespaceOption{namespace: namespace}
}

func (o *namespaceOption) apply(e *Exporter) {
	e.namespace = o.namespace
}

func WithLabels(labels map[string]string) Option {
	return &labelsOption{labels: labels}
}

func (o *labelsOption) apply(e *Exporter) {
	e.labels = o.labels
}

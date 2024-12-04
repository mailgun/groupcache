package groupcache

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	SummaryObjectives = map[float64]float64{
		0.5:  0.05,
		0.99: 0.001,
		1:    0.001,
	}
	metricGetFromPeerLatency = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "groupcache_get_from_peer_latency",
		Help:       "The latency in seconds getting value from remote peer",
		Objectives: SummaryObjectives,
	}, []string{"group", "peer"})
	metricUpdatePeerLatency = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "groupcache_update_peer_latency",
		Help:       "The latency in seconds updating a remote peer during a Set",
		Objectives: SummaryObjectives,
	}, []string{"group", "peer"})
)

// GetMetrics about Groupcache.
func GetMetrics() []prometheus.Collector {
	return []prometheus.Collector{
		metricGetFromPeerLatency,
		metricUpdatePeerLatency,
	}
}

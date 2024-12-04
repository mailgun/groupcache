package groupcache

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricPeerUpdateLatency = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "groupcache_peer_update_latency",
		Help: "The latency in seconds during peer update after a Set",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.95: 0.01,
			0.99: 0.001,
			1:    0.001,
		},
	}, []string{"group", "peer"})
)

// GetMetrics about Groupcache.
func GetMetrics() []prometheus.Collector {
	return []prometheus.Collector{
		metricPeerUpdateLatency,
	}
}

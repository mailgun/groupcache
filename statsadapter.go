package groupcache_exporter

import (
	"github.com/mailgun/groupcache/v2"
)

// Group implements interface GroupStatistics to extract metrics from groupcache group.
type statsAdapter struct {
	group *groupcache.Group
}

// New creates a new Group.
func newStatsAdapter(group *groupcache.Group) *statsAdapter {
	return &statsAdapter{group: group}
}

// Name returns the group's name
func (g *statsAdapter) Name() string {
	return g.group.Name()
}

// Gets represents any Get request, including from peers
func (g *statsAdapter) Gets() int64 {
	return g.group.Stats.Gets.Get()
}

// CacheHits represents either cache was good
func (g *statsAdapter) CacheHits() int64 {
	return g.group.Stats.CacheHits.Get()
}

// GetFromPeersLatencyLower represents slowest duration to request value from peers
func (g *statsAdapter) GetFromPeersLatencyLower() float64 {
	latencyMs := g.group.Stats.GetFromPeersLatencyLower.Get()
	if latencyMs == 0 {
		return 0
	}
	return float64(latencyMs) / 1000
}

// PeerLoads represents either remote load or remote cache hit (not an error)
func (g *statsAdapter) PeerLoads() int64 {
	return g.group.Stats.PeerLoads.Get()
}

// PeerErrors represents a count of errors from peers
func (g *statsAdapter) PeerErrors() int64 {
	return g.group.Stats.PeerErrors.Get()
}

// Loads represents (gets - cacheHits)
func (g *statsAdapter) Loads() int64 {
	return g.group.Stats.Loads.Get()
}

// LoadsDeduped represents after singleflight
func (g *statsAdapter) LoadsDeduped() int64 {
	return g.group.Stats.LoadsDeduped.Get()
}

// LocalLoads represents total good local loads
func (g *statsAdapter) LocalLoads() int64 {
	return g.group.Stats.LocalLoads.Get()
}

// LocalLoadErrs represents total bad local loads
func (g *statsAdapter) LocalLoadErrs() int64 {
	return g.group.Stats.LocalLoadErrs.Get()
}

// ServerRequests represents gets that came over the network from peers
func (g *statsAdapter) ServerRequests() int64 {
	return g.group.Stats.ServerRequests.Get()
}

// MainCacheItems represents number of items in the main cache
func (g *statsAdapter) MainCacheItems() int64 {
	return g.group.CacheStats(groupcache.MainCache).Items
}

// MainCacheBytes represents number of bytes in the main cache
func (g *statsAdapter) MainCacheBytes() int64 {
	return g.group.CacheStats(groupcache.MainCache).Bytes
}

// MainCacheGets represents number of get requests in the main cache
func (g *statsAdapter) MainCacheGets() int64 {
	return g.group.CacheStats(groupcache.MainCache).Gets
}

// MainCacheHits represents number of hit in the main cache
func (g *statsAdapter) MainCacheHits() int64 {
	return g.group.CacheStats(groupcache.MainCache).Hits
}

// MainCacheEvictions represents number of evictions in the main cache
func (g *statsAdapter) MainCacheEvictions() int64 {
	return g.group.CacheStats(groupcache.MainCache).Evictions
}

// MainCacheEvictionsNonExpired represents number of evictions for non-expired keys in the main cache
func (g *statsAdapter) MainCacheEvictionsNonExpired() int64 {
	return 0
}

// HotCacheItems represents number of items in the main cache
func (g *statsAdapter) HotCacheItems() int64 {
	return g.group.CacheStats(groupcache.HotCache).Items
}

// HotCacheBytes represents number of bytes in the hot cache
func (g *statsAdapter) HotCacheBytes() int64 {
	return g.group.CacheStats(groupcache.HotCache).Bytes
}

// HotCacheGets represents number of get requests in the hot cache
func (g *statsAdapter) HotCacheGets() int64 {
	return g.group.CacheStats(groupcache.HotCache).Gets
}

// HotCacheHits represents number of hit in the hot cache
func (g *statsAdapter) HotCacheHits() int64 {
	return g.group.CacheStats(groupcache.HotCache).Hits
}

// HotCacheEvictions represents number of evictions in the hot cache
func (g *statsAdapter) HotCacheEvictions() int64 {
	return g.group.CacheStats(groupcache.HotCache).Evictions
}

// HotCacheEvictionsNonExpired represents number of evictions for non-expired keys in the hot cache
func (g *statsAdapter) HotCacheEvictionsNonExpired() int64 {
	return 0
}

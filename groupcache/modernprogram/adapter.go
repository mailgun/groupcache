// Package modernprogram implements an adapter to extract metrics from modernprogram groupcache.
package modernprogram

import (
	"github.com/modernprogram/groupcache/v2"
)

// GroupMailgun implements interface GroupStatistics to extract metrics from mailgun groupcache group.
type GroupMailgun struct {
	group *groupcache.Group
}

// New creates a new GroupMailgun.
func New(group *groupcache.Group) *GroupMailgun {
	return &GroupMailgun{group: group}
}

// Name returns the group's name
func (g *GroupMailgun) Name() string {
	return g.group.Name()
}

// Gets represents any Get request, including from peers
func (g *GroupMailgun) Gets() int64 {
	return g.group.Stats.Gets.Get()
}

// CacheHits represents either cache was good
func (g *GroupMailgun) CacheHits() int64 {
	return g.group.Stats.CacheHits.Get()
}

// GetFromPeersLatencyLower represents slowest duration to request value from peers
func (g *GroupMailgun) GetFromPeersLatencyLower() int64 {
	return g.group.Stats.GetFromPeersLatencyLower.Get()
}

// PeerLoads represents either remote load or remote cache hit (not an error)
func (g *GroupMailgun) PeerLoads() int64 {
	return g.group.Stats.PeerLoads.Get()
}

// PeerErrors represents a count of errors from peers
func (g *GroupMailgun) PeerErrors() int64 {
	return g.group.Stats.PeerErrors.Get()
}

// Loads represents (gets - cacheHits)
func (g *GroupMailgun) Loads() int64 {
	return g.group.Stats.Loads.Get()
}

// LoadsDeduped represents after singleflight
func (g *GroupMailgun) LoadsDeduped() int64 {
	return g.group.Stats.LoadsDeduped.Get()
}

// LocalLoads represents total good local loads
func (g *GroupMailgun) LocalLoads() int64 {
	return g.group.Stats.LocalLoads.Get()
}

// LocalLoadErrs represents total bad local loads
func (g *GroupMailgun) LocalLoadErrs() int64 {
	return g.group.Stats.LocalLoadErrs.Get()
}

// ServerRequests represents gets that came over the network from peers
func (g *GroupMailgun) ServerRequests() int64 {
	return g.group.Stats.ServerRequests.Get()
}

// MainCacheItems represents number of items in the main cache
func (g *GroupMailgun) MainCacheItems() int64 {
	return g.group.CacheStats(groupcache.MainCache).Items
}

// MainCacheBytes represents number of bytes in the main cache
func (g *GroupMailgun) MainCacheBytes() int64 {
	return g.group.CacheStats(groupcache.MainCache).Bytes
}

// MainCacheGets represents number of get requests in the main cache
func (g *GroupMailgun) MainCacheGets() int64 {
	return g.group.CacheStats(groupcache.MainCache).Gets
}

// MainCacheHits represents number of hit in the main cache
func (g *GroupMailgun) MainCacheHits() int64 {
	return g.group.CacheStats(groupcache.MainCache).Hits
}

// MainCacheEvictions represents number of evictions in the main cache
func (g *GroupMailgun) MainCacheEvictions() int64 {
	return g.group.CacheStats(groupcache.MainCache).Evictions
}

// HotCacheItems represents number of items in the main cache
func (g *GroupMailgun) HotCacheItems() int64 {
	return g.group.CacheStats(groupcache.HotCache).Items
}

// HotCacheBytes represents number of bytes in the hot cache
func (g *GroupMailgun) HotCacheBytes() int64 {
	return g.group.CacheStats(groupcache.HotCache).Bytes
}

// HotCacheGets represents number of get requests in the hot cache
func (g *GroupMailgun) HotCacheGets() int64 {
	return g.group.CacheStats(groupcache.HotCache).Gets
}

// HotCacheHits represents number of hit in the hot cache
func (g *GroupMailgun) HotCacheHits() int64 {
	return g.group.CacheStats(groupcache.HotCache).Hits
}

// HotCacheEvictions represents number of evictions in the hot cache
func (g *GroupMailgun) HotCacheEvictions() int64 {
	return g.group.CacheStats(groupcache.HotCache).Evictions
}

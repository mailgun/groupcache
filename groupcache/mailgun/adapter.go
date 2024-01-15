// Package mailgun implements an adapter to extract metrics from mailgun groupcache.
package mailgun

import (
	"github.com/mailgun/groupcache/v2"
)

// Group implements interface GroupStatistics to extract metrics from mailgun groupcache group.
type Group struct {
	group *groupcache.Group
}

// New creates a new GroupMailgun.
func New(group *groupcache.Group) *Group {
	return &Group{group: group}
}

// Name returns the group's name
func (g *Group) Name() string {
	return g.group.Name()
}

// Gets represents any Get request, including from peers
func (g *Group) Gets() int64 {
	return g.group.Stats.Gets.Get()
}

// CacheHits represents either cache was good
func (g *Group) CacheHits() int64 {
	return g.group.Stats.CacheHits.Get()
}

// GetFromPeersLatencyLower represents slowest duration to request value from peers
func (g *Group) GetFromPeersLatencyLower() int64 {
	return g.group.Stats.GetFromPeersLatencyLower.Get()
}

// PeerLoads represents either remote load or remote cache hit (not an error)
func (g *Group) PeerLoads() int64 {
	return g.group.Stats.PeerLoads.Get()
}

// PeerErrors represents a count of errors from peers
func (g *Group) PeerErrors() int64 {
	return g.group.Stats.PeerErrors.Get()
}

// Loads represents (gets - cacheHits)
func (g *Group) Loads() int64 {
	return g.group.Stats.Loads.Get()
}

// LoadsDeduped represents after singleflight
func (g *Group) LoadsDeduped() int64 {
	return g.group.Stats.LoadsDeduped.Get()
}

// LocalLoads represents total good local loads
func (g *Group) LocalLoads() int64 {
	return g.group.Stats.LocalLoads.Get()
}

// LocalLoadErrs represents total bad local loads
func (g *Group) LocalLoadErrs() int64 {
	return g.group.Stats.LocalLoadErrs.Get()
}

// ServerRequests represents gets that came over the network from peers
func (g *Group) ServerRequests() int64 {
	return g.group.Stats.ServerRequests.Get()
}

// MainCacheItems represents number of items in the main cache
func (g *Group) MainCacheItems() int64 {
	return g.group.CacheStats(groupcache.MainCache).Items
}

// MainCacheBytes represents number of bytes in the main cache
func (g *Group) MainCacheBytes() int64 {
	return g.group.CacheStats(groupcache.MainCache).Bytes
}

// MainCacheGets represents number of get requests in the main cache
func (g *Group) MainCacheGets() int64 {
	return g.group.CacheStats(groupcache.MainCache).Gets
}

// MainCacheHits represents number of hit in the main cache
func (g *Group) MainCacheHits() int64 {
	return g.group.CacheStats(groupcache.MainCache).Hits
}

// MainCacheEvictions represents number of evictions in the main cache
func (g *Group) MainCacheEvictions() int64 {
	return g.group.CacheStats(groupcache.MainCache).Evictions
}

// HotCacheItems represents number of items in the main cache
func (g *Group) HotCacheItems() int64 {
	return g.group.CacheStats(groupcache.HotCache).Items
}

// HotCacheBytes represents number of bytes in the hot cache
func (g *Group) HotCacheBytes() int64 {
	return g.group.CacheStats(groupcache.HotCache).Bytes
}

// HotCacheGets represents number of get requests in the hot cache
func (g *Group) HotCacheGets() int64 {
	return g.group.CacheStats(groupcache.HotCache).Gets
}

// HotCacheHits represents number of hit in the hot cache
func (g *Group) HotCacheHits() int64 {
	return g.group.CacheStats(groupcache.HotCache).Hits
}

// HotCacheEvictions represents number of evictions in the hot cache
func (g *Group) HotCacheEvictions() int64 {
	return g.group.CacheStats(groupcache.HotCache).Evictions
}

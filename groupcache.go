/*
Copyright 2012 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package groupcache provides a data loading mechanism with caching
// and de-duplication that works across a set of peer processes.
//
// Each data Get first consults its local cache, otherwise delegates
// to the requested key's canonical owner, which then checks its cache
// or finally gets the data.  In the common case, many concurrent
// cache misses across a set of peers for the same key result in just
// one cache fill.
package groupcache

import (
	"context"
	"errors"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/mailgun/groupcache/v2/groupcachepb"
	"github.com/mailgun/groupcache/v2/lru"
	"github.com/mailgun/groupcache/v2/singleflight"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Entry

func SetLogger(log *logrus.Entry) {
	logger = log
}

// A Getter loads data for a key.
type Getter interface {
	// Get returns the value identified by key, populating dest.
	//
	// The returned data must be unversioned. That is, key must
	// uniquely describe the loaded data, without an implicit
	// current time, and without relying on cache expiration
	// mechanisms.
	Get(ctx context.Context, key string, dest Sink) error
}

// A Getter loads data for a keys.
type BatchGetter interface {
	// Get returns the value identified by key, populating dest.
	//
	// The returned data must be unversioned. That is, key must
	// uniquely describe the loaded data, without an implicit
	// current time, and without relying on cache expiration
	// mechanisms.
	BatchGet(ctx context.Context, keyList []string, destList []Sink) []error
}

type GroupGetter interface {
	Getter
	BatchGetter
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(ctx context.Context, key string, dest Sink) error

func (f GetterFunc) Get(ctx context.Context, key string, dest Sink) error {
	return f(ctx, key, dest)
}

func (f GetterFunc) BatchGet(ctx context.Context, keyList []string, destList []Sink) []error {
	//panic("implement me")
	return nil
}

// A BatchGetterFunc implements BatchGetter with a function.
type BatchGetterFunc func(ctx context.Context, key []string, dest []Sink) []error

func (f BatchGetterFunc) Get(ctx context.Context, key string, dest Sink) error {
	keyList := []string{key}
	destList := []Sink{dest}
	errList := f(ctx, keyList, destList)
	if len(errList) > 0 {
		return errList[0]
	}
	if len(destList) > 0 {
		dest = destList[0]
	}
	return nil
}

func (f BatchGetterFunc) BatchGet(ctx context.Context, keyList []string, destList []Sink) []error {
	return f(ctx, keyList, destList)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)

	initPeerServerOnce sync.Once
	initPeerServer     func()
)

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// NewGroup creates a coordinated group-aware Getter from a Getter.
//
// The returned Getter tries (but does not guarantee) to run only one
// Get call at once for a given key across an entire set of peer
// processes. Concurrent callers both in the local process and in
// other processes receive copies of the answer once the original Get
// completes.
//
// The group name must be unique for each getter.
func NewGroup(name string, cacheBytes int64, getter GroupGetter) *Group {
	return newGroup(name, cacheBytes, getter, nil)
}

// DeregisterGroup removes group from group pool
func DeregisterGroup(name string) {
	mu.Lock()
	delete(groups, name)
	mu.Unlock()
}

// If peers is nil, the peerPicker is called via a sync.Once to initialize it.
func newGroup(name string, cacheBytes int64, getter GroupGetter, peers PeerPicker) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	initPeerServerOnce.Do(callInitPeerServer)
	if _, dup := groups[name]; dup {
		panic("duplicate registration of group " + name)
	}
	g := &Group{
		name:        name,
		getter:      getter,
		peers:       peers,
		cacheBytes:  cacheBytes,
		loadGroup:   &singleflight.Group{},
		removeGroup: &singleflight.Group{},
	}
	if fn := newGroupHook; fn != nil {
		fn(g)
	}
	groups[name] = g
	return g
}

// newGroupHook, if non-nil, is called right after a new group is created.
var newGroupHook func(*Group)

// RegisterNewGroupHook registers a hook that is run each time
// a group is created.
func RegisterNewGroupHook(fn func(*Group)) {
	if newGroupHook != nil {
		panic("RegisterNewGroupHook called more than once")
	}
	newGroupHook = fn
}

// RegisterServerStart registers a hook that is run when the first
// group is created.
func RegisterServerStart(fn func()) {
	if initPeerServer != nil {
		panic("RegisterServerStart called more than once")
	}
	initPeerServer = fn
}

func callInitPeerServer() {
	if initPeerServer != nil {
		initPeerServer()
	}
}

// A Group is a cache namespace and associated data loaded spread over
// a group of 1 or more machines.
type Group struct {
	name       string
	getter     GroupGetter
	peersOnce  sync.Once
	peers      PeerPicker
	cacheBytes int64 // limit for sum of mainCache and hotCache size

	// mainCache is a cache of the keys for which this process
	// (amongst its peers) is authoritative. That is, this cache
	// contains keys which consistent hash on to this process's
	// peer number.
	mainCache cache

	// hotCache contains keys/values for which this peer is not
	// authoritative (otherwise they would be in mainCache), but
	// are popular enough to warrant mirroring in this process to
	// avoid going over the network to fetch from a peer.  Having
	// a hotCache avoids network hotspotting, where a peer's
	// network card could become the bottleneck on a popular key.
	// This cache is used sparingly to maximize the total number
	// of key/value pairs that can be stored globally.
	hotCache cache

	// loadGroup ensures that each key is only fetched once
	// (either locally or remotely), regardless of the number of
	// concurrent callers.
	loadGroup flightGroup

	// removeGroup ensures that each removed key is only removed
	// remotely once regardless of the number of concurrent callers.
	removeGroup flightGroup

	_ int32 // force Stats to be 8-byte aligned on 32-bit platforms

	// Stats are statistics on the group.
	Stats Stats
}

// flightGroup is defined as an interface which flightgroup.Group
// satisfies.  We define this so that we may test with an alternate
// implementation.
type flightGroup interface {
	Do(key string, fn func() (interface{}, error)) (interface{}, error)
	Lock(fn func())
}

// Stats are per-group statistics.
type Stats struct {
	Gets                     AtomicInt // any Get request, including from peers
	CacheHits                AtomicInt // either cache was good
	GetFromPeersLatencyLower AtomicInt // slowest duration to request value from peers
	PeerLoads                AtomicInt // either remote load or remote cache hit (not an error)
	PeerErrors               AtomicInt
	Loads                    AtomicInt // (gets - cacheHits)
	LoadsDeduped             AtomicInt // after singleflight
	LocalLoads               AtomicInt // total good local loads
	LocalLoadErrs            AtomicInt // total bad local loads
	ServerRequests           AtomicInt // gets that came over the network from peers
}

// Name returns the name of the group.
func (g *Group) Name() string {
	return g.name
}

func (g *Group) initPeers() {
	if g.peers == nil {
		g.peers = getPeers(g.name)
	}
}

func (g *Group) Get(ctx context.Context, key string, dest Sink) error {
	g.peersOnce.Do(g.initPeers)
	g.Stats.Gets.Add(1)
	if dest == nil {
		return errors.New("groupcache: nil dest Sink")
	}
	value, cacheHit := g.lookupCache(key)

	if cacheHit {
		g.Stats.CacheHits.Add(1)
		return setSinkView(dest, value)
	}

	// Optimization to avoid double unmarshalling or copying: keep
	// track of whether the dest was already populated. One caller
	// (if local) will set this; the losers will not. The common
	// case will likely be one caller.
	destPopulated := false
	value, destPopulated, err := g.load(ctx, key, dest)
	if err != nil {
		return err
	}
	if destPopulated {
		return nil
	}
	return setSinkView(dest, value)
}

func (g *Group) BatchGet(ctx context.Context, keyList []string, destList []Sink) []error {
	g.peersOnce.Do(g.initPeers)
	errList := make([]error, 0)
	if len(keyList) == 0 {
		err := errors.New("groupcache: nil key list")
		errList = append(errList, err)
		return errList
	}

	g.Stats.Gets.Add(int64(len(keyList)))
	lookUpCachevalues, missKeyList := g.lookupCacheByKeyList(keyList)
	g.Stats.CacheHits.Add(int64(len(keyList) - len(missKeyList)))

	batchLoadValues, err := g.batchLoad(ctx, missKeyList, destList)
	if err != nil {
		errList = append(errList, err)
		return errList
	}

	// sort return data by input keys
	for index, key := range keyList {
		if value, ok := lookUpCachevalues[key]; ok {
			err := setSinkView(destList[index], value)
			if err != nil {
				errList = append(errList, err)
				continue
			}
		} else if value, ok := batchLoadValues[key]; ok {
			err := setSinkView(destList[index], value)
			if err != nil {
				errList = append(errList, err)
				continue
			}
		}
	}
	return errList
}

// Remove clears the key from our cache then forwards the remove
// request to all peers.
func (g *Group) Remove(ctx context.Context, key string) error {
	g.peersOnce.Do(g.initPeers)

	_, err := g.removeGroup.Do(key, func() (interface{}, error) {

		// Remove from key owner first
		owner, ok := g.peers.PickPeer(key)
		if ok {
			if err := g.removeFromPeer(ctx, owner, key); err != nil {
				return nil, err
			}
		}
		// Remove from our cache next
		g.localRemove(key)
		wg := sync.WaitGroup{}
		errs := make(chan error)

		// Asynchronously clear the key from all hot and main caches of peers
		for _, peer := range g.peers.GetAll() {
			// avoid deleting from owner a second time
			if peer == owner {
				continue
			}

			wg.Add(1)
			go func(peer ProtoGetter) {
				errs <- g.removeFromPeer(ctx, peer, key)
				wg.Done()
			}(peer)
		}
		go func() {
			wg.Wait()
			close(errs)
		}()

		// TODO(thrawn01): Should we report all errors? Reporting context
		//  cancelled error for each peer doesn't make much sense.
		var err error
		for e := range errs {
			err = e
		}

		return nil, err
	})
	return err
}

// load loads key either by invoking the getter locally or by sending it to another machine.
func (g *Group) load(ctx context.Context, key string, dest Sink) (value ByteView, destPopulated bool, err error) {
	g.Stats.Loads.Add(1)
	viewi, err := g.loadGroup.Do(key, func() (interface{}, error) {
		// Check the cache again because singleflight can only dedup calls
		// that overlap concurrently.  It's possible for 2 concurrent
		// requests to miss the cache, resulting in 2 load() calls.  An
		// unfortunate goroutine scheduling would result in this callback
		// being run twice, serially.  If we don't check the cache again,
		// cache.nbytes would be incremented below even though there will
		// be only one entry for this key.
		//
		// Consider the following serialized event ordering for two
		// goroutines in which this callback gets called twice for hte
		// same key:
		// 1: Get("key")
		// 2: Get("key")
		// 1: lookupCache("key")
		// 2: lookupCache("key")
		// 1: load("key")
		// 2: load("key")
		// 1: loadGroup.Do("key", fn)
		// 1: fn()
		// 2: loadGroup.Do("key", fn)
		// 2: fn()
		if value, cacheHit := g.lookupCache(key); cacheHit {
			g.Stats.CacheHits.Add(1)
			return value, nil
		}
		g.Stats.LoadsDeduped.Add(1)
		var value ByteView
		var err error
		if peer, ok := g.peers.PickPeer(key); ok {

			// metrics duration start
			start := time.Now()

			// get value from peers
			value, err = g.getFromPeer(ctx, peer, key)

			// metrics duration compute
			duration := int64(time.Since(start)) / int64(time.Millisecond)

			// metrics only store the slowest duration
			if g.Stats.GetFromPeersLatencyLower.Get() < duration {
				g.Stats.GetFromPeersLatencyLower.Store(duration)
			}

			if err == nil {
				g.Stats.PeerLoads.Add(1)
				return value, nil
			}

			if logger != nil {
				logger.WithFields(logrus.Fields{
					"err":      err,
					"key":      key,
					"category": "groupcache",
				}).Errorf("error retrieving key from peer '%s'", peer.GetURL())
			}

			g.Stats.PeerErrors.Add(1)
			if ctx != nil && ctx.Err() != nil {
				// Return here without attempting to get locally
				// since the context is no longer valid
				return nil, err
			}
			// TODO(bradfitz): log the peer's error? keep
			// log of the past few for /groupcachez?  It's
			// probably boring (normal task movement), so not
			// worth logging I imagine.
		}

		value, err = g.getLocally(ctx, key, dest)
		if err != nil {
			g.Stats.LocalLoadErrs.Add(1)
			return nil, err
		}
		g.Stats.LocalLoads.Add(1)
		destPopulated = true // only one caller of load gets this return value
		g.populateCache(key, value, &g.mainCache)
		return value, nil
	})
	if err == nil {
		value = viewi.(ByteView)
	}
	return
}

func (g *Group) batchLoad(ctx context.Context, keyList []string, destList []Sink) (values map[string]ByteView, err error) {
	if len(keyList) == 0 {
		return
	}

	peersMap := make(map[ProtoGetter][]string)
	locallyKeyList := make([]string, 0)
	for _, key := range keyList {
		if peer, ok := g.peers.PickPeer(key); ok {
			if peerKeyList, ok := peersMap[peer]; ok {
				peerKeyList = append(peerKeyList, key)
				peersMap[peer] = peerKeyList
			} else {
				peerKeyList := make([]string, 0)
				peerKeyList = append(peerKeyList, key)
				peersMap[peer] = peerKeyList
			}
		} else {
			locallyKeyList = append(locallyKeyList, key)
		}
	}

	syncMap := struct {
		sync.RWMutex
		values map[string]ByteView
	}{values: make(map[string]ByteView)}
	var waitGroup sync.WaitGroup
	if len(locallyKeyList) > 0 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			valueList, errList := g.batchGetLocally(ctx, locallyKeyList, destList)
			if len(errList) > 0 {
				g.Stats.LocalLoadErrs.Add(int64(len(errList)))
			}
			for index, key := range locallyKeyList {
				syncMap.Lock()
				syncMap.values[key] = valueList[index]
				g.populateCache(key, valueList[index], &g.mainCache)
				syncMap.Unlock()
			}
		}()
	}

	p := 50
	c := make(chan func(), p)
	go func() {
		for peerGetter, keyList := range peersMap {
			waitGroup.Add(1)
			newPeerGetter := peerGetter
			newKeyList := keyList
			c <- func() {
				defer waitGroup.Done()
				// metrics duration start
				start := time.Now()
				values, err := g.getFromPeerByKeyList(ctx, newPeerGetter, newKeyList)
				duration := int64(time.Since(start)) / int64(time.Millisecond)

				// metrics only store the slowest duration
				if g.Stats.GetFromPeersLatencyLower.Get() < duration {
					g.Stats.GetFromPeersLatencyLower.Store(duration)
				}

				if err == nil {
					g.Stats.PeerLoads.Add(1)
					syncMap.Lock()
					for index, value := range values {
						syncMap.values[newKeyList[index]] = value
					}
					syncMap.Unlock()
					return
				}

				if logger != nil {
					logger.WithFields(logrus.Fields{
						"err":      err,
						"keyList":  keyList,
						"category": "groupcache",
					}).Errorf("error retrieving key from peer '%s'", newPeerGetter.GetURL())
				}
				g.Stats.PeerErrors.Add(1)

				// if request peer is failing, then get data from local
				newDestList := make([]Sink, len(newKeyList))
				valueList, errList := g.batchGetLocally(ctx, newKeyList, newDestList)
				if len(errList) > 0 {
					g.Stats.LocalLoadErrs.Add(int64(len(errList)))
				}
				for index, key := range newKeyList {
					syncMap.Lock()
					syncMap.values[key] = valueList[index]
					g.populateCache(key, valueList[index], &g.mainCache)
					syncMap.Unlock()
				}
			}
			close(c)
		}
	}()
	parallelDo(ctx, p, c)
	return syncMap.values, err
}

func parallelDo(ctx context.Context, p int, c <-chan func()) {
	wg := sync.WaitGroup{}
	for i := 0; i < p; i++ {
		wg.Add(1)
		go recoverFuncWithWg(func() {
			for f := range c {
				f()
			}
		}, ctx, &wg)
	}
	wg.Wait()
}

func recoverFuncWithWg(f func(), ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			logger.Error(ctx, "_panic", "panic", p, "stack", string(debug.Stack()))
			wg.Done()
		} else {
			wg.Done()
		}
	}()
	f()
}
func (g *Group) getLocally(ctx context.Context, key string, dest Sink) (ByteView, error) {
	err := g.getter.Get(ctx, key, dest)
	if err != nil {
		return ByteView{}, err
	}
	return dest.view()
}

func (g *Group) batchGetLocally(ctx context.Context, keyList []string, destList []Sink) (byteViewList []ByteView, errList []error) {
	errList = g.getter.BatchGet(ctx, keyList, destList)
	if len(errList) > 0 {
		return byteViewList, errList
	}

	byteViewList = make([]ByteView, len(destList))
	for index, dest := range destList {
		view, err := dest.view()
		if err != nil {
			byteViewList[index] = ByteView{}
		} else {
			byteViewList[index] = view
		}
	}
	return byteViewList, nil
}

func (g *Group) getFromPeer(ctx context.Context, peer ProtoGetter, key string) (ByteView, error) {
	req := &pb.GetRequest{
		Group: &g.name,
		Key:   &key,
	}
	res := &pb.GetResponse{}
	err := peer.Get(ctx, req, res)
	if err != nil {
		return ByteView{}, err
	}

	var expire time.Time
	if res.Expire != nil && *res.Expire != 0 {
		expire = time.Unix(*res.Expire/int64(time.Second), *res.Expire%int64(time.Second))
		if time.Now().After(expire) {
			return ByteView{}, errors.New("peer returned expired value")
		}
	}

	value := ByteView{b: res.Value, e: expire}

	// Always populate the hot cache
	g.populateCache(key, value, &g.hotCache)
	return value, nil
}

func (g *Group) getFromPeerByKeyList(ctx context.Context, peer ProtoGetter, keyList []string) ([]ByteView, error) {
	req := &pb.GetListRequest{
		Group:   &g.name,
		KeyList: keyList,
	}
	res := &pb.GetListResponse{}
	err := peer.BatchGet(ctx, req, res)
	if err != nil {
		return []ByteView{}, err
	}

	resultByteView := make([]ByteView, 0, len(keyList))
	for index, kv := range res.KvList {
		var expire time.Time
		if kv.Expire != nil && *kv.Expire != 0 {
			expire = time.Unix(*kv.Expire/int64(time.Second), *kv.Expire%int64(time.Second))
			if time.Now().After(expire) {
				resultByteView = append(resultByteView, ByteView{})
				continue
			}
		}

		value := ByteView{b: kv.Value, e: expire}
		g.populateCache(keyList[index], value, &g.hotCache)
		resultByteView = append(resultByteView, value)
	}
	return resultByteView, nil
}

func (g *Group) removeFromPeer(ctx context.Context, peer ProtoGetter, key string) error {
	req := &pb.GetRequest{
		Group: &g.name,
		Key:   &key,
	}
	return peer.Remove(ctx, req)
}

func (g *Group) lookupCache(key string) (value ByteView, ok bool) {
	if g.cacheBytes <= 0 {
		return
	}
	value, ok = g.mainCache.get(key)
	if ok {
		return
	}
	value, ok = g.hotCache.get(key)
	return
}

func (g *Group) lookupCacheByKeyList(keyList []string) (values map[string]ByteView, missKey []string) {
	if g.cacheBytes <= 0 {
		return values, keyList
	}
	missKey = make([]string, 0)
	values = make(map[string]ByteView)
	for _, key := range keyList {
		value, ok := g.mainCache.get(key)
		if ok {
			values[key] = value
			continue
		}
		value, ok = g.hotCache.get(key)
		if ok {
			values[key] = value
			continue
		}
		missKey = append(missKey, key)
	}
	return
}

func (g *Group) localRemove(key string) {
	// Clear key from our local cache
	if g.cacheBytes <= 0 {
		return
	}

	// Ensure no requests are in flight
	g.loadGroup.Lock(func() {
		g.hotCache.remove(key)
		g.mainCache.remove(key)
	})
}

func (g *Group) populateCache(key string, value ByteView, cache *cache) {
	if g.cacheBytes <= 0 {
		return
	}
	cache.add(key, value)

	// Evict items from cache(s) if necessary.
	for {
		mainBytes := g.mainCache.bytes()
		hotBytes := g.hotCache.bytes()
		if mainBytes+hotBytes <= g.cacheBytes {
			return
		}

		// TODO(bradfitz): this is good-enough-for-now logic.
		// It should be something based on measurements and/or
		// respecting the costs of different resources.
		victim := &g.mainCache
		if hotBytes > mainBytes/8 {
			victim = &g.hotCache
		}
		victim.removeOldest()
	}
}

// CacheType represents a type of cache.
type CacheType int

const (
	// The MainCache is the cache for items that this peer is the
	// owner for.
	MainCache CacheType = iota + 1

	// The HotCache is the cache for items that seem popular
	// enough to replicate to this node, even though it's not the
	// owner.
	HotCache
)

// CacheStats returns stats about the provided cache within the group.
func (g *Group) CacheStats(which CacheType) CacheStats {
	switch which {
	case MainCache:
		return g.mainCache.stats()
	case HotCache:
		return g.hotCache.stats()
	default:
		return CacheStats{}
	}
}

// cache is a wrapper around an *lru.Cache that adds synchronization,
// makes values always be ByteView, and counts the size of all keys and
// values.
type cache struct {
	mu         sync.RWMutex
	nbytes     int64 // of all keys and values
	lru        *lru.Cache
	nhit, nget int64
	nevict     int64 // number of evictions
}

func (c *cache) stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return CacheStats{
		Bytes:     c.nbytes,
		Items:     c.itemsLocked(),
		Gets:      c.nget,
		Hits:      c.nhit,
		Evictions: c.nevict,
	}
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = &lru.Cache{
			OnEvicted: func(key lru.Key, value interface{}) {
				val := value.(ByteView)
				c.nbytes -= int64(len(key.(string))) + int64(val.Len())
				c.nevict++
			},
		}
	}
	c.lru.Add(key, value, value.Expire())
	c.nbytes += int64(len(key)) + int64(value.Len())
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nget++
	if c.lru == nil {
		return
	}
	vi, ok := c.lru.Get(key)
	if !ok {
		return
	}
	c.nhit++
	return vi.(ByteView), true
}

func (c *cache) remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	c.lru.Remove(key)
}

func (c *cache) removeOldest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru != nil {
		c.lru.RemoveOldest()
	}
}

func (c *cache) bytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.nbytes
}

func (c *cache) items() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.itemsLocked()
}

func (c *cache) itemsLocked() int64 {
	if c.lru == nil {
		return 0
	}
	return int64(c.lru.Len())
}

// An AtomicInt is an int64 to be accessed atomically.
type AtomicInt int64

// Add atomically adds n to i.
func (i *AtomicInt) Add(n int64) {
	atomic.AddInt64((*int64)(i), n)
}

// Store atomically stores n to i.
func (i *AtomicInt) Store(n int64) {
	atomic.StoreInt64((*int64)(i), n)
}

// Get atomically gets the value of i.
func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

func (i *AtomicInt) String() string {
	return strconv.FormatInt(i.Get(), 10)
}

// CacheStats are returned by stats accessors on Group.
type CacheStats struct {
	Bytes     int64
	Items     int64
	Gets      int64
	Hits      int64
	Evictions int64
}

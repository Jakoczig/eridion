package atlas

import (
	"container/list"

	"github.com/Jakoczig/eridion/go/internal/perfstats"
)

// spatialCache holds parsed indexes and decoded blocks (G3K PERF-08). Keys
// are the assets' verified SHA-256 content hashes, so a release change,
// rollback or replaced file can never reuse another content's entry: entries
// are only ever created from bytes whose length and hash matched the signed
// manifest. Memory is bounded by an explicit byte budget with LRU eviction.
// It is owned by SpatialManager.ioMu (no separate lock).
type spatialCache struct {
	budget, used int64
	order        *list.List
	items        map[string]*list.Element
	// verified remembers asset hashes read and verified from disk since the
	// last eviction/activation, so prefetch planning does not re-hash them.
	verified map[string]bool
}

type cacheEntry struct {
	key   string
	bytes int64
	value any
}

// decodedCacheBudget bounds decoded index/block memory. Phase-A host PSS was
// about 30 MB; a full viewport decodes at most 16 MiB (query limit), so the
// cache holds one or two overlapping viewports.
const decodedCacheBudget = 24 << 20

const maxVerified = 4096

func newSpatialCache() *spatialCache {
	return &spatialCache{budget: decodedCacheBudget, order: list.New(), items: map[string]*list.Element{}, verified: map[string]bool{}}
}

func (c *spatialCache) get(key string) (any, bool) {
	if e, ok := c.items[key]; ok {
		c.order.MoveToFront(e)
		perfstats.Add("spatial.cache.hit", 1)
		return e.Value.(*cacheEntry).value, true
	}
	perfstats.Add("spatial.cache.miss", 1)
	return nil, false
}

func (c *spatialCache) put(key string, value any, bytes int64) {
	if bytes > c.budget {
		return
	}
	if e, ok := c.items[key]; ok {
		c.used -= e.Value.(*cacheEntry).bytes
		c.order.Remove(e)
		delete(c.items, key)
	}
	for c.used+bytes > c.budget && c.order.Len() > 0 {
		last := c.order.Back()
		ent := last.Value.(*cacheEntry)
		c.order.Remove(last)
		delete(c.items, ent.key)
		c.used -= ent.bytes
		perfstats.Add("spatial.cache.evict", 1)
	}
	c.items[key] = c.order.PushFront(&cacheEntry{key: key, bytes: bytes, value: value})
	c.used += bytes
}

func (c *spatialCache) markVerified(hash string) {
	if len(c.verified) >= maxVerified {
		c.verified = map[string]bool{}
	}
	c.verified[hash] = true
}

// forgetFiles drops the record of verified files (after eviction deleted
// some of them). Decoded entries stay valid: they are content-addressed.
func (c *spatialCache) forgetFiles() { c.verified = map[string]bool{} }

// reset frees everything (activation, rollback, Bubble removal).
func (c *spatialCache) reset() {
	c.order.Init()
	c.items = map[string]*list.Element{}
	c.used = 0
	c.forgetFiles()
}

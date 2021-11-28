package yocache

import (
	"sync"
	"time"
	"yocache/lru"
)

// cache is a concurrent version of lru.Cache with value type ByteView.
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64

	// each entry in lru.Cache lives for ttl duration
	// will not ttl if set to zero
	ttl time.Duration
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	var expire time.Time
	if c.ttl != 0 {
		expire = time.Now().Add(c.ttl)
	}
	c.lru.Add(key, value, expire)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}

	return
}

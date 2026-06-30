package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type DistributedCache struct {
	localCache map[string]cacheEntry
	mu         sync.RWMutex
	prefix     string
	fallback   *FallbackCache
	stats      cacheStats
}

type cacheEntry struct {
	Value     string
	ExpiresAt time.Time
	Hits      int
}

type cacheStats struct {
	mu              sync.RWMutex
	hits            int64
	misses          int64
	localHits       int64
	redisHits       int64
}

type FallbackCache struct {
	mu    sync.RWMutex
	data  map[string]cacheEntry
	maxSize int
}

func NewFallbackCache(maxSize int) *FallbackCache {
	return &FallbackCache{
		data:    make(map[string]cacheEntry),
		maxSize: maxSize,
	}
}

func (f *FallbackCache) Get(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if entry, ok := f.data[key]; ok && time.Now().Before(entry.ExpiresAt) {
		entry.Hits++
		f.mu.Lock()
		f.data[key] = entry
		f.mu.Unlock()
		return entry.Value, true
	}
	return "", false
}

func (f *FallbackCache) Set(key, value string, ttl time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.data) >= f.maxSize {
		f.evict()
	}

	f.data[key] = cacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		Hits:      0,
	}
}

func (f *FallbackCache) Delete(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, key)
}

func (f *FallbackCache) evict() {
	var oldestKey string
	var oldestTime time.Time
	for k, v := range f.data {
		if oldestTime.IsZero() || v.ExpiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.ExpiresAt
		}
	}
	if oldestKey != "" {
		delete(f.data, oldestKey)
	}
}

func NewDistributedCache(enableLocal bool) *DistributedCache {
	cache := &DistributedCache{
		localCache:  make(map[string]cacheEntry),
		prefix:      "wisehoof:",
		fallback:    NewFallbackCache(1000),
	}

	log.Printf("[Cache] Initialized: local=%v", enableLocal)

	return cache
}

func (c *DistributedCache) Get(ctx context.Context, key string) (string, bool) {
	fullKey := c.prefix + key

	if val, ok := c.localCacheGet(fullKey); ok {
		c.stats.mu.Lock()
		c.stats.hits++
		c.stats.localHits++
		c.stats.mu.Unlock()
		return val, true
	}

	if val, ok := c.fallback.Get(fullKey); ok {
		c.stats.mu.Lock()
		c.stats.hits++
		c.stats.mu.Unlock()
		c.localCacheSet(fullKey, val, 5*time.Minute)
		return val, true
	}

	c.stats.mu.Lock()
	c.stats.misses++
	c.stats.mu.Unlock()

	return "", false
}

func (c *DistributedCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	fullKey := c.prefix + key

	c.localCacheSet(fullKey, value, ttl)
	c.fallback.Set(fullKey, value, ttl)

	return nil
}

func (c *DistributedCache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key

	c.localCacheDelete(fullKey)
	c.fallback.Delete(fullKey)

	return nil
}

func (c *DistributedCache) GetMulti(ctx context.Context, keys []string) map[string]string {
	result := make(map[string]string)
	for _, key := range keys {
		if val, ok := c.Get(ctx, key); ok {
			result[key] = val
		}
	}
	return result
}

func (c *DistributedCache) localCacheGet(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.localCache[key]; ok && time.Now().Before(entry.ExpiresAt) {
		entry.Hits++
		c.localCache[key] = entry
		return entry.Value, true
	}
	return "", false
}

func (c *DistributedCache) localCacheSet(key, value string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.localCache) > 1000 {
		c.localCacheEvict()
	}

	c.localCache[key] = cacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		Hits:      0,
	}
}

func (c *DistributedCache) localCacheDelete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.localCache, key)
}

func (c *DistributedCache) localCacheEvict() {
	var evictKey string
	var oldestTime time.Time
	for k, v := range c.localCache {
		if oldestTime.IsZero() || v.ExpiresAt.Before(oldestTime) {
			evictKey = k
			oldestTime = v.ExpiresAt
		}
	}
	if evictKey != "" {
		delete(c.localCache, evictKey)
	}
}

func (c *DistributedCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	localSize := len(c.localCache)
	c.mu.Unlock()

	c.stats.mu.RLock()
	stats := map[string]interface{}{
		"local_cache_size": localSize,
		"local_fallback_size": len(c.fallback.data),
		"hits":             c.stats.hits,
		"misses":           c.stats.misses,
		"local_hits":       c.stats.localHits,
		"hit_ratio":        c.calculateHitRatio(),
	}
	c.stats.mu.RUnlock()

	return stats
}

func (c *DistributedCache) calculateHitRatio() float64 {
	total := c.stats.hits + c.stats.misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.hits) / float64(total)
}

func (c *DistributedCache) Close() error {
	return nil
}

func (c *DistributedCache) InvalidatePattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	c.localCache = make(map[string]cacheEntry)
	c.mu.Unlock()

	return nil
}

var _ = fmt.Sprintf

type SemanticCache struct {
	distCache *DistributedCache
 normalize func(string) string
}

func NewSemanticCache() *SemanticCache {
	return &SemanticCache{
		distCache: NewDistributedCache(true),
		normalize: normalizeQueryForMatching,
	}
}

func (s *SemanticCache) Get(ctx context.Context, query string) (string, bool) {
	normalized := s.normalize(query)
	if val, ok := s.distCache.Get(ctx, "sem:"+normalized); ok {
		return val, true
	}
	return "", false
}

func (s *SemanticCache) Set(ctx context.Context, query, response string, ttl time.Duration) error {
	normalized := s.normalize(query)
	return s.distCache.Set(ctx, "sem:"+normalized, response, ttl)
}

func (s *SemanticCache) Invalidate(ctx context.Context) error {
	return s.distCache.InvalidatePattern(ctx, "sem:*")
}

func (s *SemanticCache) GetStats() map[string]interface{} {
	return s.distCache.GetStats()
}


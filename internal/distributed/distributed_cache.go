package distributed

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
	Version   int64
}

type DistributedCache struct {
	mu       sync.RWMutex
	data     map[string]CacheEntry
	gossip   GossipProtocol
	nodeID   string
	ttl      time.Duration
	version  int64
}

func NewDistributedCache(gossip GossipProtocol, nodeID string, ttl time.Duration) *DistributedCache {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	c := &DistributedCache{
		data:    make(map[string]CacheEntry),
		gossip:  gossip,
		nodeID:  nodeID,
		ttl:     ttl,
		version: 1,
	}
	gossip.OnMessage(c.handleMessage)
	go c.cleanupLoop()
	return c
}

func (c *DistributedCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.data[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Value, true
}

func (c *DistributedCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.version++
	c.data[key] = CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
		Version:   c.version,
	}
	_ = c.gossip.Broadcast(&GossipMessage{
		Type:      "cache_update",
		FromNode:  c.nodeID,
		Payload:   map[string]interface{}{"key": key, "version": c.version},
		Timestamp: time.Now(),
	})
}

func (c *DistributedCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	c.version++
	_ = c.gossip.Broadcast(&GossipMessage{
		Type:      "cache_delete",
		FromNode:  c.nodeID,
		Payload:   map[string]interface{}{"key": key, "version": c.version},
		Timestamp: time.Now(),
	})
}

func (c *DistributedCache) Sync(ctx context.Context) error {
	return nil
}

func (c *DistributedCache) handleMessage(msg *GossipMessage) {
	if msg.FromNode == c.nodeID {
		return
	}
	switch msg.Type {
	case "cache_update", "cache_delete":
		c.mu.Lock()
		if key, ok := msg.Payload["key"].(string); ok {
			if msg.Type == "cache_delete" {
				delete(c.data, key)
			} else if version, ok := msg.Payload["version"].(int64); ok {
				if entry, exists := c.data[key]; !exists || version > entry.Version {
					c.data[key] = CacheEntry{
						Value:     msg.Payload["value"],
						ExpiresAt: time.Now().Add(c.ttl),
						Version:   version,
					}
				}
			}
		}
		c.mu.Unlock()
	}
}

func (c *DistributedCache) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.data {
			if now.After(entry.ExpiresAt) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *DistributedCache) MarshalEntries() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(c.data)
}

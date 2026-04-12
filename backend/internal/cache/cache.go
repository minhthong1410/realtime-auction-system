package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Cache struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

// Get retrieves a cached value and unmarshals it into dest.
// Returns false if cache miss.
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) bool {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		logger.Error("cache unmarshal error", zap.String("key", key), zap.Error(err))
		return false
	}
	return true
}

// Set caches a value with TTL.
func (c *Cache) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) {
	data, err := json.Marshal(val)
	if err != nil {
		return
	}
	c.rdb.Set(ctx, key, data, ttl)
}

// Del removes one or more cache keys.
func (c *Cache) Del(ctx context.Context, keys ...string) {
	c.rdb.Del(ctx, keys...)
}

// DelPattern removes all keys matching a pattern (e.g. "auctions:list:*").
func (c *Cache) DelPattern(ctx context.Context, pattern string) {
	iter := c.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		c.rdb.Del(ctx, keys...)
	}
}

// Key helpers
func KeyAuction(id string) string { return fmt.Sprintf("cache:auction:%s", id) }
func KeyAuctionList(page, size int32) string {
	return fmt.Sprintf("cache:auctions:list:%d:%d", page, size)
}
func KeyAuctionCount() string { return "cache:auctions:count" }

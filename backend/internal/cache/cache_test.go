package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15}) // use DB 15 for tests
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not available, skipping cache tests")
	}
	t.Cleanup(func() {
		rdb.FlushDB(context.Background())
		rdb.Close()
	})
	return New(rdb)
}

func TestCacheSetAndGet(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	type item struct {
		Name  string `json:"name"`
		Price int    `json:"price"`
	}

	// Set
	c.Set(ctx, "test:item", item{Name: "Rolex", Price: 50000}, 10*time.Second)

	// Get
	var result item
	found := c.Get(ctx, "test:item", &result)
	require.True(t, found)
	assert.Equal(t, "Rolex", result.Name)
	assert.Equal(t, 50000, result.Price)
}

func TestCacheGetMiss(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	var result string
	found := c.Get(ctx, "nonexistent:key", &result)
	assert.False(t, found)
}

func TestCacheDel(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	c.Set(ctx, "test:del", "value", 10*time.Second)

	var result string
	assert.True(t, c.Get(ctx, "test:del", &result))

	c.Del(ctx, "test:del")
	assert.False(t, c.Get(ctx, "test:del", &result))
}

func TestCacheDelPattern(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	c.Set(ctx, "cache:auctions:list:1:10", "page1", 10*time.Second)
	c.Set(ctx, "cache:auctions:list:2:10", "page2", 10*time.Second)
	c.Set(ctx, "cache:auction:abc", "detail", 10*time.Second)

	c.DelPattern(ctx, "cache:auctions:list:*")

	var result string
	assert.False(t, c.Get(ctx, "cache:auctions:list:1:10", &result))
	assert.False(t, c.Get(ctx, "cache:auctions:list:2:10", &result))
	assert.True(t, c.Get(ctx, "cache:auction:abc", &result)) // not deleted
}

func TestCacheTTLExpiry(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	c.Set(ctx, "test:ttl", "expire-me", 1*time.Second)

	var result string
	assert.True(t, c.Get(ctx, "test:ttl", &result))

	time.Sleep(1100 * time.Millisecond)
	assert.False(t, c.Get(ctx, "test:ttl", &result))
}

func TestKeyHelpers(t *testing.T) {
	assert.Equal(t, "cache:auction:abc-123", KeyAuction("abc-123"))
	assert.Equal(t, "cache:auctions:list:10:20", KeyAuctionList(10, 20))
	assert.Equal(t, "cache:auctions:count", KeyAuctionCount())
}

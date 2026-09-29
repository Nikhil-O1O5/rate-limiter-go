package limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/limiter"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, mr
}

func defaultCfg() limiter.BucketConfig {
	return limiter.BucketConfig{Capacity: 5, RefillRate: 1}
}

func TestAllow_FirstRequestAllowed(t *testing.T) {
	rdb, _ := setupRedis(t)
	tb := limiter.NewTokenBucket(rdb)


	result, err := tb.Allow(context.Background(), "user1", "/feed", defaultCfg())

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 4, result.Remaining)
	assert.Equal(t, int64(0), result.RetryAfterMS)
}

func TestAllow_CapacityExhausted(t *testing.T) {
	rdb, _ := setupRedis(t)
	tb := limiter.NewTokenBucket(rdb)
	cfg := defaultCfg()

	// drain the bucket
	for i := 0; i < int(cfg.Capacity); i++ {
		res, err := tb.Allow(context.Background(), "user1", "/feed", cfg)
		require.NoError(t, err)
		assert.True(t, res.Allowed, "request %d should be allowed", i+1)
	}

	// next request should be denied
	result, err := tb.Allow(context.Background(), "user1", "/feed", cfg)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
	assert.Greater(t, result.RetryAfterMS, int64(0))
}

func TestAllow_RetryAfterIsSet(t *testing.T) {
	rdb, _ := setupRedis(t)
	tb := limiter.NewTokenBucket(rdb)
	cfg := limiter.BucketConfig{Capacity: 1, RefillRate: 0.5} // 1 token every 2 seconds

	tb.Allow(context.Background(), "user1", "/hash", cfg) // consume the only token

	result, err := tb.Allow(context.Background(), "user1", "/hash", cfg)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.GreaterOrEqual(t, result.RetryAfterMS, int64(1000)) // at least 1 second wait
}

func TestAllow_TokensRefillOverTime(t *testing.T) {
	rdb, _ := setupRedis(t)
	cfg := limiter.BucketConfig{Capacity: 2, RefillRate: 1}

	// use a fake clock starting at a fixed time
	fakeNow := time.Now().UnixMilli()
	tb := limiter.NewTokenBucketWithClock(rdb, func() int64 { return fakeNow })

	// drain the bucket
	tb.Allow(context.Background(), "user1", "/feed", cfg)
	tb.Allow(context.Background(), "user1", "/feed", cfg)

	denied, err := tb.Allow(context.Background(), "user1", "/feed", cfg)
	require.NoError(t, err)
	assert.False(t, denied.Allowed)

	// advance fake clock by 2 seconds — 2 new tokens should refill
	fakeNow += 2000

	result, err := tb.Allow(context.Background(), "user1", "/feed", cfg)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestAllow_UserIsolation(t *testing.T) {
	rdb, _ := setupRedis(t)
	tb := limiter.NewTokenBucket(rdb)
	cfg := limiter.BucketConfig{Capacity: 1, RefillRate: 1}

	// exhaust user1
	tb.Allow(context.Background(), "user1", "/feed", cfg)
	denied, err := tb.Allow(context.Background(), "user1", "/feed", cfg)
	require.NoError(t, err)
	assert.False(t, denied.Allowed)

	// user2 should have a full bucket
	result, err := tb.Allow(context.Background(), "user2", "/feed", cfg)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestAllow_EndpointIsolation(t *testing.T) {
	rdb, _ := setupRedis(t)
	tb := limiter.NewTokenBucket(rdb)
	cfg := limiter.BucketConfig{Capacity: 1, RefillRate: 1}

	// exhaust /hash for user1
	tb.Allow(context.Background(), "user1", "/hash", cfg)
	denied, err := tb.Allow(context.Background(), "user1", "/hash", cfg)
	require.NoError(t, err)
	assert.False(t, denied.Allowed)

	// /feed should have its own full bucket for the same user
	result, err := tb.Allow(context.Background(), "user1", "/feed", cfg)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestAllow_RemainingDecrementsCorrectly(t *testing.T) {
	rdb, _ := setupRedis(t)
	tb := limiter.NewTokenBucket(rdb)
	cfg := limiter.BucketConfig{Capacity: 3, RefillRate: 1}

	expected := []int{2, 1, 0}
	for i, want := range expected {
		res, err := tb.Allow(context.Background(), "user1", "/feed", cfg)
		require.NoError(t, err)
		assert.True(t, res.Allowed)
		assert.Equal(t, want, res.Remaining, "after request %d", i+1)
	}
}

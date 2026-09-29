package limiter

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/token_bucket.lua
var tokenBucketScript string

// Result is returned for every Allow call.
type Result struct {
	Allowed        bool
	Remaining      int
	RetryAfterMS   int64
}

// BucketConfig defines the shape of a single rate limit rule.
type BucketConfig struct {
	Capacity   float64 // max tokens the bucket can hold
	RefillRate float64 // tokens added per second
}

type TokenBucket struct {
	rdb    *redis.Client
	script *redis.Script
}

func NewTokenBucket(rdb *redis.Client) *TokenBucket {
	return &TokenBucket{
		rdb:    rdb,
		script: redis.NewScript(tokenBucketScript),
	}
}

// Allow checks whether the given user is allowed to proceed for the given endpoint.
func (tb *TokenBucket) Allow(ctx context.Context, userID, endpoint string, cfg BucketConfig) (*Result, error) {
	key := fmt.Sprintf("rate_limit:%s:%s", userID, endpoint)
	nowMS := time.Now().UnixMilli()
	ttl := int(cfg.Capacity/cfg.RefillRate) * 2 // bucket drains and refills twice before expiry

	res, err := tb.script.Run(ctx, tb.rdb, []string{key},
		cfg.Capacity,
		cfg.RefillRate,
		nowMS,
		ttl,
	).Int64Slice()
	if err != nil {
		return nil, fmt.Errorf("token bucket script: %w", err)
	}

	return &Result{
		Allowed:      res[0] == 1,
		Remaining:    int(res[1]),
		RetryAfterMS: res[2],
	}, nil
}

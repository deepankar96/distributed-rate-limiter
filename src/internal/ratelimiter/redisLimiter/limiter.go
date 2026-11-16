package redislimiter

import (
	"context"
	_ "embed"
	"fmt"
	"math"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	client     *redis.Client
	capacity   float64
	refillRate float64
}

//go:embed token_bucket.lua
var tokenBucketScript string

func NewLimiter(c *redis.Client, capacity, refillRate float64) (*Limiter, error) {

	if c == nil {
		return nil, fmt.Errorf("Redis client must not be nil")
	}
	if math.IsNaN(capacity) || math.IsInf(capacity, 0) || capacity < 1 {
		return nil, fmt.Errorf("capacity must be finite and at least 1")
	}
	if math.IsNaN(refillRate) || math.IsInf(refillRate, 0) || refillRate <= 0 {
		return nil, fmt.Errorf("refill rate must be finite and positive")
	}

	return &Limiter{
		client:     c,
		capacity:   capacity,
		refillRate: refillRate,
	}, nil
}

func (l *Limiter) buildRedisKey(userID string) string {
	key := "ratelimit:user:" + userID
	return key
}

func (l *Limiter) checkLimit(ctx context.Context, key string) (int64, error) {
	return l.client.Eval(
		ctx,
		tokenBucketScript,
		[]string{key},
		l.capacity,
		l.refillRate,
	).Int64()
}

func (l *Limiter) Allow(ctx context.Context, userId string) (bool, error) {
	key := l.buildRedisKey(userId)

	decision, err := l.checkLimit(ctx, key)
	if err != nil {
		return false, fmt.Errorf("Redis rate-limit check: %w", err)
	}

	if decision == 1 {
		return true, nil
	} else {
		return false, nil
	}

}

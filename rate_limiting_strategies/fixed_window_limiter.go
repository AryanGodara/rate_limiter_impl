package rate_limiting_strategies

import (
	"context"
	"errors"
	"fmt"
	"github.com/aryangodara/rate_limiter_impl"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	_ rate_limiter_impl.Strategy = &fixedWindowLimiter{}
)

const (
	keyDNE      = -2
	keyNoExpire = -1
)

type fixedWindowLimiter struct {
	client *redis.Client
	now    func() time.Time
}

// NewFixedWindowLimiter creates a new fixed window rate limiter.
func NewFixedWindowLimiter(client *redis.Client, now func() time.Time) rate_limiter_impl.Strategy {
	return &fixedWindowLimiter{
		client: client,
		now:    now,
	}
}

// Execute performs rate limiting using a fixed window strategy.
func (f *fixedWindowLimiter) Execute(ctx context.Context, r *rate_limiter_impl.Request) (*rate_limiter_impl.Result, error) {
	// Redis pipeline to optimize network round trips.
	pipe := f.client.Pipeline()
	getCmd := pipe.Get(ctx, r.Key)
	ttlCmd := pipe.TTL(ctx, r.Key)

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("error executing Redis pipeline for key %v: %w", r.Key, err)
	}

	var ttl time.Duration

	if duration, err := ttlCmd.Result(); err != nil || duration == keyDNE || duration == keyNoExpire {
		ttl = r.Duration
		if err := f.client.Expire(ctx, r.Key, r.Duration).Err(); err != nil {
			return nil, fmt.Errorf("error setting expiration for key %v: %w", r.Key, err)
		}
	} else {
		ttl = duration
	}

	expirationTime := f.now().Add(ttl)

	if count, err := getCmd.Uint64(); err != nil && errors.Is(err, redis.Nil) {
	} else if count >= r.Limit {
		return &rate_limiter_impl.Result{
			State:         rate_limiter_impl.Deny,
			TotalRequests: count,
			ExpiresAt:     expirationTime,
		}, nil
	}

	incrementCmd := f.client.Incr(ctx, r.Key)
	requestCount, err := incrementCmd.Uint64()
	if err != nil {
		return nil, fmt.Errorf("error incrementing key %v: %w", r.Key, err)
	}

	if requestCount > r.Limit {
		return &rate_limiter_impl.Result{
			State:         rate_limiter_impl.Deny,
			TotalRequests: requestCount,
			ExpiresAt:     expirationTime,
		}, nil
	}

	return &rate_limiter_impl.Result{
		State:         rate_limiter_impl.Allow,
		TotalRequests: requestCount,
		ExpiresAt:     expirationTime,
	}, nil
}

package rate_limiting_strategies

import (
	"context"
	"errors"
	"fmt"
	"github.com/aryangodara/rate_limiter_impl"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

type tokenBucketLimiter struct {
	client        *redis.Client
	latRefillTime func() time.Time
	maxTokens     int64
	refillTime    time.Duration
	refillAmount  int64
}

// NewTokenBucketLimiter creates a new Token Bucket rate limiter.
func NewTokenBucketLimiter(client *redis.Client, now func() time.Time, maxTokens int64, refillTime time.Duration, refillAmount int64) rate_limiter_impl.Strategy {
	return &tokenBucketLimiter{
		client:        client,
		latRefillTime: now,
		maxTokens:     maxTokens,
		refillTime:    refillTime,
		refillAmount:  refillAmount,
	}
}

func (t *tokenBucketLimiter) Execute(ctx context.Context, r *rate_limiter_impl.Request) (*rate_limiter_impl.Result, error) {
	now := t.latRefillTime().Unix()
	lastUpdateKey := r.Key + ":lastUpdate"
	tokenCountKey := r.Key + ":tokens"

	// Fetch last update time
	lastUpdateStr, err := t.client.Get(ctx,
		lastUpdateKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to get last update time: %w", err)
	}

	var lastUpdate int64
	if lastUpdateStr != "" {
		lastUpdate, err = strconv.ParseInt(lastUpdateStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last update time: %w", err)
		}
	}

	// Fetch current token count
	tokenCountStr, err := t.client.Get(ctx, tokenCountKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to get token count: %w", err)
	}

	var tokenCount int64
	if tokenCountStr != "" {
		tokenCount, err = strconv.ParseInt(tokenCountStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse token count: %w", err)
		}
	} else {
		tokenCount = t.maxTokens
	}

	// Calculate the number of tokens to refill
	if lastUpdate > 0 {
		refillCount := (now - lastUpdate) / int64(t.refillTime.Seconds())
		if refillCount > 0 {
			tokenCount += refillCount * t.refillAmount
			if tokenCount > t.maxTokens {
				tokenCount = t.maxTokens
			}
			lastUpdate = now
		}
	} else {
		lastUpdate = now
	}

	// Update tokens and last update time in Redis
	p := t.client.Pipeline()
	p.Set(ctx, tokenCountKey, tokenCount, 0)
	p.Set(ctx, lastUpdateKey, lastUpdate, 0)
	if _, err := p.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to update tokens and last update time: %w", err)
	}

	// Check if request can be allowed
	if tokenCount > 0 {
		tokenCount--
		p.Set(ctx, tokenCountKey, tokenCount, 0)
		if _, err := p.Exec(ctx); err != nil {
			return nil, fmt.Errorf("failed to decrement token count: %w", err)
		}

		return &rate_limiter_impl.Result{
			State:         rate_limiter_impl.Allow,
			TotalRequests: uint64(tokenCount),
			ExpiresAt:     time.Unix(now, 0).Add(t.refillTime),
		}, nil
	}

	return &rate_limiter_impl.Result{
		State:         rate_limiter_impl.Deny,
		TotalRequests: 0,
		ExpiresAt:     time.Unix(lastUpdate, 0).Add(t.refillTime),
	}, nil
}

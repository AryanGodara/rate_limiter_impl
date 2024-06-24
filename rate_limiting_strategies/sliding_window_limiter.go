package rate_limiting_strategies

import (
	"context"
	"fmt"
	"github.com/aryangodara/rate_limiter_impl"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

var (
	_ rate_limiter_impl.Strategy = &slidingWindowLimiter{}
)

const (
	maxSortedSetScore = "+inf"
	minSortedSetScore = "-inf"
)

type slidingWindowLimiter struct {
	client *redis.Client
	now    func() time.Time
}

// NewSlidingWindowLimiter initializes a new sliding window rate limiter.
func NewSlidingWindowLimiter(client *redis.Client, now func() time.Time) rate_limiter_impl.Strategy {
	return &slidingWindowLimiter{
		client: client,
		now:    now,
	}
}

// Execute performs rate limiting using a sliding window strategy.
func (s *slidingWindowLimiter) Execute(ctx context.Context, r *rate_limiter_impl.Request) (*rate_limiter_impl.Result, error) {
	now := s.now()
	expiresAt := now.Add(r.Duration)
	minimum := now.Add(-r.Duration)

	result, err := s.client.ZCount(ctx, r.Key, strconv.FormatInt(minimum.UnixMilli(), 10), maxSortedSetScore).Uint64()
	if err == nil && result >= r.Limit {
		return &rate_limiter_impl.Result{
			State:         rate_limiter_impl.Deny,
			TotalRequests: result,
			ExpiresAt:     expiresAt,
		}, nil
	}

	// every request needs an UUID
	item := uuid.New()

	p := s.client.Pipeline()

	// we then remove all the expired requests
	removeByScore := p.ZRemRangeByScore(ctx, r.Key, "0", strconv.FormatInt(minimum.UnixMilli(), 10))

	// we add the current request
	add := p.ZAdd(ctx, r.Key, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: item.String(),
	})

	// count how many non-expired requests we have on the sorted set
	count := p.ZCount(ctx, r.Key, minSortedSetScore, maxSortedSetScore)

	if _, err := p.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to execute sorted set pipeline for key: %v: %w", r.Key, err)
	}

	if err := removeByScore.Err(); err != nil {
		return nil, fmt.Errorf("failed to remove old requests from key %v: %w", r.Key, err)
	}

	if err := add.Err(); err != nil {
		return nil, fmt.Errorf("failed to add item to key %v: %w", r.Key, err)
	}

	totalRequests, err := count.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to count items for key %v: %w", r.Key, err)
	}

	requests := uint64(totalRequests)

	if requests > r.Limit {
		return &rate_limiter_impl.Result{
			State:         rate_limiter_impl.Deny,
			TotalRequests: requests,
			ExpiresAt:     expiresAt,
		}, nil
	}

	return &rate_limiter_impl.Result{
		State:         rate_limiter_impl.Allow,
		TotalRequests: requests,
		ExpiresAt:     expiresAt,
	}, nil
}

package rate_limiting_strategies

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/aryangodara/rate_limiter_impl"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTokenBucketLimiter_Execute(t *testing.T) {
	tt := []struct {
		desc        string
		runs        int64
		req         *rate_limiter_impl.Request
		res         *rate_limiter_impl.Result
		err         error
		timeAdvance time.Duration
	}{
		{
			desc: "returns Allow for requests under limit",
			req: &rate_limiter_impl.Request{
				Key:      "some-user",
				Limit:    10,
				Duration: time.Minute,
			},
			res: &rate_limiter_impl.Result{
				State:         rate_limiter_impl.Allow,
				TotalRequests: 5,
				ExpiresAt:     time.Date(2024, time.June, 23, 10, 16, 30, 0, time.Local),
			},
			runs: 5,
			err:  nil,
		},
		{
			desc: "returns Deny for requests over limit",
			req: &rate_limiter_impl.Request{
				Key:      "some-user",
				Limit:    10,
				Duration: time.Minute,
			},
			res: &rate_limiter_impl.Result{
				State:         rate_limiter_impl.Deny,
				TotalRequests: 0,
				ExpiresAt:     time.Date(2024, time.June, 23, 10, 16, 30, 0, time.Local),
			},
			runs: 11,
			err:  nil,
		},
		{
			desc: "refills tokens after interval",
			req: &rate_limiter_impl.Request{
				Key:      "some-user",
				Limit:    10,
				Duration: time.Minute,
			},
			res: &rate_limiter_impl.Result{
				State:         rate_limiter_impl.Allow,
				TotalRequests: 9,
				ExpiresAt:     time.Date(2024, time.June, 23, 10, 25, 30, 0, time.Local),
			},
			runs:        10,
			timeAdvance: time.Minute,
			err:         nil,
		},
	}

	for _, ts := range tt {
		t.Run(ts.desc, func(t *testing.T) {
			server, err := miniredis.Run()
			require.NoError(t, err)
			defer server.Close()

			now := time.Date(2024, time.June, 23, 10, 15, 30, 0, time.Local)

			client := redis.NewClient(&redis.Options{
				Addr: server.Addr(),
			})
			defer client.Close()

			limiter := NewTokenBucketLimiter(client, func() time.Time {
				return now
			}, 10, time.Minute, 10)
			var lastRes *rate_limiter_impl.Result
			var lastErr error

			for x := int64(0); x < ts.runs; x++ {
				lastRes, lastErr = limiter.Execute(context.Background(), ts.req)
				if ts.timeAdvance != 0 {
					server.FastForward(ts.timeAdvance)
					now = now.Add(ts.timeAdvance)
				}
			}

			assert.Equal(t, ts.res, lastRes)
			assert.Equal(t, ts.err, lastErr)
		})
	}
}

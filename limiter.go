package rate_limiter_impl

import (
	"context"
	"time"
)

// Request defines a request to be rate-limited.
type Request struct {
	Key      string
	Limit    uint64
	Duration time.Duration
}

// State represents the result of rate limiting.
type State int64

const (
	Deny State = iota
	Allow
)

// State strings for HTTP headers
var stateStrings = map[State]string{
	Allow: "Allow",
	Deny:  "Deny",
}

// Result is the outcome of a rate limit check.
type Result struct {
	State         State
	TotalRequests uint64
	ExpiresAt     time.Time
}

// Strategy interface defines the contract for rate limiting strategies.
type Strategy interface {
	Execute(ctx context.Context, r *Request) (*Result, error)
}

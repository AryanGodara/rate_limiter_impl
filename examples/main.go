package main

import (
	rate_limiting_strategies "github.com/aryangodara/rate_limiter_impl/rate_limiting_strategies"
	"log"
	"net/http"
	"time"

	"github.com/aryangodara/rate_limiter_impl"
	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Rate limiter strategies
	fixedWindowLimiter := rate_limiting_strategies.NewFixedWindowLimiter(client, time.Now)
	rollingWindowLimiter := rate_limiting_strategies.NewSlidingWindowLimiter(client, time.Now)
	tokenBucketLimiter := rate_limiting_strategies.NewTokenBucketLimiter(client, time.Now, 10, time.Minute, 5)

	// Rate limiter configs
	fixedWindowConfig := &rate_limiter_impl.RateLimiterConfig{
		Extractor:   rate_limiter_impl.NewHttpHeaderExtractor("X-Client-ID"),
		Strategy:    fixedWindowLimiter,
		Expiration:  time.Minute,
		MaxRequests: 5,
	}

	rollingWindowConfig := &rate_limiter_impl.RateLimiterConfig{
		Extractor:   rate_limiter_impl.NewHttpHeaderExtractor("X-Client-ID"),
		Strategy:    rollingWindowLimiter,
		Expiration:  time.Minute,
		MaxRequests: 5,
	}

	tokenBucketConfig := &rate_limiter_impl.RateLimiterConfig{
		Extractor:   rate_limiter_impl.NewHttpHeaderExtractor("X-Client-ID"),
		Strategy:    tokenBucketLimiter,
		Expiration:  time.Minute,
		MaxRequests: 5,
	}

	// Define HTTP handler
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// Wrap http handler with rate-limiter middleware
	http.Handle("/fixed-window-limiter", rate_limiter_impl.NewHTTPRateLimiterHandler(originalHandler, fixedWindowConfig))
	http.Handle("/sliding-window-limiter", rate_limiter_impl.NewHTTPRateLimiterHandler(originalHandler, rollingWindowConfig))
	http.Handle("/token-bucket-limiter", rate_limiter_impl.NewHTTPRateLimiterHandler(originalHandler, tokenBucketConfig))

	log.Println("Server started at :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

# Rate Limiting Implementation and Strategies

## Introduction

This repository demonstrates the implementation of three different rate limiting strategies:

1. **Fixed Window Counter**: A simple yet effective technique that counts the number of requests in a fixed time window.
2. **Sliding Window Log**: A more accurate method that records request timestamps and provides a moving window for rate limiting.
3. **Token Bucket**: Allows a burst of traffic by accumulating tokens that are consumed by incoming requests.

## How to Use

To integrate this rate limiting library into your own API, follow these steps:

1. **Install the package**:
   First, ensure you have the repository in your `go.mod` file:
   ```bash
   go get github.com/aryangodara/rate_limiter_impl
   ```

2. **Choose a Strategy**:
   Decide which rate limiting strategy you want to use: Fixed Window, Sliding Window, or Token Bucket.

3. **Initialize the Rate Limiter**:
   Set up the rate limiter in your API code. Here’s an example using the Fixed Window strategy:

   ```go
   package main

   import (
       "github.com/aryangodara/rate_limiter_impl"
       "github.com/aryangodara/rate_limiter_impl/rate_limiting_strategies"
       "github.com/redis/go-redis/v9"
       "net/http"
       "time"
   )

   func main() {
       client := redis.NewClient(&redis.Options{
           Addr: "localhost:6379", // Redis server address
       })

       rateLimiter := rate_limiting_strategies.NewFixedWindowLimiter(client, time.Now)

       config := &rate_limiter_impl.RateLimiterConfig{
           Extractor:   rate_limiter_impl.NewHttpHeaderExtractor("X-API-KEY"),
           Strategy:    rateLimiter,
           Expiration:  time.Minute,
           MaxRequests: 100, // Maximum requests per time window
       }

       handler := rate_limiter_impl.NewHTTPRateLimiterHandler(http.HandlerFunc(myHandler), config)
       http.Handle("/", handler)
       http.ListenAndServe(":8080", nil)
   }

   func myHandler(w http.ResponseWriter, r *http.Request) {
       w.Write([]byte("Request allowed!"))
   }
   ```

4. **Run the API**:
   Start your API server:
   ```bash
   go run main.go
   ```

In this setup, the rate limiter checks the `X-API-KEY` header to identify unique clients. The `myHandler` function will be called only if the rate limit has not been exceeded.

## Running Tests

To run the test cases for the rate limiting strategies, use the following command:

```bash
go test ./rate_limiting_strategies/...
```

This will execute all the test cases defined in the `fixed_window_limiter_test.go`, `sliding_window_limiter_test.go`, and `token_bucket_limiter_test.go` files.

## Repository Structure

```
.
├── README.md
├── go.mod
├── go.sum
├── middleware.go
├── limiter.go
└── rate_limiting_strategies
    ├── fixed_window_limiter.go
    ├── fixed_window_limiter_test.go
    ├── sliding_window_limiter.go
    ├── sliding_window_limiter_test.go
    ├── token_bucket_limiter.go
    └── token_bucket_limiter_test.go
```

### Description of Files

- **middleware.go**: Contains HTTP handler for rate limiting.
- **limiter.go**: Defines common interfaces and structures for rate limiting strategies.
- **rate_limiting_strategies/**: Directory containing the implementation and tests for each rate limiting strategy.

### Brief Description of Rate Limiting Strategies

1. **Fixed Window Limiter**:
    - Implements a counter that resets after a fixed time window.
    - Suitable for simple rate limiting needs but can allow bursts of traffic at the window boundaries.

2. **Sliding Window Limiter**:
    - Uses a sorted set to keep track of request timestamps.
    - Provides more accurate rate limiting by smoothing out bursts.

3. **Token Bucket Limiter**:
    - Tokens are added to a bucket at a fixed rate.
    - Each request consumes a token, allowing for handling bursts efficiently.

### Pros and Cons of Each Strategy

#### Fixed Window Counter

**Pros:**
- Simple to implement.
- Memory efficient.

**Cons:**
- Susceptible to bursts of traffic at window edges.
- Less accurate compared to sliding window methods.

#### Sliding Window Log

**Pros:**
- More accurate than fixed window counters.
- Smooths out spikes in traffic.

**Cons:**
- Higher memory usage due to storing request logs.
- More complex to implement.

#### Token Bucket

**Pros:**
- Allows handling of burst traffic.
- Memory efficient and easy to implement.

**Cons:**
- Potential race conditions in distributed systems.
- Requires synchronization mechanisms like distributed locks.

By understanding and using these different strategies, you can choose the one that best fits your application's requirements for rate limiting.
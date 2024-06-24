package rate_limiter_impl

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	_ http.Handler = &httpRateLimiterHandler{}
	_ Extractor    = &httpHeaderExtractor{}
)

const (
	rateLimitingTotalRequests = "Rate-limiting-Total-Requests"
	rateLimitingState         = "Rate-Limiting-State"
	rateLimitingExpiresAt     = "Rate-Limiting-Expires-At"
)

// Extractor extracts a key from an HTTP request for rate limiting.
type Extractor interface {
	Extract(r *http.Request) (string, error)
}

type httpHeaderExtractor struct {
	headers []string
}

// Extract extracts values from HTTP headers to build the key.
func (h *httpHeaderExtractor) Extract(r *http.Request) (string, error) {
	values := make([]string, 0, len(h.headers))

	for _, key := range h.headers {
		// if we can't find a value for a header we should return an error
		if value := strings.TrimSpace(r.Header.Get(key)); value != "" {
			values = append(values, value)
		} else {
			return "", fmt.Errorf("header %v must have a value set", key)
		}
	}

	return strings.Join(values, "-"), nil
}

// NewHttpHeaderExtractor creates a new Extractor.
func NewHttpHeaderExtractor(headers ...string) Extractor {
	return &httpHeaderExtractor{headers: headers}
}

// RateLimiterConfig holds configuration for rate limiting.
type RateLimiterConfig struct {
	Extractor   Extractor
	Strategy    Strategy
	Expiration  time.Duration
	MaxRequests uint64
}

type httpRateLimiterHandler struct {
	handler http.Handler
	config  *RateLimiterConfig
}

// NewHTTPRateLimiterHandler wraps an existing http.Handler and performs rate limiting before forwarding the
// request to the API
func NewHTTPRateLimiterHandler(originalHandler http.Handler, config *RateLimiterConfig) http.Handler {
	return &httpRateLimiterHandler{
		handler: originalHandler,
		config:  config,
	}
}

// ServeHTTP performs rate limiting and forwards the request if allowed.
func (h *httpRateLimiterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, err := h.config.Extractor.Extract(r)
	if err != nil {
		h.writeRespone(w, http.StatusBadRequest, "failed to connect rate limiting key from request: %v", err)
		return
	}

	result, err := h.config.Strategy.Execute(r.Context(), &Request{
		Key:      key,
		Limit:    h.config.MaxRequests,
		Duration: h.config.Expiration,
	})

	if err != nil {
		h.writeRespone(w, http.StatusInternalServerError, "failed to run rate limiting for request: %v", err)
		return
	}

	w.Header().Set(rateLimitingTotalRequests, strconv.FormatUint(result.TotalRequests, 10))
	w.Header().Set(rateLimitingState, stateStrings[result.State])
	w.Header().Set(rateLimitingExpiresAt, result.ExpiresAt.Format(time.RFC3339))

	// Too many requests
	if result.State == Deny {
		h.writeRespone(w, http.StatusTooManyRequests, "you have sent too many requests to this service, slow down please")
		return
	}

	h.handler.ServeHTTP(w, r)
}

func (h *httpRateLimiterHandler) writeRespone(w http.ResponseWriter, status int, msg string, args ...interface{}) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(fmt.Sprintf(msg, args...))); err != nil {
		fmt.Printf("failed to write body to HTTP request: %v", err)
	}
}

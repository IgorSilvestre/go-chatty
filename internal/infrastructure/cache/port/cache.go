package port

import (
	"context"
	"time"
)

// Cache defines the minimal contract for a key-value cache used by the application.
// Implementations should be concurrency-safe.
// All methods must be context-aware to allow caller-driven timeouts/cancellation.
//
// Note: Values are stored as strings to keep the port generic and avoid coupling
// to serialization concerns. Adapters may add helper methods in their own packages
// if needed, but this is the stable port exposed to the rest of the app.
type Cache interface {
	// Get fetches the value for key. It returns an empty string and nil error
	// if the key exists but empty; it returns a non-nil error only for transport
	// or server errors. Cache misses should be represented as ("", nil, ErrMiss)
	// pattern; to simplify, we return ("", ErrMiss) for misses.
	Get(ctx context.Context, key string) (string, error)

	// Set stores value at key with the provided TTL. Zero or negative TTL means
	// no expiration (persist until evicted).
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Del removes one or more keys and returns the number of keys removed.
	Del(ctx context.Context, keys ...string) (int64, error)

	// Ping verifies connectivity with the cache backend.
	Ping(ctx context.Context) error

	// Close releases any resources held by the cache.
	Close() error
}

// ErrMiss should be used by adapters to signal a cache miss in a typed way.
// This allows callers to differentiate misses from transport errors if desired.
var ErrMiss = errMiss{}

type errMiss struct{}

func (e errMiss) Error() string { return "cache: miss" }

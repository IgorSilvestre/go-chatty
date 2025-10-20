package port

import (
	"context"
	"time"
)

// Task represents a background job message with a type and opaque payload bytes.
// Type should be a stable string identifier. Payload encoding is up to callers.
// Keep this port free from serialization concerns to avoid coupling.
type Task struct {
	Type    string
	Payload []byte
}

// Handler processes a Task. Return a non-nil error to signal retry per adapter policy.
// Handlers must be idempotent.
type Handler func(ctx context.Context, task Task) error

// EnqueueOption controls enqueue behavior. Adapters map supported fields to the
// underlying backend as best-effort; unsupported fields may be ignored.
// Zero values mean "unspecified".
type EnqueueOption struct {
	Queue     string        // logical queue name
	ProcessIn time.Duration // delay before processing
	ProcessAt time.Time     // absolute schedule time (takes precedence over ProcessIn if set)
	MaxRetry  int           // max retries for the task
	UniqueTTL time.Duration // enforce uniqueness within TTL window (if supported)
	Retention time.Duration // keep result metadata for this duration (if supported)
	Deadline  time.Time     // hard deadline for processing (if supported)
}

// Client enqueues tasks for background processing.
type Client interface {
	Enqueue(ctx context.Context, t Task, opts ...EnqueueOption) (id string, err error)
	Close() error
}

// Server runs background workers that handle tasks.
// Implementations should block in Run until Stop/Shutdown is called or context is canceled.
type Server interface {
	Register(taskType string, h Handler)
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

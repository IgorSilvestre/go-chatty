package adapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hibiken/asynq"

	"go-chatty/internal/infrastructure/queue/port"
)

// ===================== Client =====================

// AsynqClient implements port.Client using github.com/hibiken/asynq
// and Redis as the backing store.
type AsynqClient struct {
	client *asynq.Client
}

// NewAsynqClientFromEnv constructs a client using REDIS_URL env var.
func NewAsynqClientFromEnv() (*AsynqClient, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, errors.New("asynq: REDIS_URL environment variable is not set")
	}
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("asynq: parse REDIS_URL: %w", err)
	}
	c := asynq.NewClient(opt)
	return &AsynqClient{client: c}, nil
}

// Ensure interface is satisfied
var _ port.Client = (*AsynqClient)(nil)

func (a *AsynqClient) Enqueue(ctx context.Context, t port.Task, opts ...port.EnqueueOption) (string, error) {
	if t.Type == "" {
		return "", errors.New("asynq: task type is required")
	}
	at := asynq.NewTask(t.Type, t.Payload)
	var asynqOpts []asynq.Option
	if len(opts) > 0 {
		// Use first option only to keep port minimal; callers can pass one consolidated option.
		op := opts[0]
		if !op.ProcessAt.IsZero() {
			asynqOpts = append(asynqOpts, asynq.ProcessAt(op.ProcessAt))
		} else if op.ProcessIn > 0 {
			asynqOpts = append(asynqOpts, asynq.ProcessIn(op.ProcessIn))
		}
		if op.Queue != "" {
			asynqOpts = append(asynqOpts, asynq.Queue(op.Queue))
		}
		if op.MaxRetry > 0 {
			asynqOpts = append(asynqOpts, asynq.MaxRetry(op.MaxRetry))
		}
		if op.UniqueTTL > 0 {
			asynqOpts = append(asynqOpts, asynq.Unique(op.UniqueTTL))
		}
		if op.Retention > 0 {
			asynqOpts = append(asynqOpts, asynq.Retention(op.Retention))
		}
		if !op.Deadline.IsZero() {
			asynqOpts = append(asynqOpts, asynq.Deadline(op.Deadline))
		}
	}
	info, err := a.client.EnqueueContext(ctx, at, asynqOpts...)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

func (a *AsynqClient) Close() error {
	return a.client.Close()
}

// ===================== Server =====================

// AsynqServer implements port.Server using github.com/hibiken/asynq
type AsynqServer struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewAsynqServer constructs a server using REDIS_URL and optional config:
// - ASYNQ_CONCURRENCY: int (default 10)
// - ASYNQ_QUEUES: CSV like "critical=6,default=3,low=1" (default "default=1")
func NewAsynqServer() (*AsynqServer, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, errors.New("asynq: REDIS_URL environment variable is not set")
	}
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("asynq: parse REDIS_URL: %w", err)
	}

	concurrency := 10
	if v := strings.TrimSpace(os.Getenv("ASYNQ_CONCURRENCY")); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			concurrency = i
		}
	}

	// Default to consuming both "default" and "chat" queues so tasks are picked up when running API directly
	queues := map[string]int{"default": 1, "chat": 1}
	if v := strings.TrimSpace(os.Getenv("ASYNQ_QUEUES")); v != "" {
		parsed := parseQueueWeights(v)
		if len(parsed) > 0 {
			queues = parsed
		}
	}

	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: concurrency,
		Queues:      queues,
		// Base settings can be extended later as needed
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			// Best-effort log to stderr without introducing logging deps
			_, _ = fmt.Fprintf(os.Stderr, "asynq error: type=%s err=%v\n", task.Type(), err)
		}),
	})
	return &AsynqServer{server: srv, mux: asynq.NewServeMux()}, nil
}

// Ensure interface is satisfied
var _ port.Server = (*AsynqServer)(nil)

func (s *AsynqServer) Register(taskType string, h port.Handler) {
	s.mux.HandleFunc(taskType, func(ctx context.Context, t *asynq.Task) error {
		pt := port.Task{Type: t.Type(), Payload: t.Payload()}
		return h(ctx, pt)
	})
}

// Run starts the server and blocks until the context is canceled, then gracefully shuts down.
func (s *AsynqServer) Run(ctx context.Context) error {
	if err := s.server.Start(s.mux); err != nil {
		return err
	}
	// Wait for cancellation
	<-ctx.Done()
	// Graceful shutdown (no context argument supported in current asynq version)
	s.server.Shutdown()
	return nil
}

// Stop gracefully shuts down the server.
func (s *AsynqServer) Stop(ctx context.Context) error {
	_ = ctx // context not used by current Shutdown signature
	s.server.Shutdown()
	return nil
}

// parseQueueWeights parses strings like "critical=6,default=3,low=1" into a map.
func parseQueueWeights(s string) map[string]int {
	res := make(map[string]int)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		name := strings.TrimSpace(kv[0])
		if name == "" {
			continue
		}
		w := 1
		if len(kv) == 2 {
			if i, err := strconv.Atoi(strings.TrimSpace(kv[1])); err == nil && i > 0 {
				w = i
			}
		}
		res[name] = w
	}
	return res
}

package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a pgx connection pool using the provided DSN and verifies the connection with a ping.
// Example DSN formats supported:
//   - postgres://user:pass@host:port/dbname?sslmode=disable
//   - postgresql://user:pass@host:port/dbname
//   - postgresql+asyncpg://user:pass@host:port/dbname  ("+asyncpg" will be normalized)
func Connect(ctx context.Context, dsn string, opts ...func(*pgxpool.Config)) (*pgxpool.Pool, error) {
	normalized := normalizeDSN(dsn)

	cfg, err := pgxpool.ParseConfig(normalized)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	// Apply optional functional options
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	// Provide sensible defaults if the caller didn't override them
	if cfg.MinConns == 0 {
		cfg.MinConns = 0
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 4
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 5 * time.Minute
	}
	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = 60 * time.Minute
	}
	if cfg.HealthCheckPeriod == 0 {
		cfg.HealthCheckPeriod = 1 * time.Minute
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: new pool: %w", err)
	}

	// Verify connectivity right away
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return pool, nil
}

// NewPoolFromEnv loads the DSN from the DB_URL environment variable and creates a pgx pool.
// It supports normalizing common non-pgx DSN prefixes used in other ecosystems (e.g., "+asyncpg").
func NewPoolFromEnv(ctx context.Context, opts ...func(*pgxpool.Config)) (*pgxpool.Pool, error) {
	dsn := strings.TrimSpace(os.Getenv("DB_URL"))
	if dsn == "" {
		return nil, errors.New("postgres: DB_URL environment variable is not set")
	}
	return Connect(ctx, dsn, opts...)
}

// normalizeDSN converts known non-pgx DSN variants to a pgx-compatible DSN.
func normalizeDSN(dsn string) string {
	s := strings.TrimSpace(dsn)
	if s == "" {
		return s
	}
	// Normalize SQLAlchemy-style driver suffixes often found in .env files
	// e.g., postgresql+asyncpg:// -> postgresql://
	s = strings.Replace(s, "postgresql+asyncpg://", "postgresql://", 1)
	s = strings.Replace(s, "postgres+asyncpg://", "postgres://", 1)
	s = strings.Replace(s, "postgresql+pgx://", "postgresql://", 1)
	s = strings.Replace(s, "postgres+pgx://", "postgres://", 1)

	// pgx accepts both postgres:// and postgresql://, so no further changes required
	return s
}

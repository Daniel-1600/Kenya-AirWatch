package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
	"time"
)

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

// OpenWithRetry waits for a newly provisioned database to begin accepting
// connections. A non-positive timeout preserves Open's fail-fast behavior.
func OpenWithRetry(ctx context.Context, url string, timeout time.Duration) (*pgxpool.Pool, error) {
	if timeout <= 0 {
		return Open(ctx, url)
	}

	retryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	delay := time.Second
	var lastErr error
	for {
		pool, err := Open(retryCtx, url)
		if err == nil {
			return pool, nil
		}
		lastErr = err

		timer := time.NewTimer(delay)
		select {
		case <-retryCtx.Done():
			timer.Stop()
			return nil, fmt.Errorf("database unavailable after %s: %w", timeout, lastErr)
		case <-timer.C:
		}
		if delay < 5*time.Second {
			delay *= 2
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
		}
	}
}

func Migrate(ctx context.Context, p *pgxpool.Pool) error {
	b, err := os.ReadFile(filepath.Join("migrations", "001_init.sql"))
	if err != nil {
		return err
	}
	_, err = p.Exec(ctx, string(b))
	return err
}

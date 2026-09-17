package database

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
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
func Migrate(ctx context.Context, p *pgxpool.Pool) error {
	b, err := os.ReadFile(filepath.Join("migrations", "001_init.sql"))
	if err != nil {
		return err
	}
	_, err = p.Exec(ctx, string(b))
	return err
}

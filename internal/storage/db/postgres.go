package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	dbPool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, storagePath string) (*Postgres, error) {
	dbPool, err := pgxpool.New(ctx, storagePath)
	if err != nil {
		return nil, err
	}

	return &Postgres{
		dbPool: dbPool,
	}, nil
}

func (p *Postgres) Close() {
	p.dbPool.Close()
}

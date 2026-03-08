package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	projectmigrations "github.com/ievseev/url-shortener/migrations"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

type Postgres struct {
	dbPool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	dbPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, err
	}

	return &Postgres{
		dbPool: dbPool,
	}, nil
}

func (p *Postgres) Close() {
	p.dbPool.Close()
}

func runMigrations(dsn string) error {
	sourceDriver, err := iofs.New(projectmigrations.FS, ".")
	if err != nil {
		return fmt.Errorf("init migration source: %w", err)
	}

	migrator, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		sourceErr, databaseErr := migrator.Close()
		return errors.Join(err, sourceErr, databaseErr)
	}

	sourceErr, databaseErr := migrator.Close()
	if sourceErr != nil || databaseErr != nil {
		return errors.Join(sourceErr, databaseErr)
	}

	return nil
}

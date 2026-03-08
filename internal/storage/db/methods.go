package db

import (
	"context"
	"database/sql"
	"errors"

	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
)

func (p *Postgres) Ping(ctx context.Context) error {
	err := p.dbPool.Ping(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Postgres) SaveURLPair(ctx context.Context, urlOrigin, urlShort string) error {
	_, err := p.dbPool.Exec(
		ctx,
		`
			INSERT INTO short_urls (short_url, original_url)
			VALUES ($1, $2)
			ON CONFLICT (short_url) DO UPDATE
			SET original_url = EXCLUDED.original_url
		`,
		urlShort,
		urlOrigin,
	)
	if err != nil {
		return err
	}

	return nil
}

func (p *Postgres) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	var originalURL string

	err := p.dbPool.QueryRow(
		ctx,
		`
			SELECT original_url
			FROM short_urls
			WHERE short_url = $1
		`,
		urlShort,
	).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", URLRepo.ErrOriginURLNotFound
		}

		return "", err
	}

	return originalURL, nil
}

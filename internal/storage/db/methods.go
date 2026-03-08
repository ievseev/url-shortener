package db

import (
	"context"
	"errors"
	"fmt"

	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) Ping(ctx context.Context) error {
	err := p.dbPool.Ping(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Postgres) SaveURL(ctx context.Context, urlOrigin, shortURLBase string) (string, error) {
	tx, err := p.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	shortURL, err := saveURLTx(ctx, tx, urlOrigin, shortURLBase)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return shortURL, nil
}

func (p *Postgres) SaveURLBatch(
	ctx context.Context,
	urlOrigins, shortURLBases []string,
) ([]string, error) {
	if len(urlOrigins) != len(shortURLBases) {
		return nil, errors.New("url origins and short URL bases length mismatch")
	}

	tx, err := p.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	shortURLs := make([]string, len(urlOrigins))
	for i := range urlOrigins {
		shortURLs[i], err = saveURLTx(ctx, tx, urlOrigins[i], shortURLBases[i])
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return shortURLs, nil
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
		if errors.Is(err, pgx.ErrNoRows) {
			return "", URLRepo.ErrOriginURLNotFound
		}

		return "", err
	}

	return originalURL, nil
}

func saveURLTx(
	ctx context.Context,
	tx pgx.Tx,
	urlOrigin, shortURLBase string,
) (string, error) {
	shortURL := shortURLBase

	for counter := 0; ; counter++ {
		if counter > 0 {
			shortURL = fmt.Sprintf("%s_%d", shortURLBase, counter)
		}

		commandTag, err := tx.Exec(
			ctx,
			`
				INSERT INTO short_urls (short_url, original_url)
				VALUES ($1, $2)
				ON CONFLICT (short_url) DO NOTHING
			`,
			shortURL,
			urlOrigin,
		)
		if err != nil {
			return "", err
		}

		if commandTag.RowsAffected() == 1 {
			return shortURL, nil
		}

		var existingURL string
		err = tx.QueryRow(
			ctx,
			`
				SELECT original_url
				FROM short_urls
				WHERE short_url = $1
			`,
			shortURL,
		).Scan(&existingURL)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}

			return "", err
		}

		if existingURL == urlOrigin {
			return shortURL, nil
		}
	}
}

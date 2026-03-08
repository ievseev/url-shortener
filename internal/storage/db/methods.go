package db

import (
	"context"
	"errors"
	"fmt"

	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type urlQueryer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

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
	if err != nil && !errors.Is(err, URLRepo.ErrOriginalURLConflict) {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return shortURL, err
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

	shortURLs, err := saveURLBatchTx(ctx, tx, urlOrigins, shortURLBases)
	if err != nil {
		return nil, err
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

func saveURLBatchTx(
	ctx context.Context,
	queryer urlQueryer,
	urlOrigins, shortURLBases []string,
) ([]string, error) {
	shortURLs := make([]string, len(urlOrigins))
	for i := range urlOrigins {
		shortURL, err := saveURLTx(ctx, queryer, urlOrigins[i], shortURLBases[i])
		if err != nil && !errors.Is(err, URLRepo.ErrOriginalURLConflict) {
			return nil, err
		}

		shortURLs[i] = shortURL
	}

	return shortURLs, nil
}

func saveURLTx(
	ctx context.Context,
	queryer urlQueryer,
	urlOrigin, shortURLBase string,
) (string, error) {
	shortURL := shortURLBase

	for counter := 0; ; counter++ {
		if counter > 0 {
			shortURL = fmt.Sprintf("%s_%d", shortURLBase, counter)
		}

		commandTag, err := queryer.Exec(
			ctx,
			`
				INSERT INTO short_urls (short_url, original_url)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
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
		err = queryer.QueryRow(
			ctx,
			`
				SELECT original_url
				FROM short_urls
				WHERE short_url = $1
			`,
			shortURL,
		).Scan(&existingURL)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return "", err
			}
		} else if existingURL == urlOrigin {
			return shortURL, URLRepo.ErrOriginalURLConflict
		} else {
			continue
		}

		existingShortURL, err := getShortURLByOriginalTx(ctx, queryer, urlOrigin)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}

			return "", err
		}

		return existingShortURL, URLRepo.ErrOriginalURLConflict
	}
}

func getShortURLByOriginalTx(ctx context.Context, queryer urlQueryer, urlOrigin string) (string, error) {
	var shortURL string

	err := queryer.QueryRow(
		ctx,
		`
			SELECT short_url
			FROM short_urls
			WHERE original_url = $1
		`,
		urlOrigin,
	).Scan(&shortURL)
	if err != nil {
		return "", err
	}

	return shortURL, nil
}

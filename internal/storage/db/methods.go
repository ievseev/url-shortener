package db

import (
	"context"
	"errors"
	"fmt"

	urlrepo "github.com/ievseev/url-shortener/internal/repository/url"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type urlQueryer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

const (
	insertURLExecQuery = `
		INSERT INTO short_urls (short_url, original_url)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	insertURLBatchQuery = `
		INSERT INTO short_urls (short_url, original_url)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		RETURNING short_url
	`
	selectByShortURLQuery = `
		SELECT short_url, original_url
		FROM short_urls
		WHERE short_url = $1
	`
	selectByOriginalURLQuery = `
		SELECT short_url
		FROM short_urls
		WHERE original_url = $1
	`
)

type shortURLRecord struct {
	shortURL    string
	originalURL string
}

var errBatchInsertConflict = errors.New("batch insert conflict")

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
	if err != nil && !errors.Is(err, urlrepo.ErrOriginalURLConflict) {
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
			return "", urlrepo.ErrOriginURLNotFound
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
	if len(urlOrigins) == 0 {
		return []string{}, nil
	}

	return tryInsertBatch(ctx, queryer, urlOrigins, shortURLBases)
}

func saveURLTx(
	ctx context.Context,
	queryer urlQueryer,
	urlOrigin, shortURLBase string,
) (string, error) {
	for counter := 0; ; counter++ {
		shortURL := makeShortURLCandidate(shortURLBase, counter)

		inserted, err := tryInsertURL(ctx, queryer, shortURL, urlOrigin)
		if err != nil {
			return "", err
		}
		if inserted {
			return shortURL, nil
		}

		existingRecord, found, err := getShortURLRecordByShortTx(ctx, queryer, shortURL)
		if err != nil {
			return "", err
		}
		if found {
			if existingRecord.originalURL == urlOrigin {
				return existingRecord.shortURL, urlrepo.ErrOriginalURLConflict
			}

			continue
		}

		existingShortURL, err := getShortURLByOriginalTx(ctx, queryer, urlOrigin)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}

			return "", err
		}

		return existingShortURL, urlrepo.ErrOriginalURLConflict
	}
}

func getShortURLByOriginalTx(ctx context.Context, queryer urlQueryer, urlOrigin string) (string, error) {
	shortURL, found, err := scanShortURL(queryer.QueryRow(
		ctx,
		selectByOriginalURLQuery,
		urlOrigin,
	))
	if err != nil {
		return "", err
	}
	if !found {
		return "", pgx.ErrNoRows
	}

	return shortURL, nil
}

func tryInsertURL(ctx context.Context, queryer urlQueryer, shortURL, urlOrigin string) (bool, error) {
	commandTag, err := queryer.Exec(
		ctx,
		insertURLExecQuery,
		shortURL,
		urlOrigin,
	)
	if err != nil {
		return false, err
	}

	return commandTag.RowsAffected() == 1, nil
}

func getShortURLRecordByShortTx(
	ctx context.Context,
	queryer urlQueryer,
	shortURL string,
) (shortURLRecord, bool, error) {
	return scanShortURLRecord(queryer.QueryRow(ctx, selectByShortURLQuery, shortURL))
}

func tryInsertBatch(
	ctx context.Context,
	queryer urlQueryer,
	urlOrigins, shortURLBases []string,
) ([]string, error) {
	if len(urlOrigins) == 0 {
		return []string{}, nil
	}

	batch := &pgx.Batch{}
	for i := range urlOrigins {
		batch.Queue(insertURLBatchQuery, shortURLBases[i], urlOrigins[i])
	}

	results := queryer.SendBatch(ctx, batch)
	shortURLs := make([]string, len(urlOrigins))
	hasConflict := false

	for i := range urlOrigins {
		shortURL, found, err := scanShortURL(results.QueryRow())
		if err != nil {
			results.Close()
			return nil, err
		}

		if !found {
			hasConflict = true
			continue
		}

		shortURLs[i] = shortURL
	}

	if err := results.Close(); err != nil {
		return nil, err
	}

	if hasConflict {
		return nil, errBatchInsertConflict
	}

	return shortURLs, nil
}

func scanShortURLRecord(row pgx.Row) (shortURLRecord, bool, error) {
	var record shortURLRecord

	err := row.Scan(&record.shortURL, &record.originalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shortURLRecord{}, false, nil
		}

		return shortURLRecord{}, false, err
	}

	return record, true, nil
}

func scanShortURL(row pgx.Row) (string, bool, error) {
	var shortURL string

	err := row.Scan(&shortURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}

		return "", false, err
	}

	return shortURL, true, nil
}

func makeShortURLCandidate(shortURLBase string, counter int) string {
	if counter == 0 {
		return shortURLBase
	}

	return fmt.Sprintf("%s_%d", shortURLBase, counter)
}

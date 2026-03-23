package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/ievseev/url-shortener/internal/model"
	urlrepo "github.com/ievseev/url-shortener/internal/repository/url"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type urlQueryer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const (
	insertURLExecQuery = `
		INSERT INTO short_urls (short_url, original_url)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	selectByShortURLQuery = `
		SELECT id, short_url, original_url
		FROM short_urls
		WHERE short_url = $1
	`
	selectByOriginalURLQuery = `
		SELECT id, short_url, original_url
		FROM short_urls
		WHERE original_url = $1
	`
	insertUserURLQuery = `
		INSERT INTO user_urls (user_id, url_id, is_creator)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`
	selectOriginURLQuery = `
		SELECT original_url, is_deleted
		FROM short_urls
		WHERE short_url = $1
	`
	selectUserURLsQuery = `
		SELECT su.short_url, su.original_url
		FROM user_urls uu
		JOIN short_urls su ON su.id = uu.url_id
		WHERE uu.user_id = $1
		ORDER BY uu.id
	`
	deleteUserURLsQuery = `
		UPDATE short_urls AS su
		SET is_deleted = TRUE
		FROM user_urls AS uu
		WHERE su.id = uu.url_id
			AND uu.user_id = $1
			AND uu.is_creator = TRUE
			AND su.short_url = ANY($2)
			AND su.is_deleted = FALSE
	`
)

type urlRecord struct {
	id          int64
	shortURL    string
	originalURL string
}

func (p *Postgres) Ping(ctx context.Context) error {
	err := p.dbPool.Ping(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Postgres) SaveURL(ctx context.Context, userID, urlOrigin, shortURLBase string) (string, error) {
	tx, err := p.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	shortURL, err := saveURLTx(ctx, tx, userID, urlOrigin, shortURLBase)
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
	userID string,
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

	shortURLs, err := saveURLBatchTx(ctx, tx, userID, urlOrigins, shortURLBases)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return shortURLs, nil
}

func (p *Postgres) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	return getOriginURLTx(ctx, p.dbPool, urlShort)
}

func (p *Postgres) GetUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	rows, err := p.dbPool.Query(ctx, selectUserURLsQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.UserURL
	for rows.Next() {
		var userURL model.UserURL
		if err := rows.Scan(&userURL.ShortURL, &userURL.OriginalURL); err != nil {
			return nil, err
		}

		result = append(result, userURL)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (p *Postgres) DeleteUserURLs(ctx context.Context, userID string, shortURLs []string) error {
	return deleteUserURLsTx(ctx, p.dbPool, userID, shortURLs)
}

func saveURLBatchTx(
	ctx context.Context,
	queryer urlQueryer,
	userID string,
	urlOrigins, shortURLBases []string,
) ([]string, error) {
	if len(urlOrigins) == 0 {
		return []string{}, nil
	}

	shortURLs := make([]string, len(urlOrigins))
	for i := range urlOrigins {
		shortURL, err := saveURLTx(ctx, queryer, userID, urlOrigins[i], shortURLBases[i])
		if err != nil && !errors.Is(err, urlrepo.ErrOriginalURLConflict) {
			return nil, err
		}

		shortURLs[i] = shortURL
	}

	return shortURLs, nil
}

func saveURLTx(
	ctx context.Context,
	queryer urlQueryer,
	userID, urlOrigin, shortURLBase string,
) (string, error) {
	for counter := 0; ; counter++ {
		shortURL := makeShortURLCandidate(shortURLBase, counter)

		record, inserted, err := tryInsertURL(ctx, queryer, shortURL, urlOrigin)
		if err != nil {
			return "", err
		}
		if inserted {
			if err := ensureUserURLTx(ctx, queryer, userID, record.id, true); err != nil {
				return "", err
			}

			return record.shortURL, nil
		}

		existingRecord, found, err := getURLRecordByShortTx(ctx, queryer, shortURL)
		if err != nil {
			return "", err
		}
		if found {
			if existingRecord.originalURL == urlOrigin {
				if err := ensureUserURLTx(ctx, queryer, userID, existingRecord.id, false); err != nil {
					return "", err
				}

				return existingRecord.shortURL, urlrepo.ErrOriginalURLConflict
			}

			continue
		}

		existingRecord, found, err = getURLRecordByOriginalTx(ctx, queryer, urlOrigin)
		if err != nil {
			return "", err
		}
		if !found {
			continue
		}

		if err := ensureUserURLTx(ctx, queryer, userID, existingRecord.id, false); err != nil {
			return "", err
		}

		return existingRecord.shortURL, urlrepo.ErrOriginalURLConflict
	}
}

func tryInsertURL(ctx context.Context, queryer urlQueryer, shortURL, urlOrigin string) (urlRecord, bool, error) {
	commandTag, err := queryer.Exec(
		ctx,
		insertURLExecQuery,
		shortURL,
		urlOrigin,
	)
	if err != nil {
		return urlRecord{}, false, err
	}

	if commandTag.RowsAffected() != 1 {
		return urlRecord{}, false, nil
	}

	record, found, err := getURLRecordByShortTx(ctx, queryer, shortURL)
	if err != nil {
		return urlRecord{}, false, err
	}
	if !found {
		return urlRecord{}, false, errors.New("inserted short URL record not found")
	}

	return record, true, nil
}

func getURLRecordByShortTx(
	ctx context.Context,
	queryer urlQueryer,
	shortURL string,
) (urlRecord, bool, error) {
	return scanURLRecord(queryer.QueryRow(ctx, selectByShortURLQuery, shortURL))
}

func getURLRecordByOriginalTx(
	ctx context.Context,
	queryer urlQueryer,
	urlOrigin string,
) (urlRecord, bool, error) {
	return scanURLRecord(queryer.QueryRow(ctx, selectByOriginalURLQuery, urlOrigin))
}

func ensureUserURLTx(ctx context.Context, queryer urlQueryer, userID string, urlID int64, isCreator bool) error {
	_, err := queryer.Exec(ctx, insertUserURLQuery, userID, urlID, isCreator)
	return err
}

func getOriginURLTx(ctx context.Context, queryer urlQueryer, urlShort string) (string, error) {
	var (
		originalURL string
		isDeleted   bool
	)

	err := queryer.QueryRow(ctx, selectOriginURLQuery, urlShort).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", urlrepo.ErrOriginURLNotFound
		}

		return "", err
	}

	if isDeleted {
		return "", urlrepo.ErrOriginURLDeleted
	}

	return originalURL, nil
}

func deleteUserURLsTx(ctx context.Context, queryer urlQueryer, userID string, shortURLs []string) error {
	if len(shortURLs) == 0 {
		return nil
	}

	_, err := queryer.Exec(ctx, deleteUserURLsQuery, userID, shortURLs)
	return err
}

func scanURLRecord(row pgx.Row) (urlRecord, bool, error) {
	var record urlRecord

	err := row.Scan(&record.id, &record.shortURL, &record.originalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return urlRecord{}, false, nil
		}

		return urlRecord{}, false, err
	}

	return record, true, nil
}

func makeShortURLCandidate(shortURLBase string, counter int) string {
	if counter == 0 {
		return shortURLBase
	}

	return fmt.Sprintf("%s_%d", shortURLBase, counter)
}

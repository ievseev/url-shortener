package db

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	urlrepo "github.com/ievseev/url-shortener/internal/repository/url"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSaveURLTxCreatesUserOwnershipForNewURL(t *testing.T) {
	queryer := &scriptedQueryer{
		t: t,
		execs: []scriptedExec{
			{
				sqlContains: "INSERT INTO short_urls",
				args:        []any{"abc123", "https://example.com"},
				tag:         pgconn.NewCommandTag("INSERT 0 1"),
			},
			{
				sqlContains: "INSERT INTO user_urls",
				args:        []any{"user-1", int64(42), true},
				tag:         pgconn.NewCommandTag("INSERT 0 1"),
			},
		},
		queries: []scriptedQuery{
			{
				sqlContains: "SELECT id, short_url, original_url",
				args:        []any{"abc123"},
				values:      []any{int64(42), "abc123", "https://example.com"},
			},
		},
	}

	shortURL, err := saveURLTx(context.Background(), queryer, "user-1", "https://example.com", "abc123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if shortURL != "abc123" {
		t.Fatalf("expected short URL %q, got %q", "abc123", shortURL)
	}

	queryer.assertDone()
}

func TestSaveURLTxReturnsExistingShortURLByOriginalLookup(t *testing.T) {
	queryer := &scriptedQueryer{
		t: t,
		execs: []scriptedExec{
			{
				sqlContains: "INSERT INTO short_urls",
				args:        []any{"abc123", "https://example.com"},
				tag:         pgconn.NewCommandTag("INSERT 0 0"),
			},
			{
				sqlContains: "INSERT INTO user_urls",
				args:        []any{"user-2", int64(7), false},
				tag:         pgconn.NewCommandTag("INSERT 0 1"),
			},
		},
		queries: []scriptedQuery{
			{
				sqlContains: "SELECT id, short_url, original_url",
				args:        []any{"abc123"},
				err:         pgx.ErrNoRows,
			},
			{
				sqlContains: "SELECT id, short_url, original_url",
				args:        []any{"https://example.com"},
				values:      []any{int64(7), "abc123_1", "https://example.com"},
			},
		},
	}

	shortURL, err := saveURLTx(context.Background(), queryer, "user-2", "https://example.com", "abc123")
	if !errors.Is(err, urlrepo.ErrOriginalURLConflict) {
		t.Fatalf("expected original URL conflict, got %v", err)
	}

	if shortURL != "abc123_1" {
		t.Fatalf("expected existing short URL %q, got %q", "abc123_1", shortURL)
	}

	queryer.assertDone()
}

func TestSaveURLBatchTxReturnsShortURLsInInputOrder(t *testing.T) {
	queryer := &scriptedQueryer{
		t: t,
		execs: []scriptedExec{
			{
				sqlContains: "INSERT INTO short_urls",
				args:        []any{"abc123", "https://example.com/1"},
				tag:         pgconn.NewCommandTag("INSERT 0 1"),
			},
			{
				sqlContains: "INSERT INTO user_urls",
				args:        []any{"user-1", int64(1), true},
				tag:         pgconn.NewCommandTag("INSERT 0 1"),
			},
			{
				sqlContains: "INSERT INTO short_urls",
				args:        []any{"abc123", "https://example.com/1"},
				tag:         pgconn.NewCommandTag("INSERT 0 0"),
			},
			{
				sqlContains: "INSERT INTO user_urls",
				args:        []any{"user-1", int64(1), false},
				tag:         pgconn.NewCommandTag("INSERT 0 0"),
			},
		},
		queries: []scriptedQuery{
			{
				sqlContains: "SELECT id, short_url, original_url",
				args:        []any{"abc123"},
				values:      []any{int64(1), "abc123", "https://example.com/1"},
			},
			{
				sqlContains: "SELECT id, short_url, original_url",
				args:        []any{"abc123"},
				values:      []any{int64(1), "abc123", "https://example.com/1"},
			},
		},
	}

	shortURLs, err := saveURLBatchTx(
		context.Background(),
		queryer,
		"user-1",
		[]string{"https://example.com/1", "https://example.com/1"},
		[]string{"abc123", "abc123"},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	expected := []string{"abc123", "abc123"}
	if !reflect.DeepEqual(shortURLs, expected) {
		t.Fatalf("expected short URLs %v, got %v", expected, shortURLs)
	}

	queryer.assertDone()
}

func TestGetOriginURLTxReturnsDeletedError(t *testing.T) {
	queryer := &scriptedQueryer{
		t: t,
		queries: []scriptedQuery{
			{
				sqlContains: "SELECT original_url, is_deleted",
				args:        []any{"abc123"},
				values:      []any{"https://example.com", true},
			},
		},
	}

	_, err := getOriginURLTx(context.Background(), queryer, "abc123")
	if !errors.Is(err, urlrepo.ErrOriginURLDeleted) {
		t.Fatalf("expected deleted error, got %v", err)
	}

	queryer.assertDone()
}

func TestDeleteUserURLsTxUsesBatchUpdate(t *testing.T) {
	queryer := &scriptedQueryer{
		t: t,
		execs: []scriptedExec{
			{
				sqlContains: "UPDATE short_urls AS su",
				args:        []any{"user-1", []string{"abc123", "def456"}},
				tag:         pgconn.NewCommandTag("UPDATE 2"),
			},
		},
	}

	err := deleteUserURLsTx(context.Background(), queryer, "user-1", []string{"abc123", "def456"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	queryer.assertDone()
}

type scriptedExec struct {
	sqlContains string
	args        []any
	tag         pgconn.CommandTag
	err         error
}

type scriptedQuery struct {
	sqlContains string
	args        []any
	values      []any
	err         error
}

type scriptedQueryer struct {
	t       *testing.T
	execs   []scriptedExec
	queries []scriptedQuery
}

func (s *scriptedQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	s.t.Helper()

	if len(s.execs) == 0 {
		s.t.Fatalf("unexpected Exec call with sql %q", sql)
	}

	expected := s.execs[0]
	s.execs = s.execs[1:]

	if !strings.Contains(sql, expected.sqlContains) {
		s.t.Fatalf("expected Exec SQL to contain %q, got %q", expected.sqlContains, sql)
	}

	if !reflect.DeepEqual(arguments, expected.args) {
		s.t.Fatalf("expected Exec args %v, got %v", expected.args, arguments)
	}

	return expected.tag, expected.err
}

func (s *scriptedQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	s.t.Helper()
	s.t.Fatalf("unexpected Query call with sql %q", sql)
	return nil, nil
}

func (s *scriptedQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	s.t.Helper()

	if len(s.queries) == 0 {
		s.t.Fatalf("unexpected QueryRow call with sql %q", sql)
	}

	expected := s.queries[0]
	s.queries = s.queries[1:]

	if !strings.Contains(sql, expected.sqlContains) {
		s.t.Fatalf("expected QueryRow SQL to contain %q, got %q", expected.sqlContains, sql)
	}

	if !reflect.DeepEqual(args, expected.args) {
		s.t.Fatalf("expected QueryRow args %v, got %v", expected.args, args)
	}

	return scriptedRow{
		t:      s.t,
		values: expected.values,
		err:    expected.err,
	}
}

func (s *scriptedQueryer) assertDone() {
	s.t.Helper()

	if len(s.execs) != 0 {
		s.t.Fatalf("not all Exec expectations were used: %d remaining", len(s.execs))
	}

	if len(s.queries) != 0 {
		s.t.Fatalf("not all QueryRow expectations were used: %d remaining", len(s.queries))
	}
}

type scriptedRow struct {
	t      *testing.T
	values []any
	err    error
}

func (r scriptedRow) Scan(dest ...any) error {
	r.t.Helper()

	if r.err != nil {
		return r.err
	}

	if len(dest) != len(r.values) {
		r.t.Fatalf("expected %d Scan destinations, got %d", len(r.values), len(dest))
	}

	for i, destination := range dest {
		value := reflect.ValueOf(destination)
		if value.Kind() != reflect.Ptr || value.IsNil() {
			r.t.Fatalf("Scan destination at index %d must be a non-nil pointer", i)
		}

		value.Elem().Set(reflect.ValueOf(r.values[i]))
	}

	return nil
}

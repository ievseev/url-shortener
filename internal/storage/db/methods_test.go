package db

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSaveURLTxReturnsExistingShortURLByOriginalLookup(t *testing.T) {
	queryer := &scriptedQueryer{
		t: t,
		execs: []scriptedExec{
			{
				sqlContains: "INSERT INTO short_urls",
				args:        []any{"abc123", "https://example.com"},
				tag:         pgconn.NewCommandTag("INSERT 0 0"),
			},
		},
		queries: []scriptedQuery{
			{
				sqlContains: "SELECT original_url",
				args:        []any{"abc123"},
				err:         pgx.ErrNoRows,
			},
			{
				sqlContains: "SELECT short_url",
				args:        []any{"https://example.com"},
				values:      []any{"abc123_1"},
			},
		},
	}

	shortURL, err := saveURLTx(context.Background(), queryer, "https://example.com", "abc123")
	if !errors.Is(err, URLRepo.ErrOriginalURLConflict) {
		t.Fatalf("expected original URL conflict, got %v", err)
	}

	if shortURL != "abc123_1" {
		t.Fatalf("expected existing short URL %q, got %q", "abc123_1", shortURL)
	}

	queryer.assertDone()
}

func TestSaveURLBatchTxKeepsExistingShortURLOnDuplicate(t *testing.T) {
	queryer := &scriptedQueryer{
		t: t,
		execs: []scriptedExec{
			{
				sqlContains: "INSERT INTO short_urls",
				args:        []any{"abc123", "https://example.com"},
				tag:         pgconn.NewCommandTag("INSERT 0 1"),
			},
			{
				sqlContains: "INSERT INTO short_urls",
				args:        []any{"abc123", "https://example.com"},
				tag:         pgconn.NewCommandTag("INSERT 0 0"),
			},
		},
		queries: []scriptedQuery{
			{
				sqlContains: "SELECT original_url",
				args:        []any{"abc123"},
				values:      []any{"https://example.com"},
			},
		},
	}

	shortURLs, err := saveURLBatchTx(
		context.Background(),
		queryer,
		[]string{"https://example.com", "https://example.com"},
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

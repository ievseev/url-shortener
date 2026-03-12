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
				sqlContains: "SELECT short_url, original_url",
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
	if !errors.Is(err, urlrepo.ErrOriginalURLConflict) {
		t.Fatalf("expected original URL conflict, got %v", err)
	}

	if shortURL != "abc123_1" {
		t.Fatalf("expected existing short URL %q, got %q", "abc123_1", shortURL)
	}

	queryer.assertDone()
}

func TestSaveURLBatchTxFailsOnDuplicateOriginalURL(t *testing.T) {
	queryer := &scriptedBatchQueryer{
		t: t,
		batches: []scriptedBatch{
			{
				queries: []scriptedBatchQuery{
					{
						sqlContains: "INSERT INTO short_urls",
						args:        []any{"abc123", "https://example.com"},
						values:      []any{"abc123"},
					},
					{
						sqlContains: "INSERT INTO short_urls",
						args:        []any{"abc123", "https://example.com"},
						err:         pgx.ErrNoRows,
					},
				},
			},
		},
	}

	_, err := saveURLBatchTx(
		context.Background(),
		queryer,
		[]string{"https://example.com", "https://example.com"},
		[]string{"abc123", "abc123"},
	)
	if !errors.Is(err, errBatchInsertConflict) {
		t.Fatalf("expected batch insert conflict, got %v", err)
	}

	queryer.assertDone()
}

func TestSaveURLBatchTxReturnsCreatedShortURLsInInputOrder(t *testing.T) {
	queryer := &scriptedBatchQueryer{
		t: t,
		batches: []scriptedBatch{
			{
				queries: []scriptedBatchQuery{
					{
						sqlContains: "INSERT INTO short_urls",
						args:        []any{"abc123", "https://example.com/1"},
						values:      []any{"abc123"},
					},
					{
						sqlContains: "INSERT INTO short_urls",
						args:        []any{"def456", "https://example.com/2"},
						values:      []any{"def456"},
					},
				},
			},
		},
	}

	shortURLs, err := saveURLBatchTx(
		context.Background(),
		queryer,
		[]string{"https://example.com/1", "https://example.com/2"},
		[]string{"abc123", "def456"},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	expected := []string{"abc123", "def456"}
	if !reflect.DeepEqual(shortURLs, expected) {
		t.Fatalf("expected short URLs %v, got %v", expected, shortURLs)
	}

	queryer.assertDone()
}

func TestSaveURLBatchTxFailsOnShortURLCollision(t *testing.T) {
	queryer := &scriptedBatchQueryer{
		t: t,
		batches: []scriptedBatch{
			{
				queries: []scriptedBatchQuery{
					{
						sqlContains: "INSERT INTO short_urls",
						args:        []any{"abc123", "https://example.com/1"},
						values:      []any{"abc123"},
					},
					{
						sqlContains: "INSERT INTO short_urls",
						args:        []any{"abc123", "https://example.com/2"},
						err:         pgx.ErrNoRows,
					},
				},
			},
		},
	}

	_, err := saveURLBatchTx(
		context.Background(),
		queryer,
		[]string{"https://example.com/1", "https://example.com/2"},
		[]string{"abc123", "abc123"},
	)
	if !errors.Is(err, errBatchInsertConflict) {
		t.Fatalf("expected batch insert conflict, got %v", err)
	}

	queryer.assertDone()
}

func TestSaveURLBatchTxFailsOnExistingOriginalURL(t *testing.T) {
	queryer := &scriptedBatchQueryer{
		t: t,
		batches: []scriptedBatch{
			{
				queries: []scriptedBatchQuery{
					{
						sqlContains: "INSERT INTO short_urls",
						args:        []any{"abc123", "https://example.com"},
						err:         pgx.ErrNoRows,
					},
				},
			},
		},
	}

	_, err := saveURLBatchTx(
		context.Background(),
		queryer,
		[]string{"https://example.com"},
		[]string{"abc123"},
	)
	if !errors.Is(err, errBatchInsertConflict) {
		t.Fatalf("expected batch insert conflict, got %v", err)
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

func (s *scriptedQueryer) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	s.t.Helper()
	s.t.Fatalf("unexpected SendBatch call with %d queued queries", b.Len())
	return nil
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

type scriptedBatchQuery struct {
	sqlContains string
	args        []any
	values      []any
	err         error
}

type scriptedBatch struct {
	queries  []scriptedBatchQuery
	closeErr error
}

type scriptedBatchQueryer struct {
	t       *testing.T
	batches []scriptedBatch
}

func (s *scriptedBatchQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	s.t.Helper()
	s.t.Fatalf("unexpected Exec call with sql %q", sql)
	return pgconn.CommandTag{}, nil
}

func (s *scriptedBatchQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	s.t.Helper()
	s.t.Fatalf("unexpected QueryRow call with sql %q", sql)
	return nil
}

func (s *scriptedBatchQueryer) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	s.t.Helper()

	if len(s.batches) == 0 {
		s.t.Fatalf("unexpected SendBatch call with %d queued queries", b.Len())
	}

	expected := s.batches[0]
	s.batches = s.batches[1:]

	if len(expected.queries) != b.Len() {
		s.t.Fatalf("expected %d batched queries, got %d", len(expected.queries), b.Len())
	}

	for i, query := range expected.queries {
		queued := b.QueuedQueries[i]
		if !strings.Contains(queued.SQL, query.sqlContains) {
			s.t.Fatalf("expected batched SQL to contain %q, got %q", query.sqlContains, queued.SQL)
		}

		if !reflect.DeepEqual(queued.Arguments, query.args) {
			s.t.Fatalf("expected batched args %v, got %v", query.args, queued.Arguments)
		}
	}

	return &scriptedBatchResults{
		t:        s.t,
		queries:  expected.queries,
		closeErr: expected.closeErr,
	}
}

func (s *scriptedBatchQueryer) assertDone() {
	s.t.Helper()

	if len(s.batches) != 0 {
		s.t.Fatalf("not all batch expectations were used: %d remaining", len(s.batches))
	}
}

type scriptedBatchResults struct {
	t        *testing.T
	queries  []scriptedBatchQuery
	closeErr error
	index    int
}

func (r *scriptedBatchResults) Exec() (pgconn.CommandTag, error) {
	r.t.Helper()
	r.t.Fatal("unexpected batch Exec call")
	return pgconn.CommandTag{}, nil
}

func (r *scriptedBatchResults) Query() (pgx.Rows, error) {
	r.t.Helper()
	r.t.Fatal("unexpected batch Query call")
	return nil, nil
}

func (r *scriptedBatchResults) QueryRow() pgx.Row {
	r.t.Helper()

	if r.index >= len(r.queries) {
		r.t.Fatal("unexpected batch QueryRow call")
	}

	query := r.queries[r.index]
	r.index++

	return scriptedRow{
		t:      r.t,
		values: query.values,
		err:    query.err,
	}
}

func (r *scriptedBatchResults) Close() error {
	r.t.Helper()

	if r.index != len(r.queries) {
		r.t.Fatalf("not all batch results were read: %d unread", len(r.queries)-r.index)
	}

	return r.closeErr
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

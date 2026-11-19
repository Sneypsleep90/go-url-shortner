//go:build integration

package postgres_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
	"github.com/Sneypsleep90/go-url-shortner/internal/storage/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCreateAndGet(t *testing.T) {
	t.Parallel()

	store := postgres.New(newTestPool(t))
	link := domain.Link{ShortCode: "ABcd1234_5", OriginalURL: "https://example.com/path?q=go#part"}
	created, err := store.Create(t.Context(), link)
	if err != nil {
		t.Fatal(err)
	}
	if created != link {
		t.Fatalf("Create() = %+v, want %+v", created, link)
	}
	found, err := store.GetByCode(t.Context(), link.ShortCode)
	if err != nil {
		t.Fatal(err)
	}
	if found != link {
		t.Fatalf("GetByCode() = %+v, want %+v", found, link)
	}
}

func TestCreateExistingURL(t *testing.T) {
	t.Parallel()

	store := postgres.New(newTestPool(t))
	link := domain.Link{ShortCode: "abcdefghij", OriginalURL: "https://example.com"}
	if _, err := store.Create(t.Context(), link); err != nil {
		t.Fatal(err)
	}

	duplicate := domain.Link{ShortCode: "klmnopqrst", OriginalURL: link.OriginalURL}
	created, err := store.Create(t.Context(), duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if created != link {
		t.Fatalf("Create() = %+v, want existing link %+v", created, link)
	}
	found, err := store.GetByCode(t.Context(), link.ShortCode)
	if err != nil {
		t.Fatal(err)
	}
	if found != link {
		t.Fatalf("stored link changed to %+v", found)
	}
	if _, err := store.GetByCode(t.Context(), duplicate.ShortCode); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("new code lookup error = %v, want ErrNotFound", err)
	}
}

func TestCreateTakenCode(t *testing.T) {
	t.Parallel()

	store := postgres.New(newTestPool(t))
	link := domain.Link{ShortCode: "abcdefghij", OriginalURL: "https://example.com"}
	if _, err := store.Create(t.Context(), link); err != nil {
		t.Fatal(err)
	}

	other := domain.Link{ShortCode: link.ShortCode, OriginalURL: "https://example.org"}
	if _, err := store.Create(t.Context(), other); !errors.Is(err, domain.ErrCodeTaken) {
		t.Fatalf("Create() error = %v, want ErrCodeTaken", err)
	}
	found, err := store.GetByCode(t.Context(), link.ShortCode)
	if err != nil {
		t.Fatal(err)
	}
	if found != link {
		t.Fatalf("stored link changed to %+v", found)
	}
}

func TestGetByCodeNotFound(t *testing.T) {
	t.Parallel()

	store := postgres.New(newTestPool(t))
	if _, err := store.GetByCode(t.Context(), "abcdefghij"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByCode() error = %v, want ErrNotFound", err)
	}
}

func TestCreateConcurrent(t *testing.T) {
	t.Parallel()

	const workers = 20
	type result struct {
		link domain.Link
		err  error
	}
	store := postgres.New(newTestPool(t))
	ctx := t.Context()
	results := make(chan result, workers)
	start := make(chan struct{})
	for i := range workers {
		go func() {
			<-start
			link, err := store.Create(ctx, domain.Link{
				ShortCode:   fmt.Sprintf("%010d", i),
				OriginalURL: "https://example.com",
			})
			results <- result{link: link, err: err}
		}()
	}
	close(start)

	var first domain.Link
	for range workers {
		got := <-results
		if got.err != nil {
			t.Error(got.err)
			continue
		}
		if first.ShortCode == "" {
			first = got.link
		}
		if got.link != first {
			t.Errorf("same URL produced different links: %+v and %+v", first, got.link)
		}
	}
	if t.Failed() {
		return
	}
	found, err := store.GetByCode(ctx, first.ShortCode)
	if err != nil {
		t.Fatal(err)
	}
	if found != first || found.OriginalURL != "https://example.com" {
		t.Fatalf("unexpected stored link: %+v", found)
	}
}

func TestStorageErrors(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t)
	store := postgres.New(pool)
	if _, err := pool.Exec(t.Context(), "DROP TABLE links"); err != nil {
		t.Fatal(err)
	}
	t.Run("create", func(t *testing.T) {
		_, err := store.Create(t.Context(), domain.Link{ShortCode: "abcdefghij", OriginalURL: "https://example.com"})
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42P01" {
			t.Fatalf("Create() error = %v, want undefined table error", err)
		}
	})
	t.Run("get", func(t *testing.T) {
		_, err := store.GetByCode(t.Context(), "abcdefghij")
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42P01" {
			t.Fatalf("GetByCode() error = %v, want undefined table error", err)
		}
	})
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DB_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DB_URL is required for integration tests")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + strings.ToLower(rand.Text())
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
	})
	migration, err := os.ReadFile("../../../migrations/00001_create_links.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(migration), "-- +goose Down")
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatal(err)
	}
	return pool
}

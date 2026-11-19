package memory_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
	"github.com/Sneypsleep90/go-url-shortner/internal/storage/memory"
)

func TestCreateAndGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		link domain.Link
	}{
		{
			name: "lowercase code",
			link: domain.Link{ShortCode: "abcdefghij", OriginalURL: "https://example.com"},
		},
		{
			name: "mixed code",
			link: domain.Link{ShortCode: "ABcd1234_5", OriginalURL: "http://example.org:8080/path?q=go#part"},
		},
		{
			name: "encoded URL",
			link: domain.Link{ShortCode: "_012345678", OriginalURL: "https://example.net/a%20b"},
		},
	}
	store := memory.New()
	for _, tt := range tests {
		created, err := store.Create(t.Context(), tt.link)
		if err != nil {
			t.Fatal(err)
		}
		if created != tt.link {
			t.Fatalf("Create() = %+v, want %+v", created, tt.link)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			found, err := store.GetByCode(t.Context(), tt.link.ShortCode)
			if err != nil {
				t.Fatal(err)
			}
			if found != tt.link {
				t.Fatalf("GetByCode() = %+v, want %+v", found, tt.link)
			}
		})
	}
}

func TestCreateExistingURL(t *testing.T) {
	t.Parallel()

	store := memory.New()
	link := domain.Link{ShortCode: "abcdefghij", OriginalURL: "https://example.com"}
	if _, err := store.Create(t.Context(), link); err != nil {
		t.Fatal(err)
	}

	duplicate := domain.Link{ShortCode: "klmnopqrst", OriginalURL: link.OriginalURL}
	got, err := store.Create(t.Context(), duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if got != link {
		t.Fatalf("Create() = %+v, want existing link %+v", got, link)
	}
	if _, err := store.GetByCode(t.Context(), duplicate.ShortCode); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("new code lookup error = %v, want ErrNotFound", err)
	}
}

func TestCreateTakenCode(t *testing.T) {
	t.Parallel()

	store := memory.New()
	link := domain.Link{ShortCode: "abcdefghij", OriginalURL: "https://example.com/first"}
	if _, err := store.Create(t.Context(), link); err != nil {
		t.Fatal(err)
	}

	other := domain.Link{ShortCode: link.ShortCode, OriginalURL: "https://example.com/second"}
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

	_, err := memory.New().GetByCode(t.Context(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
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
	store := memory.New()
	ctx := t.Context()
	results := make(chan result, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			link, err := store.Create(ctx, domain.Link{
				ShortCode:   fmt.Sprintf("%010d", i),
				OriginalURL: "https://example.com",
			})
			results <- result{link: link, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var first domain.Link
	for got := range results {
		if got.err != nil {
			t.Fatal(got.err)
		}
		if first.ShortCode == "" {
			first = got.link
		}
		if got.link != first {
			t.Fatalf("same URL produced different links: %+v and %+v", first, got.link)
		}
	}
	found, err := store.GetByCode(ctx, first.ShortCode)
	if err != nil {
		t.Fatal(err)
	}
	if found != first {
		t.Fatalf("stored link = %+v, want %+v", found, first)
	}
}

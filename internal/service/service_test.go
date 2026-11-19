package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
	"github.com/Sneypsleep90/go-url-shortner/internal/service"
)

func TestCreateCodeRetriesCollision(t *testing.T) {
	t.Parallel()

	repo := &repoStub{collisions: 1}
	svc := service.NewService(repo)
	link, err := svc.CreateCode(t.Context(), domain.Link{OriginalURL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.calls != 2 {
		t.Fatalf("Create() calls = %d, want 2", repo.calls)
	}
	if link.ShortCode == "" || link.OriginalURL != "https://example.com" {
		t.Fatalf("unexpected link: %+v", link)
	}
}

func TestCreateCodeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantCalls int
	}{
		{name: "collision limit", err: domain.ErrCodeTaken, wantCalls: 5},
		{name: "storage error", err: errors.New("database unavailable"), wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &repoStub{err: fmt.Errorf("storage: %w", tt.err)}
			link, err := service.NewService(repo).CreateCode(t.Context(), domain.Link{OriginalURL: "https://example.com"})
			if !errors.Is(err, tt.err) {
				t.Fatalf("CreateCode() error = %v, want %v", err, tt.err)
			}
			if link != (domain.Link{}) {
				t.Fatalf("CreateCode() returned link on error: %+v", link)
			}
			if repo.calls != tt.wantCalls {
				t.Fatalf("Create() calls = %d, want %d", repo.calls, tt.wantCalls)
			}
		})
	}
}

func TestGetURLByCode(t *testing.T) {
	t.Parallel()

	link := domain.Link{ShortCode: "abcdefghij", OriginalURL: "https://example.com"}
	tests := []struct {
		name string
		link domain.Link
		err  error
	}{
		{name: "found", link: link},
		{name: "not found", err: domain.ErrNotFound},
		{name: "storage error", err: errors.New("database unavailable")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &repoStub{link: tt.link, err: tt.err}
			got, err := service.NewService(repo).GetURLByCode(t.Context(), link.ShortCode)
			if !errors.Is(err, tt.err) {
				t.Fatalf("GetURLByCode() error = %v, want %v", err, tt.err)
			}
			if got != tt.link {
				t.Fatalf("GetURLByCode() = %+v, want %+v", got, tt.link)
			}
			if repo.code != link.ShortCode {
				t.Fatalf("GetByCode() code = %q, want %q", repo.code, link.ShortCode)
			}
		})
	}
}

type repoStub struct {
	link       domain.Link
	err        error
	collisions int
	calls      int
	code       string
}

func (r *repoStub) Create(_ context.Context, link domain.Link) (domain.Link, error) {
	r.calls++
	if r.calls <= r.collisions {
		return domain.Link{}, domain.ErrCodeTaken
	}
	if r.err != nil {
		return domain.Link{}, r.err
	}
	return link, nil
}

func (r *repoStub) GetByCode(_ context.Context, code string) (domain.Link, error) {
	r.code = code
	return r.link, r.err
}

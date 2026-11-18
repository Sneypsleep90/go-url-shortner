package handler

import (
	"context"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
)

type LinkService interface {
	CreateCode(ctx context.Context, link domain.Link) (domain.Link, error)
	GetUrlByCode(ctx context.Context, code string) (domain.Link, error)
}

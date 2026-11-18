package service

import (
	"context"
	"fmt"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
	"github.com/Sneypsleep90/go-url-shortner/internal/generator"
)

type Service struct {
	repo LinkRepo
}

func NewService(repo LinkRepo) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateCode(ctx context.Context, link domain.Link) (domain.Link, error) {
	code, err := generator.Generate()
	if err != nil {
		return domain.Link{}, fmt.Errorf("generate code: %w", err)
	}
	link.ShortCode = code

	created, err := s.repo.CreateCode(ctx, link)
	if err != nil {
		return domain.Link{}, fmt.Errorf("create link: %w", err)
	}

	return created, nil
}

func (s *Service) GetUrlByCode(ctx context.Context, code string) (domain.Link, error) {
	link, err := s.repo.GetUrlByCode(ctx, code)
	if err != nil {
		return domain.Link{}, fmt.Errorf("get link by code: %w", err)
	}

	return link, nil
}

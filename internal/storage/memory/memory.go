package memory

import (
	"context"
	"sync"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
)

type Storage struct {
	mu sync.RWMutex

	byURL  map[string]domain.Link
	byCode map[string]domain.Link
}

func New() *Storage {
	return &Storage{
		byURL:  make(map[string]domain.Link),
		byCode: make(map[string]domain.Link),
	}
}

func (s *Storage) Create(ctx context.Context, link domain.Link) (domain.Link, error) {
	err := ctx.Err()
	if err != nil {
		return domain.Link{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	exist, ok := s.byURL[link.OriginalURL]

	if ok {
		return exist, nil
	}

	_, ok = s.byCode[link.ShortCode]

	if ok {
		return domain.Link{}, domain.ErrCodeTaken
	}

	s.byURL[link.OriginalURL] = link
	s.byCode[link.ShortCode] = link

	return link, nil
}

func (s *Storage) GetByCode(ctx context.Context, code string) (domain.Link, error) {
	err := ctx.Err()
	if err != nil {
		return domain.Link{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	link, ok := s.byCode[code]
	if !ok {
		return domain.Link{}, domain.ErrNotFound
	}

	return link, nil
}

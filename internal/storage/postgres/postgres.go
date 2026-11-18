package postgres

import (
	"context"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
}

func NewPgStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

func (s *Storage) CreateCode(ctx context.Context, link domain.Link) (domain.Link, error) {
	query := `
    INSERT INTO links (short_code, original_url)
    VALUES ($1, $2)
    RETURNING short_code, original_url
`
	var res domain.Link

	err := s.pool.QueryRow(
		ctx,
		query,
		link.ShortCode,
		link.OriginalUrl,
	).Scan(
		&res.ShortCode,
		&res.OriginalUrl,
	)

	if err != nil {
		return domain.Link{}, err
	}

	return res, nil
}

func (s *Storage) GetUrlByCode(ctx context.Context, code string) (domain.Link, error) {
	query := `
	SELECT short_code, original_url 
	FROM links 
	WHERE short_code = $1
`

	var link domain.Link
	err := s.pool.QueryRow(
		ctx,
		query,
		code,
	).Scan(
		&link.ShortCode,
		&link.OriginalUrl,
	)

	if err != nil {
		return domain.Link{}, err
	}

	return link, nil
}

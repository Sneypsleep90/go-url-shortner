package postgres

import (
	"context"
	"errors"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
}

func NewPgStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

func (s *Storage) Create(ctx context.Context, link domain.Link) (domain.Link, error) {
	query := `
    INSERT INTO links (short_code, original_url)
    VALUES ($1, $2)
    ON CONFLICT (original_url)
    DO UPDATE SET original_url = EXCLUDED.original_url
    RETURNING short_code, original_url
`
	var res domain.Link

	err := s.pool.QueryRow(
		ctx,
		query,
		link.ShortCode,
		link.OriginalURL,
	).Scan(
		&res.ShortCode,
		&res.OriginalURL,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.Link{}, domain.ErrCodeTaken
	}
	if err != nil {
		return domain.Link{}, err
	}

	return res, nil
}

func (s *Storage) GetByCode(ctx context.Context, code string) (domain.Link, error) {
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
		&link.OriginalURL,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Link{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Link{}, err
	}

	return link, nil
}

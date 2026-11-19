package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
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
	query, args, err := sq.Insert("links").
		Columns("short_code", "original_url").
		Values(link.ShortCode, link.OriginalURL).
		Suffix("ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url").
		Suffix("RETURNING short_code, original_url").
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return domain.Link{}, fmt.Errorf("build create link query: %w", err)
	}

	var res domain.Link
	err = s.pool.QueryRow(ctx, query, args...).Scan(
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
	query, args, err := sq.Select("short_code", "original_url").
		From("links").
		Where(sq.Eq{"short_code": code}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domain.Link{}, fmt.Errorf("build get link query: %w", err)
	}

	var link domain.Link
	err = s.pool.QueryRow(ctx, query, args...).Scan(
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

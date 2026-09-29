package repo

import (
	"context"
	"fmt"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/model"
	"github.com/jmoiron/sqlx"
)

type SearchRepo struct {
	db *sqlx.DB
}

func NewSearchRepo(db *sqlx.DB) *SearchRepo {
	return &SearchRepo{db: db}
}

func (r *SearchRepo) SearchByName(ctx context.Context, query string) ([]model.User, error) {
	var users []model.User
	err := r.db.SelectContext(ctx, &users,
		`SELECT id, name, email, created_at
		 FROM users
		 WHERE to_tsvector('english', name) @@ plainto_tsquery('english', $1)
		 ORDER BY created_at DESC`,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	return users, nil
}

func (r *SearchRepo) Feed(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	var users []model.User
	err := r.db.SelectContext(ctx, &users,
		`SELECT id, name, email, created_at
		 FROM users
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("feed query: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("feed count: %w", err)
	}

	return users, total, nil
}

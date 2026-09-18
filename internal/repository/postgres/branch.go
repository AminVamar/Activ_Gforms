package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gform/internal/domain"
)

type BranchRepo struct{ pool *pgxpool.Pool }

func NewBranchRepo(pool *pgxpool.Pool) *BranchRepo { return &BranchRepo{pool: pool} }

func (r *BranchRepo) List(ctx context.Context) ([]domain.Branch, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, code, name, created_at FROM branches ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("список филиалов: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Branch, 0, 24)
	for rows.Next() {
		var b domain.Branch
		if err := rows.Scan(&b.ID, &b.Code, &b.Name, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *BranchRepo) Create(ctx context.Context, code, name string) (*domain.Branch, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return nil, fmt.Errorf("%w: код и название филиала обязательны", domain.ErrValidation)
	}

	var b domain.Branch
	err := r.pool.QueryRow(ctx, `
		INSERT INTO branches (code, name) VALUES ($1, $2)
		ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
		RETURNING id, code, name, created_at`, code, name).
		Scan(&b.ID, &b.Code, &b.Name, &b.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("добавление филиала: %w", err)
	}
	return &b, nil
}

func (r *BranchRepo) GetByCode(ctx context.Context, code string) (*domain.Branch, error) {
	var b domain.Branch
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, name, created_at FROM branches WHERE code = $1`, strings.TrimSpace(code)).
		Scan(&b.ID, &b.Code, &b.Name, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("поиск филиала: %w", err)
	}
	return &b, nil
}

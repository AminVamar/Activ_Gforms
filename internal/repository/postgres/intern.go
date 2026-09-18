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

type InternRepo struct{ pool *pgxpool.Pool }

func NewInternRepo(pool *pgxpool.Pool) *InternRepo { return &InternRepo{pool: pool} }

const internAggregates = `
	LEFT JOIN (
		SELECT at.intern_id,
		       count(*) AS attempts_count,
		       count(*) FILTER (
		           WHERE EXISTS (SELECT 1 FROM answers an WHERE an.attempt_id = at.id AND an.score IS NULL)
		       ) AS pending_attempts
		FROM attempts at
		GROUP BY at.intern_id
	) agg ON agg.intern_id = i.id`

const internWhere = `
	WHERE ($1 = '' OR (
			i.last_name || ' ' || i.first_name || ' ' || i.middle_name ILIKE $2 ESCAPE '\'
			OR i.phone ILIKE $2 ESCAPE '\'
		))
	  AND ($3::bigint IS NULL OR i.branch_id = $3)`

func (r *InternRepo) List(ctx context.Context, f domain.InternFilter) ([]domain.InternListItem, int, error) {
	f.Normalize()
	search := strings.TrimSpace(f.Search)
	pattern := "%" + escapeLike(search) + "%"

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM interns i`+internWhere, search, pattern, f.BranchID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("подсчёт стажёров: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.phone, i.last_name, i.first_name, i.middle_name, i.branch_id,
		       COALESCE(b.code, ''), COALESCE(b.name, ''), i.created_at, i.updated_at,
		       COALESCE(agg.attempts_count, 0), COALESCE(agg.pending_attempts, 0) > 0
		FROM interns i
		LEFT JOIN branches b ON b.id = i.branch_id`+internAggregates+internWhere+`
		ORDER BY i.last_name, i.first_name, i.id
		LIMIT $4 OFFSET $5`, search, pattern, f.BranchID, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("список стажёров: %w", err)
	}
	defer rows.Close()

	out := make([]domain.InternListItem, 0, f.Limit)
	for rows.Next() {
		var it domain.InternListItem
		if err := rows.Scan(&it.ID, &it.Phone, &it.LastName, &it.FirstName, &it.MiddleName,
			&it.BranchID, &it.BranchCode, &it.BranchName, &it.CreatedAt, &it.UpdatedAt,
			&it.AttemptsCount, &it.NeedsGrading); err != nil {
			return nil, 0, err
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}

func (r *InternRepo) GetByID(ctx context.Context, id int64) (*domain.Intern, error) {
	var in domain.Intern
	err := r.pool.QueryRow(ctx, `
		SELECT i.id, i.phone, i.last_name, i.first_name, i.middle_name, i.branch_id,
		       COALESCE(b.code, ''), COALESCE(b.name, ''), i.created_at, i.updated_at
		FROM interns i
		LEFT JOIN branches b ON b.id = i.branch_id
		WHERE i.id = $1`, id).
		Scan(&in.ID, &in.Phone, &in.LastName, &in.FirstName, &in.MiddleName, &in.BranchID,
			&in.BranchCode, &in.BranchName, &in.CreatedAt, &in.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("чтение стажёра: %w", err)
	}
	return &in, nil
}

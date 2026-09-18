package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gform/internal/domain"
)

type FormRepo struct{ pool *pgxpool.Pool }

func NewFormRepo(pool *pgxpool.Pool) *FormRepo { return &FormRepo{pool: pool} }

const formColumns = `id, form_id, test_id, test_title, duration_minutes,
	form_url, respondent_url, edit_url, item_map, question_count, status, closed_at, created_at`

func (r *FormRepo) Create(ctx context.Context, f *domain.FormSession) error {
	itemMap, err := json.Marshal(f.ItemMap)
	if err != nil {
		return fmt.Errorf("сериализация item_map: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO form_sessions
			(form_id, test_id, test_title, duration_minutes, form_url, respondent_url,
			 edit_url, item_map, question_count, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'open')
		RETURNING id, status, created_at`,
		f.FormID, f.TestID, f.TestTitle, f.DurationMinutes, f.FormURL, f.RespondentURL,
		f.EditURL, itemMap, f.QuestionCount).
		Scan(&f.ID, &f.Status, &f.CreatedAt)
	if err != nil {
		return fmt.Errorf("сохранение формы: %w", err)
	}
	return nil
}

func (r *FormRepo) GetByID(ctx context.Context, id int64) (*domain.FormSession, error) {
	return r.one(ctx, `SELECT `+formColumns+` FROM form_sessions WHERE id = $1`, id)
}

func (r *FormRepo) GetByFormID(ctx context.Context, formID string) (*domain.FormSession, error) {
	return r.one(ctx, `SELECT `+formColumns+` FROM form_sessions WHERE form_id = $1`, formID)
}

func (r *FormRepo) List(ctx context.Context, status string, p domain.Pagination) ([]domain.FormSession, int, error) {
	p.Normalize()

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM form_sessions WHERE ($1 = '' OR status = $1)`, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("подсчёт форм: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+formColumns+` FROM form_sessions
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`, status, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("список форм: %w", err)
	}
	defer rows.Close()

	out := make([]domain.FormSession, 0, p.Limit)
	for rows.Next() {
		f, err := scanForm(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *f)
	}
	return out, total, rows.Err()
}

func (r *FormRepo) Close(ctx context.Context, id int64) (*domain.FormSession, error) {
	f, err := r.one(ctx, `
		UPDATE form_sessions
		SET status = 'closed',
		    closed_at = COALESCE(closed_at, now())
		WHERE id = $1
		RETURNING `+formColumns, id)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (r *FormRepo) one(ctx context.Context, query string, args ...any) (*domain.FormSession, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("чтение формы: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("чтение формы: %w", err)
		}
		return nil, domain.ErrNotFound
	}
	f, err := scanForm(rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func scanForm(rows pgx.Rows) (*domain.FormSession, error) {
	var (
		f       domain.FormSession
		itemMap []byte
	)
	err := rows.Scan(&f.ID, &f.FormID, &f.TestID, &f.TestTitle, &f.DurationMinutes,
		&f.FormURL, &f.RespondentURL, &f.EditURL, &itemMap, &f.QuestionCount,
		&f.Status, &f.ClosedAt, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("разбор формы: %w", err)
	}
	f.ItemMap = map[string]int64{}
	if len(itemMap) > 0 {
		if err := json.Unmarshal(itemMap, &f.ItemMap); err != nil {
			return nil, fmt.Errorf("разбор item_map: %w", err)
		}
	}
	return &f, nil
}

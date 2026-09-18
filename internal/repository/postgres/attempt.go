package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gform/internal/domain"
)

type AttemptRepo struct{ pool *pgxpool.Pool }

func NewAttemptRepo(pool *pgxpool.Pool) *AttemptRepo { return &AttemptRepo{pool: pool} }

func (r *AttemptRepo) SaveSubmission(ctx context.Context, in domain.SaveSubmissionInput) (*domain.SaveSubmissionResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("начало транзакции: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	internID, err := ensureIntern(ctx, tx, in)
	if err != nil {
		return nil, err
	}

	submittedAt := time.Now()
	if in.Sub.SubmittedAt != nil {
		submittedAt = *in.Sub.SubmittedAt
	}

	var attemptID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO attempts
			(response_id, form_session_id, intern_id, test_id, branch_id,
			 total_questions, auto_submitted, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (response_id) DO NOTHING
		RETURNING id`,
		in.Sub.ResponseID, in.Form.ID, internID, in.Form.TestID, in.BranchID,
		in.TotalQuestions, in.Sub.AutoSubmitted, submittedAt).Scan(&attemptID)

	if errors.Is(err, pgx.ErrNoRows) {
		return duplicateResult(ctx, tx, in.Sub.ResponseID)
	}
	if err != nil {
		return nil, fmt.Errorf("сохранение попытки: %w", err)
	}

	scored := 0
	for i, a := range in.Answers {
		var answerID int64
		err := tx.QueryRow(ctx, `
			INSERT INTO answers
				(attempt_id, question_id, question_type, item_id, answer_text, score, graded_by, graded_at, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id`,
			attemptID, a.QuestionID, a.QuestionType, a.ItemID, a.AnswerText,
			a.Score, a.GradedBy, gradedAt(a.Score), i).Scan(&answerID)
		if err != nil {
			return nil, fmt.Errorf("сохранение ответа: %w", err)
		}
		if a.Score != nil {
			scored += *a.Score
			if err := logGrade(ctx, tx, domain.GradeChange{
				AnswerID: answerID, AttemptID: attemptID,
				OldScore: nil, NewScore: a.Score, Source: domain.GradedAuto,
			}); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("фиксация попытки: %w", err)
	}

	return &domain.SaveSubmissionResult{
		AttemptID: attemptID,
		InternID:  internID,
		Scored:    scored,
		Total:     in.TotalQuestions,
	}, nil
}

func ensureIntern(ctx context.Context, tx pgx.Tx, in domain.SaveSubmissionInput) (int64, error) {
	var internID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO interns (phone, last_name, first_name, middle_name, branch_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (phone) DO UPDATE SET
			last_name   = EXCLUDED.last_name,
			first_name  = EXCLUDED.first_name,
			middle_name = EXCLUDED.middle_name,
			branch_id   = COALESCE(EXCLUDED.branch_id, interns.branch_id),
			updated_at  = now()
		RETURNING id`,
		in.Sub.Phone, in.Sub.LastName, in.Sub.FirstName, in.Sub.MiddleName, in.BranchID).Scan(&internID)
	if err != nil {
		return 0, fmt.Errorf("сохранение стажёра: %w", err)
	}
	return internID, nil
}

func duplicateResult(ctx context.Context, tx pgx.Tx, responseID string) (*domain.SaveSubmissionResult, error) {
	var res domain.SaveSubmissionResult
	err := tx.QueryRow(ctx, `
		SELECT a.id, a.intern_id, a.total_questions,
		       COALESCE((SELECT sum(an.score) FROM answers an WHERE an.attempt_id = a.id), 0)
		FROM attempts a WHERE a.response_id = $1`, responseID).
		Scan(&res.AttemptID, &res.InternID, &res.Total, &res.Scored)
	if err != nil {
		return nil, fmt.Errorf("чтение существующей попытки: %w", err)
	}
	res.Duplicate = true
	return &res, nil
}

func gradedAt(score *int) *time.Time {
	if score == nil {
		return nil
	}
	now := time.Now()
	return &now
}

func logGrade(ctx context.Context, tx pgx.Tx, c domain.GradeChange) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO grading_log (answer_id, attempt_id, old_score, new_score, source)
		VALUES ($1, $2, $3, $4, $5)`, c.AnswerID, c.AttemptID, c.OldScore, c.NewScore, c.Source)
	if err != nil {
		return fmt.Errorf("запись истории правок: %w", err)
	}
	return nil
}

const attemptSelect = `
	SELECT a.id, a.response_id, a.form_session_id, fs.form_id, a.intern_id, a.test_id,
	       fs.test_title, a.branch_id, a.total_questions, a.auto_submitted,
	       a.submitted_at, a.created_at,
	       i.phone,
	       btrim(i.last_name || ' ' || i.first_name || ' ' || i.middle_name),
	       COALESCE(b.code, ''), COALESCE(b.name, ''),
	       COALESCE(sum(an.score), 0)::int AS scored,
	       count(an.id) FILTER (WHERE an.score IS NULL)::int AS pending
	FROM attempts a
	JOIN form_sessions fs ON fs.id = a.form_session_id
	JOIN interns i ON i.id = a.intern_id
	LEFT JOIN branches b ON b.id = a.branch_id
	LEFT JOIN answers an ON an.attempt_id = a.id`

const attemptWhere = `
	WHERE ($1::bigint IS NULL OR a.intern_id = $1)
	  AND ($2::bigint IS NULL OR a.test_id = $2)
	  AND ($3::bigint IS NULL OR a.branch_id = $3)
	GROUP BY a.id, fs.form_id, fs.test_title, i.phone, i.last_name, i.first_name, i.middle_name, b.code, b.name
	HAVING $4 = ''
	    OR ($4 = 'needs_grading' AND count(an.id) FILTER (WHERE an.score IS NULL) > 0)
	    OR ($4 = 'graded'        AND count(an.id) FILTER (WHERE an.score IS NULL) = 0)`

func (r *AttemptRepo) GetByID(ctx context.Context, id int64) (*domain.Attempt, error) {
	rows, err := r.pool.Query(ctx, attemptSelect+`
		WHERE a.id = $1
		GROUP BY a.id, fs.form_id, fs.test_title, i.phone, i.last_name, i.first_name, i.middle_name, b.code, b.name`, id)
	if err != nil {
		return nil, fmt.Errorf("чтение попытки: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("чтение попытки: %w", err)
		}
		return nil, domain.ErrNotFound
	}
	return scanAttempt(rows)
}

func (r *AttemptRepo) List(ctx context.Context, f domain.AttemptFilter) ([]domain.Attempt, int, error) {
	f.Normalize()
	status := strings.TrimSpace(f.Status)

	var total int
	countQuery := `SELECT count(*) FROM (` + attemptSelect + attemptWhere + `) t`
	if err := r.pool.QueryRow(ctx, countQuery, f.InternID, f.TestID, f.BranchID, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("подсчёт попыток: %w", err)
	}

	rows, err := r.pool.Query(ctx, attemptSelect+attemptWhere+`
		ORDER BY a.submitted_at DESC, a.id DESC
		LIMIT $5 OFFSET $6`, f.InternID, f.TestID, f.BranchID, status, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("список попыток: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Attempt, 0, f.Limit)
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *a)
	}
	return out, total, rows.Err()
}

func (r *AttemptRepo) ListPending(ctx context.Context, f domain.AttemptFilter) ([]domain.Attempt, int, error) {
	f.Status = domain.StatusNeedsGrading
	return r.List(ctx, f)
}

func scanAttempt(rows pgx.Rows) (*domain.Attempt, error) {
	var a domain.Attempt
	err := rows.Scan(&a.ID, &a.ResponseID, &a.FormSessionID, &a.FormID, &a.InternID, &a.TestID,
		&a.TestTitle, &a.BranchID, &a.TotalQuestions, &a.AutoSubmitted,
		&a.SubmittedAt, &a.CreatedAt, &a.InternPhone, &a.InternName, &a.BranchCode, &a.BranchName,
		&a.Scored, &a.PendingCount)
	if err != nil {
		return nil, fmt.Errorf("разбор попытки: %w", err)
	}
	a.NeedsGrading = a.PendingCount > 0
	return &a, nil
}

func (r *AttemptRepo) Answers(ctx context.Context, attemptID int64, onlyPending bool) ([]domain.Answer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, attempt_id, question_id, question_type, item_id, answer_text,
		       score, graded_by, graded_at, position
		FROM answers
		WHERE attempt_id = $1 AND ($2 = FALSE OR score IS NULL)
		ORDER BY position, id`, attemptID, onlyPending)
	if err != nil {
		return nil, fmt.Errorf("чтение ответов: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Answer, 0, 32)
	for rows.Next() {
		var a domain.Answer
		if err := rows.Scan(&a.ID, &a.AttemptID, &a.QuestionID, &a.QuestionType, &a.ItemID,
			&a.AnswerText, &a.Score, &a.GradedBy, &a.GradedAt, &a.Position); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AttemptRepo) ApplyGrades(ctx context.Context, attemptID int64, grades []domain.GradeInput, source string) (int, error) {
	if len(grades) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("начало транзакции: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	changed := 0
	for _, g := range grades {
		var (
			oldScore *int
			owner    int64
		)
		err := tx.QueryRow(ctx,
			`SELECT attempt_id, score FROM answers WHERE id = $1 FOR UPDATE`, g.AnswerID).
			Scan(&owner, &oldScore)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("%w: ответ %d не найден", domain.ErrValidation, g.AnswerID)
		}
		if err != nil {
			return 0, fmt.Errorf("чтение ответа: %w", err)
		}
		if owner != attemptID {
			return 0, fmt.Errorf("%w: ответ %d не принадлежит попытке %d", domain.ErrValidation, g.AnswerID, attemptID)
		}
		if sameScore(oldScore, g.Score) {
			continue
		}

		var gradedBy *string
		if g.Score != nil {
			s := source
			gradedBy = &s
		}
		if _, err := tx.Exec(ctx, `
			UPDATE answers SET score = $2, graded_by = $3, graded_at = $4 WHERE id = $1`,
			g.AnswerID, g.Score, gradedBy, gradedAt(g.Score)); err != nil {
			return 0, fmt.Errorf("обновление балла: %w", err)
		}
		if err := logGrade(ctx, tx, domain.GradeChange{
			AnswerID: g.AnswerID, AttemptID: attemptID,
			OldScore: oldScore, NewScore: g.Score, Source: source,
		}); err != nil {
			return 0, err
		}
		changed++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("фиксация правок: %w", err)
	}
	return changed, nil
}

func (r *AttemptRepo) GradingLog(ctx context.Context, attemptID int64) ([]domain.GradingLogEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, answer_id, attempt_id, old_score, new_score, source, created_at
		FROM grading_log WHERE attempt_id = $1 ORDER BY created_at DESC, id DESC`, attemptID)
	if err != nil {
		return nil, fmt.Errorf("чтение истории правок: %w", err)
	}
	defer rows.Close()

	out := make([]domain.GradingLogEntry, 0, 16)
	for rows.Next() {
		var e domain.GradingLogEntry
		if err := rows.Scan(&e.ID, &e.AnswerID, &e.AttemptID, &e.OldScore, &e.NewScore,
			&e.Source, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func sameScore(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

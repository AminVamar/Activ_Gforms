package usecase

import (
	"context"
	"errors"
	"fmt"

	"gform/internal/domain"
)

type Grading struct {
	attempts domain.AttemptRepository
	source   domain.TestSource
}

func NewGrading(attempts domain.AttemptRepository, source domain.TestSource) *Grading {
	return &Grading{attempts: attempts, source: source}
}

type AttemptView struct {
	Attempt *domain.Attempt `json:"attempt"`
	Answers []domain.Answer `json:"answers"`
	Status  string          `json:"status"`
	Score   int             `json:"score"`
	Total   int             `json:"total"`

	SourceAvailable bool `json:"source_available"`
}

func (g *Grading) Pending(ctx context.Context, f domain.AttemptFilter) ([]domain.Attempt, int, error) {
	return g.attempts.ListPending(ctx, f)
}

func (g *Grading) Attempt(ctx context.Context, id int64, onlyPending bool) (*AttemptView, error) {
	attempt, err := g.attempts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	answers, err := g.attempts.Answers(ctx, id, onlyPending)
	if err != nil {
		return nil, err
	}

	view := &AttemptView{
		Attempt: attempt,
		Answers: answers,
		Status:  attempt.Status(),
		Score:   attempt.Scored,
		Total:   attempt.TotalQuestions,
	}

	test, err := g.source.Get(ctx, attempt.TestID)
	if err != nil {
		return view, nil
	}
	view.SourceAvailable = true
	for i := range view.Answers {
		q, ok := test.QuestionByID(view.Answers[i].QuestionID)
		if !ok {
			continue
		}
		view.Answers[i].QuestionText = q.Content
		if domain.IsAutoGraded(q.Type) {
			view.Answers[i].CorrectAnswers = q.CorrectOptions()
		}
		if view.Answers[i].QuestionType == 0 {
			view.Answers[i].QuestionType = q.Type
		}
	}
	return view, nil
}

func (g *Grading) Grade(ctx context.Context, attemptID int64, grades []domain.GradeInput) (*AttemptView, int, error) {
	if len(grades) == 0 {
		return nil, 0, fmt.Errorf("%w: список grades пуст", domain.ErrValidation)
	}
	for _, gr := range grades {
		if !validScore(gr.Score) {
			return nil, 0, fmt.Errorf("%w: балл должен быть 0, 1 или null", domain.ErrValidation)
		}
		if gr.AnswerID <= 0 {
			return nil, 0, fmt.Errorf("%w: некорректный answer_id", domain.ErrValidation)
		}
	}

	if _, err := g.attempts.GetByID(ctx, attemptID); err != nil {
		return nil, 0, err
	}

	changed, err := g.attempts.ApplyGrades(ctx, attemptID, grades, domain.GradedManual)
	if err != nil {
		return nil, 0, err
	}

	view, err := g.Attempt(ctx, attemptID, false)
	if err != nil {
		return nil, 0, err
	}
	return view, changed, nil
}

type RescoreResult struct {
	AttemptID int64  `json:"attempt_id"`
	Changed   int    `json:"changed"`
	Score     int    `json:"score"`
	Total     int    `json:"total"`
	Status    string `json:"status"`
}

func (g *Grading) Rescore(ctx context.Context, attemptID int64) (*RescoreResult, error) {
	attempt, err := g.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	test, err := g.source.Get(ctx, attempt.TestID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: тест %d исчез из источника", domain.ErrSourceUnavailable, attempt.TestID)
		}
		return nil, err
	}

	answers, err := g.attempts.Answers(ctx, attemptID, false)
	if err != nil {
		return nil, err
	}

	updates := make([]domain.GradeInput, 0, len(answers))
	for _, a := range answers {
		if a.GradedBy != nil && *a.GradedBy == domain.GradedManual {
			continue
		}
		q, ok := test.QuestionByID(a.QuestionID)
		if !ok || !domain.IsAutoGraded(q.Type) {
			continue
		}
		newScore := ScoreAnswer(q, a.AnswerText)
		if sameScore(a.Score, newScore) {
			continue
		}
		updates = append(updates, domain.GradeInput{AnswerID: a.ID, Score: newScore})
	}

	changed, err := g.attempts.ApplyGrades(ctx, attemptID, updates, domain.GradedAuto)
	if err != nil {
		return nil, err
	}

	updated, err := g.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	return &RescoreResult{
		AttemptID: attemptID,
		Changed:   changed,
		Score:     updated.Scored,
		Total:     updated.TotalQuestions,
		Status:    updated.Status(),
	}, nil
}

func sameScore(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gform/internal/domain"
)

func gradedFixture(t *testing.T) (*Grading, *fakeAttempts, *fakeSource, int64) {
	t.Helper()

	source := &fakeSource{tests: map[int64]domain.Test{35: sampleTest()}}
	forms := newFakeForms()
	_ = forms.Create(context.Background(), &domain.FormSession{
		FormID: "form-35", TestID: 35, QuestionCount: 3,
		ItemMap: map[string]int64{"i10": 10, "i11": 11, "i12": 12},
	})
	attempts := newFakeAttempts()

	intake := NewSubmissions(source, forms, newFakeBranches("5100"), attempts)
	res, err := intake.Accept(context.Background(), baseSubmission())
	if err != nil {
		t.Fatalf("подготовка попытки: %v", err)
	}
	return NewGrading(attempts, source), attempts, source, res.AttemptID
}

func TestAttemptViewEnrichesFromSource(t *testing.T) {
	grading, _, _, attemptID := gradedFixture(t)

	view, err := grading.Attempt(context.Background(), attemptID, false)
	if err != nil {
		t.Fatalf("Attempt: %v", err)
	}
	if !view.SourceAvailable {
		t.Fatal("при доступном источнике source_available должен быть true")
	}
	if view.Answers[0].QuestionText != "Закрытый вопрос" {
		t.Fatalf("текст вопроса не подтянулся: %q", view.Answers[0].QuestionText)
	}
	if len(view.Answers[0].CorrectAnswers) != 1 || view.Answers[0].CorrectAnswers[0] != "Да" {
		t.Fatalf("правильный ответ не подтянулся: %v", view.Answers[0].CorrectAnswers)
	}
	if len(view.Answers[1].CorrectAnswers) != 0 {
		t.Fatal("у открытого вопроса правильного ответа быть не может")
	}
	if view.Status != domain.StatusNeedsGrading {
		t.Fatalf("статус = %s, ожидался needs_grading", view.Status)
	}
}

func TestAttemptViewDegradesWithoutSource(t *testing.T) {
	grading, _, source, attemptID := gradedFixture(t)
	source.err = fmt.Errorf("%w: сеть", domain.ErrSourceUnavailable)

	view, err := grading.Attempt(context.Background(), attemptID, false)
	if err != nil {
		t.Fatalf("попытка должна открываться и без источника: %v", err)
	}
	if view.SourceAvailable {
		t.Fatal("source_available должен быть false")
	}
	if len(view.Answers) != 3 {
		t.Fatalf("ответы должны отдаваться целиком, получено %d", len(view.Answers))
	}
}

func TestGradeSetsScoresAndClosesQueue(t *testing.T) {
	grading, attempts, _, attemptID := gradedFixture(t)
	answers, _ := attempts.Answers(context.Background(), attemptID, true)

	grades := []domain.GradeInput{
		{AnswerID: answers[0].ID, Score: score(1)},
		{AnswerID: answers[1].ID, Score: score(0)},
	}
	view, changed, err := grading.Grade(context.Background(), attemptID, grades)
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if changed != 2 {
		t.Fatalf("изменено %d ответов, ожидалось 2", changed)
	}
	if view.Status != domain.StatusGraded {
		t.Fatalf("после проверки всех вопросов статус = %s, ожидался graded", view.Status)
	}
	if view.Score != 2 || view.Total != 3 {
		t.Fatalf("итог = %d из %d, ожидалось 2 из 3", view.Score, view.Total)
	}
}

func TestGradeNullReturnsQuestionToQueue(t *testing.T) {
	grading, attempts, _, attemptID := gradedFixture(t)
	all, _ := attempts.Answers(context.Background(), attemptID, false)

	view, changed, err := grading.Grade(context.Background(), attemptID,
		[]domain.GradeInput{{AnswerID: all[0].ID, Score: nil}})
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if changed != 1 {
		t.Fatalf("изменено %d, ожидался 1", changed)
	}
	if view.Status != domain.StatusNeedsGrading {
		t.Fatal("null должен возвращать вопрос в очередь на проверку")
	}
}

func TestGradeRejectsInvalidScore(t *testing.T) {
	grading, attempts, _, attemptID := gradedFixture(t)
	all, _ := attempts.Answers(context.Background(), attemptID, false)

	two := 2
	_, _, err := grading.Grade(context.Background(), attemptID,
		[]domain.GradeInput{{AnswerID: all[0].ID, Score: &two}})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("балл 2 должен отвергаться, получено %v", err)
	}
}

func TestRescoreKeepsManualScores(t *testing.T) {
	grading, attempts, source, attemptID := gradedFixture(t)
	ctx := context.Background()

	all, _ := attempts.Answers(ctx, attemptID, false)
	openAnswer := all[1]
	if _, _, err := grading.Grade(ctx, attemptID,
		[]domain.GradeInput{{AnswerID: openAnswer.ID, Score: score(1)}}); err != nil {
		t.Fatalf("ручная оценка: %v", err)
	}

	fixed := sampleTest()
	fixed.Questions[0].Options[0].IsCorrect = false
	fixed.Questions[0].Options[1].IsCorrect = true
	source.tests[35] = fixed

	res, err := grading.Rescore(ctx, attemptID)
	if err != nil {
		t.Fatalf("Rescore: %v", err)
	}
	if res.Changed != 1 {
		t.Fatalf("пересчёт изменил %d ответов, ожидался 1 (только закрытый вопрос)", res.Changed)
	}

	after, _ := attempts.Answers(ctx, attemptID, false)
	if after[0].Score == nil || *after[0].Score != 0 {
		t.Fatalf("закрытый вопрос должен стать 0, получено %v", after[0].Score)
	}
	if after[0].GradedBy == nil || *after[0].GradedBy != domain.GradedAuto {
		t.Fatal("источник правки должен быть auto")
	}
	if after[1].Score == nil || *after[1].Score != 1 {
		t.Fatal("ручной балл не должен затираться пересчётом")
	}
	if after[2].Score != nil {
		t.Fatal("шкала должна остаться без балла")
	}
}

func TestRescoreFailsWithoutSource(t *testing.T) {
	grading, _, source, attemptID := gradedFixture(t)
	source.err = fmt.Errorf("%w: сеть", domain.ErrSourceUnavailable)

	if _, err := grading.Rescore(context.Background(), attemptID); !errors.Is(err, domain.ErrSourceUnavailable) {
		t.Fatalf("ожидалась недоступность источника, получено %v", err)
	}
}

func TestPendingListsOnlyUngraded(t *testing.T) {
	grading, attempts, _, attemptID := gradedFixture(t)
	ctx := context.Background()

	items, _, err := grading.Pending(ctx, domain.AttemptFilter{})
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("в очереди %d попыток, ожидалась одна", len(items))
	}

	all, _ := attempts.Answers(ctx, attemptID, true)
	grades := make([]domain.GradeInput, 0, len(all))
	for _, a := range all {
		grades = append(grades, domain.GradeInput{AnswerID: a.ID, Score: score(1)})
	}
	if _, _, err := grading.Grade(ctx, attemptID, grades); err != nil {
		t.Fatalf("Grade: %v", err)
	}

	items, _, _ = grading.Pending(ctx, domain.AttemptFilter{})
	if len(items) != 0 {
		t.Fatalf("после проверки очередь должна опустеть, осталось %d", len(items))
	}
}

package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gform/internal/domain"
)

func sampleTest() domain.Test {
	return domain.Test{
		ID:    35,
		Title: "Проверочный тест",
		Questions: []domain.Question{
			{ID: 10, Content: "Закрытый вопрос", Type: domain.QuestionChoice, Options: []domain.Option{
				{Text: "Да", IsCorrect: true},
				{Text: "Нет"},
			}},
			{ID: 11, Content: "Открытый вопрос", Type: domain.QuestionOpen},
			{ID: 12, Content: "Шкала", Type: domain.QuestionScale},
		},
	}
}

func newSubmissionsFixture(sourceErr error) (*Submissions, *fakeAttempts, *fakeBranches, *fakeForms) {
	source := &fakeSource{tests: map[int64]domain.Test{35: sampleTest()}, err: sourceErr}
	forms := newFakeForms()
	_ = forms.Create(context.Background(), &domain.FormSession{
		FormID: "form-35", TestID: 35, TestTitle: "Проверочный тест", DurationMinutes: 30,
		ItemMap:       map[string]int64{"i10": 10, "i11": 11, "i12": 12},
		QuestionCount: 3,
	})
	branches := newFakeBranches("5100")
	attempts := newFakeAttempts()
	return NewSubmissions(source, forms, branches, attempts), attempts, branches, forms
}

func baseSubmission() *domain.Submission {
	return &domain.Submission{
		FormID: "form-35", ResponseID: "resp-1",
		LastName: "Каримов", FirstName: "Далер", Phone: "+992900000000",
		BranchCode: "5100", BranchName: "Филиал 5100",
		Answers: []domain.SubmittedAnswer{
			{ItemID: "i10", Answer: "Да"},
			{ItemID: "i11", Answer: "мой развёрнутый ответ"},
		},
	}
}

func TestAcceptScoresClosedQuestionsAndQueuesTheRest(t *testing.T) {
	uc, attempts, _, _ := newSubmissionsFixture(nil)

	res, err := uc.Accept(context.Background(), baseSubmission())
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if !res.Scored {
		t.Fatal("при доступном источнике scored должен быть true")
	}
	if res.Score != 1 || res.Total != 3 {
		t.Fatalf("итог = %d из %d, ожидалось 1 из 3", res.Score, res.Total)
	}
	if !res.NeedsGrading {
		t.Fatal("открытый вопрос и шкала должны отправить попытку на ручную проверку")
	}

	saved := attempts.saved[0].Answers
	if len(saved) != 3 {
		t.Fatalf("сохранено %d ответов, ожидалось 3: пропущенный вопрос тоже сохраняется", len(saved))
	}
	if saved[0].Score == nil || *saved[0].Score != 1 {
		t.Fatalf("закрытый вопрос должен получить 1, получено %v", saved[0].Score)
	}
	if saved[1].Score != nil || saved[2].Score != nil {
		t.Fatal("открытый вопрос и шкала должны сохраняться с null")
	}
	if saved[2].AnswerText != "" {
		t.Fatalf("пропущенный вопрос должен сохраняться с пустым ответом, получено %q", saved[2].AnswerText)
	}
}

func TestAcceptSurvivesUnavailableSource(t *testing.T) {
	uc, attempts, _, _ := newSubmissionsFixture(fmt.Errorf("%w: сеть", domain.ErrSourceUnavailable))

	res, err := uc.Accept(context.Background(), baseSubmission())
	if err != nil {
		t.Fatalf("приём ответов не должен срываться из-за источника: %v", err)
	}
	if res.Scored {
		t.Fatal("при недоступном источнике scored должен быть false")
	}

	saved := attempts.saved[0].Answers
	if len(saved) != 3 {
		t.Fatalf("состав вопросов должен браться из item_map: сохранено %d, ожидалось 3", len(saved))
	}
	for _, a := range saved {
		if a.Score != nil {
			t.Fatalf("все баллы должны быть null, у вопроса %d = %v", a.QuestionID, *a.Score)
		}
	}
}

func TestAcceptIsIdempotentByResponseID(t *testing.T) {
	uc, attempts, _, _ := newSubmissionsFixture(nil)
	ctx := context.Background()

	first, err := uc.Accept(ctx, baseSubmission())
	if err != nil {
		t.Fatalf("первый приём: %v", err)
	}
	second, err := uc.Accept(ctx, baseSubmission())
	if err != nil {
		t.Fatalf("повторный приём: %v", err)
	}

	if !second.Duplicate {
		t.Fatal("повтор по response_id должен помечаться как duplicate")
	}
	if second.AttemptID != first.AttemptID {
		t.Fatalf("повтор создал новую попытку: %d != %d", second.AttemptID, first.AttemptID)
	}
	if len(attempts.attempts) != 1 {
		t.Fatalf("в базе %d попыток, ожидалась одна", len(attempts.attempts))
	}
}

func TestAcceptAddsUnknownBranch(t *testing.T) {
	uc, _, branches, _ := newSubmissionsFixture(nil)

	sub := baseSubmission()
	sub.BranchCode = "9999"
	sub.BranchName = "ЦБО Новый"

	if _, err := uc.Accept(context.Background(), sub); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if _, err := branches.GetByCode(context.Background(), "9999"); err != nil {
		t.Fatal("неизвестный филиал должен добавляться в справочник")
	}
}

func TestAcceptRejectsSubmissionWithoutPhone(t *testing.T) {
	uc, _, _, _ := newSubmissionsFixture(nil)

	sub := baseSubmission()
	sub.Phone = "   "

	_, err := uc.Accept(context.Background(), sub)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ошибка валидации, получено %v", err)
	}
}

func TestAcceptUnknownFormIsNotFound(t *testing.T) {
	uc, _, _, _ := newSubmissionsFixture(nil)

	sub := baseSubmission()
	sub.FormID = "form-неизвестная"

	_, err := uc.Accept(context.Background(), sub)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидалось «не найдено», получено %v", err)
	}
}

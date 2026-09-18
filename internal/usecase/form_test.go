package usecase

import (
	"context"
	"errors"
	"testing"

	"gform/internal/domain"
)

func newFormsFixture() (*Forms, *fakeForms, *fakeAppScript) {
	source := &fakeSource{tests: map[int64]domain.Test{
		35: sampleTest(),
		99: {ID: 99, Title: "Пустой тест"},
	}}
	forms := newFakeForms()
	script := &fakeAppScript{}
	return NewForms(source, script, forms, newFakeBranches("5100", "5200"),
		"https://example.test/api/v1/webhook/submission"), forms, script
}

func TestCreateFormStoresItemMapAndUrls(t *testing.T) {
	uc, forms, script := newFormsFixture()

	session, err := uc.Create(context.Background(), 35, 30)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if session.QuestionCount != 3 {
		t.Fatalf("question_count = %d, ожидалось 3", session.QuestionCount)
	}
	if len(session.ItemMap) != 3 {
		t.Fatalf("item_map из %d записей, ожидалось 3", len(session.ItemMap))
	}
	if session.RespondentURL == "" || session.FormURL == "" {
		t.Fatal("должны вернуться и ссылка для стажёров, и ссылка на саму форму")
	}
	if len(forms.created) != 1 {
		t.Fatal("форма должна сохраниться в базе")
	}

	req := script.created[0]
	if len(req.Branches) != 2 {
		t.Fatalf("в форму передано %d филиалов, ожидалось 2", len(req.Branches))
	}
	for _, q := range req.Questions {
		for _, o := range q.Options {
			if o == "" {
				t.Fatal("вариант ответа не должен быть пустым")
			}
		}
	}
}

func TestCreateFormValidatesInput(t *testing.T) {
	uc, _, _ := newFormsFixture()
	ctx := context.Background()

	if _, err := uc.Create(ctx, 35, 0); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("duration_minutes = 0 должен давать 400, получено %v", err)
	}
	if _, err := uc.Create(ctx, 99, 30); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("тест без вопросов должен давать 400, получено %v", err)
	}
	if _, err := uc.Create(ctx, 12345, 30); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("несуществующий тест должен давать 404, получено %v", err)
	}
}

func TestCreateFormFailsWhenAppScriptDown(t *testing.T) {
	uc, forms, script := newFormsFixture()
	script.err = errors.New("apps script недоступен")

	if _, err := uc.Create(context.Background(), 35, 30); err == nil {
		t.Fatal("при недоступном Apps Script форма создаваться не должна")
	}
	if len(forms.created) != 0 {
		t.Fatal("в базе не должно остаться формы, которую не создали в Google")
	}
}

func TestCloseFormIsIdempotent(t *testing.T) {
	uc, _, script := newFormsFixture()
	ctx := context.Background()

	session, err := uc.Create(ctx, 35, 30)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := uc.Close(ctx, session.ID); err != nil {
		t.Fatalf("первое закрытие: %v", err)
	}
	closed, err := uc.Close(ctx, session.ID)
	if err != nil {
		t.Fatalf("повторное закрытие не должно быть ошибкой: %v", err)
	}
	if closed.Status != domain.FormClosed {
		t.Fatalf("статус = %s, ожидался closed", closed.Status)
	}
	if len(script.closed) != 1 {
		t.Fatalf("Apps Script вызван %d раз, ожидался один", len(script.closed))
	}
}

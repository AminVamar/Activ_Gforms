package usecase

import (
	"context"
	"fmt"

	"gform/internal/domain"
)

type Forms struct {
	source     domain.TestSource
	appscript  domain.AppScriptClient
	forms      domain.FormRepository
	branches   domain.BranchRepository
	webhookURL string
}

func NewForms(
	source domain.TestSource,
	appscript domain.AppScriptClient,
	forms domain.FormRepository,
	branches domain.BranchRepository,
	webhookURL string,
) *Forms {
	return &Forms{source: source, appscript: appscript, forms: forms, branches: branches, webhookURL: webhookURL}
}

func (f *Forms) Create(ctx context.Context, testID int64, durationMinutes int) (*domain.FormSession, error) {
	if durationMinutes <= 0 {
		return nil, fmt.Errorf("%w: duration_minutes должен быть больше нуля", domain.ErrValidation)
	}

	test, err := f.source.Get(ctx, testID)
	if err != nil {
		return nil, err
	}
	if len(test.Questions) == 0 {
		return nil, fmt.Errorf("%w: в тесте нет вопросов", domain.ErrValidation)
	}

	branches, err := f.branches.List(ctx)
	if err != nil {
		return nil, err
	}

	req := domain.CreateFormRequest{
		TestID:          test.ID,
		Title:           test.Title,
		Description:     test.Description,
		DurationMinutes: durationMinutes,
		Branches:        make([]domain.CreateFormBranch, 0, len(branches)),
		Questions:       make([]domain.CreateFormQuestion, 0, len(test.Questions)),
		WebhookURL:      f.webhookURL,
	}
	for _, b := range branches {
		req.Branches = append(req.Branches, domain.CreateFormBranch{Code: b.Code, Name: b.Name})
	}
	for _, q := range test.Questions {
		options := make([]string, 0, len(q.Options))
		for _, o := range q.Options {
			options = append(options, o.Text)
		}
		req.Questions = append(req.Questions, domain.CreateFormQuestion{
			QuestionID: q.ID,
			Content:    q.Content,
			Type:       q.Type,
			Options:    options,
		})
	}

	created, err := f.appscript.CreateForm(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("создание Google-формы: %w", err)
	}

	session := &domain.FormSession{
		FormID:          created.FormID,
		TestID:          test.ID,
		TestTitle:       test.Title,
		DurationMinutes: durationMinutes,
		FormURL:         created.FormURL,
		RespondentURL:   created.RespondentURL,
		EditURL:         created.EditURL,
		ItemMap:         created.ItemMap,
		QuestionCount:   len(test.Questions),
	}
	if err := f.forms.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (f *Forms) List(ctx context.Context, status string, p domain.Pagination) ([]domain.FormSession, int, error) {
	if status != "" && status != domain.FormOpen && status != domain.FormClosed {
		return nil, 0, fmt.Errorf("%w: status должен быть open или closed", domain.ErrValidation)
	}
	return f.forms.List(ctx, status, p)
}

func (f *Forms) Get(ctx context.Context, id int64) (*domain.FormSession, error) {
	return f.forms.GetByID(ctx, id)
}

func (f *Forms) Close(ctx context.Context, id int64) (*domain.FormSession, error) {
	session, err := f.forms.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !session.IsOpen() {
		return session, nil
	}

	if err := f.appscript.CloseForm(ctx, session.FormID); err != nil {
		return nil, fmt.Errorf("закрытие Google-формы: %w", err)
	}
	return f.forms.Close(ctx, id)
}

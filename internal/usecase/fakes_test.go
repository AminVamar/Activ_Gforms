package usecase

import (
	"context"
	"strconv"

	"gform/internal/domain"
)

type fakeSource struct {
	tests map[int64]domain.Test
	err   error
	calls int
}

func (f *fakeSource) List(context.Context) ([]domain.Test, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]domain.Test, 0, len(f.tests))
	for _, t := range f.tests {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeSource) Get(_ context.Context, id int64) (*domain.Test, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	t, ok := f.tests[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &t, nil
}

type fakeForms struct {
	byFormID map[string]*domain.FormSession
	created  []*domain.FormSession
	nextID   int64
}

func newFakeForms() *fakeForms {
	return &fakeForms{byFormID: map[string]*domain.FormSession{}}
}

func (f *fakeForms) Create(_ context.Context, s *domain.FormSession) error {
	f.nextID++
	s.ID = f.nextID
	s.Status = domain.FormOpen
	f.byFormID[s.FormID] = s
	f.created = append(f.created, s)
	return nil
}

func (f *fakeForms) GetByID(_ context.Context, id int64) (*domain.FormSession, error) {
	for _, s := range f.byFormID {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeForms) GetByFormID(_ context.Context, formID string) (*domain.FormSession, error) {
	s, ok := f.byFormID[formID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s, nil
}

func (f *fakeForms) List(context.Context, string, domain.Pagination) ([]domain.FormSession, int, error) {
	out := make([]domain.FormSession, 0, len(f.byFormID))
	for _, s := range f.byFormID {
		out = append(out, *s)
	}
	return out, len(out), nil
}

func (f *fakeForms) Close(_ context.Context, id int64) (*domain.FormSession, error) {
	s, err := f.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	s.Status = domain.FormClosed
	return s, nil
}

type fakeBranches struct {
	byCode map[string]*domain.Branch
	nextID int64
}

func newFakeBranches(codes ...string) *fakeBranches {
	b := &fakeBranches{byCode: map[string]*domain.Branch{}}
	for _, c := range codes {
		_, _ = b.Create(context.Background(), c, "Филиал "+c)
	}
	return b
}

func (f *fakeBranches) List(context.Context) ([]domain.Branch, error) {
	out := make([]domain.Branch, 0, len(f.byCode))
	for _, b := range f.byCode {
		out = append(out, *b)
	}
	return out, nil
}

func (f *fakeBranches) Create(_ context.Context, code, name string) (*domain.Branch, error) {
	if b, ok := f.byCode[code]; ok {
		return b, nil
	}
	f.nextID++
	b := &domain.Branch{ID: f.nextID, Code: code, Name: name}
	f.byCode[code] = b
	return b, nil
}

func (f *fakeBranches) GetByCode(_ context.Context, code string) (*domain.Branch, error) {
	b, ok := f.byCode[code]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return b, nil
}

type fakeAttempts struct {
	saved     []domain.SaveSubmissionInput
	answers   map[int64][]domain.Answer
	attempts  map[int64]*domain.Attempt
	log       []domain.GradeChange
	nextID    int64
	responses map[string]int64
}

func newFakeAttempts() *fakeAttempts {
	return &fakeAttempts{
		answers:   map[int64][]domain.Answer{},
		attempts:  map[int64]*domain.Attempt{},
		responses: map[string]int64{},
	}
}

func (f *fakeAttempts) SaveSubmission(_ context.Context, in domain.SaveSubmissionInput) (*domain.SaveSubmissionResult, error) {
	f.saved = append(f.saved, in)

	if id, ok := f.responses[in.Sub.ResponseID]; ok {
		a := f.attempts[id]
		return &domain.SaveSubmissionResult{
			AttemptID: id, InternID: a.InternID, Duplicate: true,
			Scored: a.Scored, Total: a.TotalQuestions,
		}, nil
	}

	f.nextID++
	id := f.nextID
	f.responses[in.Sub.ResponseID] = id

	scored, pending := 0, 0
	stored := make([]domain.Answer, 0, len(in.Answers))
	for i, a := range in.Answers {
		a.ID = id*1000 + int64(i)
		a.AttemptID = id
		a.Position = i
		if a.Score == nil {
			pending++
		} else {
			scored += *a.Score
		}
		stored = append(stored, a)
	}
	f.answers[id] = stored
	f.attempts[id] = &domain.Attempt{
		ID: id, ResponseID: in.Sub.ResponseID, InternID: 1, TestID: in.Form.TestID,
		TotalQuestions: in.TotalQuestions, Scored: scored, PendingCount: pending,
		NeedsGrading: pending > 0,
	}
	return &domain.SaveSubmissionResult{AttemptID: id, InternID: 1, Scored: scored, Total: in.TotalQuestions}, nil
}

func (f *fakeAttempts) GetByID(_ context.Context, id int64) (*domain.Attempt, error) {
	a, ok := f.attempts[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return a, nil
}

func (f *fakeAttempts) List(context.Context, domain.AttemptFilter) ([]domain.Attempt, int, error) {
	out := make([]domain.Attempt, 0, len(f.attempts))
	for _, a := range f.attempts {
		out = append(out, *a)
	}
	return out, len(out), nil
}

func (f *fakeAttempts) ListPending(ctx context.Context, filter domain.AttemptFilter) ([]domain.Attempt, int, error) {
	all, _, _ := f.List(ctx, filter)
	out := make([]domain.Attempt, 0, len(all))
	for _, a := range all {
		if a.PendingCount > 0 {
			out = append(out, a)
		}
	}
	return out, len(out), nil
}

func (f *fakeAttempts) Answers(_ context.Context, attemptID int64, onlyPending bool) ([]domain.Answer, error) {
	all := f.answers[attemptID]
	if !onlyPending {
		return all, nil
	}
	out := make([]domain.Answer, 0, len(all))
	for _, a := range all {
		if a.Score == nil {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeAttempts) ApplyGrades(_ context.Context, attemptID int64, grades []domain.GradeInput, source string) (int, error) {
	answers := f.answers[attemptID]
	changed := 0
	for _, g := range grades {
		for i := range answers {
			if answers[i].ID != g.AnswerID {
				continue
			}
			if sameScore(answers[i].Score, g.Score) {
				break
			}
			f.log = append(f.log, domain.GradeChange{
				AnswerID: g.AnswerID, AttemptID: attemptID,
				OldScore: answers[i].Score, NewScore: g.Score, Source: source,
			})
			answers[i].Score = g.Score
			if g.Score != nil {
				s := source
				answers[i].GradedBy = &s
			} else {
				answers[i].GradedBy = nil
			}
			changed++
			break
		}
	}
	f.answers[attemptID] = answers
	f.recount(attemptID)
	return changed, nil
}

func (f *fakeAttempts) recount(attemptID int64) {
	a, ok := f.attempts[attemptID]
	if !ok {
		return
	}
	scored, pending := 0, 0
	for _, an := range f.answers[attemptID] {
		if an.Score == nil {
			pending++
		} else {
			scored += *an.Score
		}
	}
	a.Scored, a.PendingCount, a.NeedsGrading = scored, pending, pending > 0
}

func (f *fakeAttempts) GradingLog(_ context.Context, attemptID int64) ([]domain.GradingLogEntry, error) {
	out := make([]domain.GradingLogEntry, 0, len(f.log))
	for i, c := range f.log {
		if c.AttemptID != attemptID {
			continue
		}
		out = append(out, domain.GradingLogEntry{
			ID: int64(i + 1), AnswerID: c.AnswerID, AttemptID: c.AttemptID,
			OldScore: c.OldScore, NewScore: c.NewScore, Source: c.Source,
		})
	}
	return out, nil
}

type fakeAppScript struct {
	err     error
	created []domain.CreateFormRequest
	closed  []string
}

func (f *fakeAppScript) CreateForm(_ context.Context, req domain.CreateFormRequest) (*domain.CreateFormResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.created = append(f.created, req)
	itemMap := map[string]int64{}
	for i, q := range req.Questions {
		itemMap["item"+strconv.Itoa(i)] = q.QuestionID
	}
	return &domain.CreateFormResponse{
		FormID:        "form-" + strconv.FormatInt(req.TestID, 10),
		FormURL:       "https://docs.google.com/forms/d/form/viewform",
		RespondentURL: "https://script.google.com/macros/s/dev/exec?form=form",
		EditURL:       "https://docs.google.com/forms/d/form/edit",
		ItemMap:       itemMap,
	}, nil
}

func (f *fakeAppScript) CloseForm(_ context.Context, formID string) error {
	if f.err != nil {
		return f.err
	}
	f.closed = append(f.closed, formID)
	return nil
}

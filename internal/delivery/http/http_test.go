package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gform/internal/domain"
	"gform/internal/usecase"
)

const testSecret = "секрет-для-тестов"

type stubSource struct{ err error }

func (s stubSource) List(context.Context) ([]domain.Test, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []domain.Test{{ID: 35, Title: "Тест"}}, nil
}

func (s stubSource) Get(_ context.Context, id int64) (*domain.Test, error) {
	if s.err != nil {
		return nil, s.err
	}
	if id != 35 {
		return nil, domain.ErrNotFound
	}
	return &domain.Test{ID: 35, Title: "Тест"}, nil
}

type stubForms struct{}

func (stubForms) Create(context.Context, *domain.FormSession) error { return nil }
func (stubForms) GetByID(context.Context, int64) (*domain.FormSession, error) {
	return nil, domain.ErrNotFound
}
func (stubForms) GetByFormID(context.Context, string) (*domain.FormSession, error) {
	return nil, domain.ErrNotFound
}
func (stubForms) List(context.Context, string, domain.Pagination) ([]domain.FormSession, int, error) {
	return nil, 0, nil
}
func (stubForms) Close(context.Context, int64) (*domain.FormSession, error) {
	return nil, domain.ErrNotFound
}

type stubBranches struct{}

func (stubBranches) List(context.Context) ([]domain.Branch, error) {
	return []domain.Branch{{ID: 1, Code: "5100", Name: "Филиал Садбарг"}}, nil
}
func (stubBranches) Create(_ context.Context, code, name string) (*domain.Branch, error) {
	return &domain.Branch{ID: 2, Code: code, Name: name}, nil
}
func (stubBranches) GetByCode(context.Context, string) (*domain.Branch, error) {
	return nil, domain.ErrNotFound
}

type stubAttempts struct{ lastFilter domain.AttemptFilter }

func (s *stubAttempts) SaveSubmission(context.Context, domain.SaveSubmissionInput) (*domain.SaveSubmissionResult, error) {
	return &domain.SaveSubmissionResult{AttemptID: 1, InternID: 1}, nil
}
func (s *stubAttempts) GetByID(context.Context, int64) (*domain.Attempt, error) {
	return nil, domain.ErrNotFound
}
func (s *stubAttempts) List(_ context.Context, f domain.AttemptFilter) ([]domain.Attempt, int, error) {
	s.lastFilter = f
	return nil, 0, nil
}
func (s *stubAttempts) ListPending(ctx context.Context, f domain.AttemptFilter) ([]domain.Attempt, int, error) {
	return s.List(ctx, f)
}
func (s *stubAttempts) Answers(context.Context, int64, bool) ([]domain.Answer, error) {
	return nil, nil
}
func (s *stubAttempts) ApplyGrades(context.Context, int64, []domain.GradeInput, string) (int, error) {
	return 0, nil
}
func (s *stubAttempts) GradingLog(context.Context, int64) ([]domain.GradingLogEntry, error) {
	return nil, nil
}

type stubInterns struct{ lastFilter domain.InternFilter }

func (s *stubInterns) List(_ context.Context, f domain.InternFilter) ([]domain.InternListItem, int, error) {
	s.lastFilter = f
	return []domain.InternListItem{}, 0, nil
}
func (s *stubInterns) GetByID(context.Context, int64) (*domain.Intern, error) {
	return nil, domain.ErrNotFound
}

type stubScript struct{}

func (stubScript) CreateForm(context.Context, domain.CreateFormRequest) (*domain.CreateFormResponse, error) {
	return &domain.CreateFormResponse{FormID: "form-1"}, nil
}
func (stubScript) CloseForm(context.Context, string) error { return nil }

func newTestServer(source domain.TestSource) (http.Handler, *stubAttempts, *stubInterns) {
	attempts := &stubAttempts{}
	interns := &stubInterns{}
	branches := stubBranches{}
	forms := stubForms{}

	srv := NewServer(Deps{
		Catalog:       usecase.NewCatalog(source),
		Forms:         usecase.NewForms(source, stubScript{}, forms, branches, ""),
		Submissions:   usecase.NewSubmissions(source, forms, branches, attempts),
		Grading:       usecase.NewGrading(attempts, source),
		Interns:       usecase.NewInterns(interns, attempts),
		Branches:      branches,
		WebhookSecret: testSecret,
	})
	return srv.Router(), attempts, interns
}

func do(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	h, _, _ := newTestServer(stubSource{})
	rec := do(t, h, http.MethodGet, "/health", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("код %d, ожидался 200", rec.Code)
	}
}

func TestWebhookRejectsWrongSecret(t *testing.T) {
	h, _, _ := newTestServer(stubSource{})

	cases := map[string]map[string]string{
		"без секрета":      nil,
		"пустой секрет":    {WebhookSecretHeader: ""},
		"неверный секрет":  {WebhookSecretHeader: "не тот"},
		"секрет с префикс": {WebhookSecretHeader: testSecret + "x"},
	}
	for name, headers := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/api/v1/webhook/ping", "{}", headers)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("код %d, ожидался 401", rec.Code)
			}
		})
	}
}

func TestWebhookPingAcceptsCorrectSecret(t *testing.T) {
	h, _, _ := newTestServer(stubSource{})
	rec := do(t, h, http.MethodPost, "/api/v1/webhook/ping", "{}",
		map[string]string{WebhookSecretHeader: testSecret})
	if rec.Code != http.StatusOK {
		t.Fatalf("код %d, ожидался 200: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookSubmissionValidatesBody(t *testing.T) {
	h, _, _ := newTestServer(stubSource{})
	headers := map[string]string{WebhookSecretHeader: testSecret}

	rec := do(t, h, http.MethodPost, "/api/v1/webhook/submission",
		`{"form_id":"form-1","response_id":"r1"}`, headers)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("код %d, ожидался 400: %s", rec.Code, rec.Body.String())
	}

	rec = do(t, h, http.MethodPost, "/api/v1/webhook/submission",
		`{"form_id":"нет такой","response_id":"r1","phone":"+992900000000"}`, headers)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("код %d, ожидался 404: %s", rec.Code, rec.Body.String())
	}
}

func TestTestsReturns503WhenSourceDown(t *testing.T) {
	h, _, _ := newTestServer(stubSource{err: domain.ErrSourceUnavailable})
	rec := do(t, h, http.MethodGet, "/api/v1/tests", "", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("код %d, ожидался 503", rec.Code)
	}
}

func TestCreateFormRejectsBadDuration(t *testing.T) {
	h, _, _ := newTestServer(stubSource{})
	rec := do(t, h, http.MethodPost, "/api/v1/admin/forms", `{"test_id":35,"duration_minutes":0}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("код %d, ожидался 400: %s", rec.Code, rec.Body.String())
	}
}

func TestPageSizeIsCapped(t *testing.T) {
	h, _, interns := newTestServer(stubSource{})
	do(t, h, http.MethodGet, "/api/v1/interns?limit=100000", "", nil)
	if interns.lastFilter.Limit != domain.MaxPageSize {
		t.Fatalf("лимит %d, ожидался %d", interns.lastFilter.Limit, domain.MaxPageSize)
	}
}

func TestAttemptFilterRejectsUnknownStatus(t *testing.T) {
	h, _, _ := newTestServer(stubSource{})
	rec := do(t, h, http.MethodGet, "/api/v1/interns/1?status=что-то", "", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("код %d, ожидался 400", rec.Code)
	}
}

func TestUnknownRouteReturnsJSON(t *testing.T) {
	h, _, _ := newTestServer(stubSource{})
	rec := do(t, h, http.MethodGet, "/api/v1/чего-то-нет", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("код %d, ожидался 404", rec.Code)
	}
	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error == "" {
		t.Fatalf("ожидался JSON с описанием ошибки, получено %q", rec.Body.String())
	}
}

func TestPanicIsContained(t *testing.T) {
	rec := httptest.NewRecorder()
	handler := recoverMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("что-то пошло не так")
	}))
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/боль", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("код %d, ожидался 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "что-то пошло не так") {
		t.Fatal("текст внутренней ошибки не должен уходить наружу")
	}
}

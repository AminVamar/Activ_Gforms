package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gform/internal/client/testsource"
	"gform/internal/domain"
	"gform/internal/repository/postgres"
	"gform/internal/usecase"
	"gform/migrations"
)

func TestE2EFullFlow(t *testing.T) {
	if os.Getenv("GFORM_E2E") != "1" {
		t.Skip("сквозной тест выключен: задайте GFORM_E2E=1")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("для сквозного теста нужна DATABASE_URL")
	}
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		secret = "e2e-секрет"
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("подключение к базе: %v", err)
	}
	defer pool.Close()

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("миграции: %v", err)
	}

	sourceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(e2eSourcePage))
	}))
	defer sourceSrv.Close()

	source := testsource.New(sourceSrv.URL, 50, 5*time.Second, 0)
	branchRepo := postgres.NewBranchRepo(pool)
	formRepo := postgres.NewFormRepo(pool)
	internRepo := postgres.NewInternRepo(pool)
	attemptRepo := postgres.NewAttemptRepo(pool)

	formID := fmt.Sprintf("e2e-form-%d", time.Now().UnixNano())
	form := &domain.FormSession{
		FormID: formID, TestID: 900, TestTitle: "Сквозной тест", DurationMinutes: 15,
		FormURL: "https://forms.test/view", RespondentURL: "https://script.test/exec",
		ItemMap:       map[string]int64{"i1": 901, "i2": 902, "i3": 903},
		QuestionCount: 3,
	}
	if err := formRepo.Create(ctx, form); err != nil {
		t.Fatalf("создание формы: %v", err)
	}

	srv := NewServer(Deps{
		Catalog:       usecase.NewCatalog(source),
		Forms:         usecase.NewForms(source, nil, formRepo, branchRepo, ""),
		Submissions:   usecase.NewSubmissions(source, formRepo, branchRepo, attemptRepo),
		Grading:       usecase.NewGrading(attemptRepo, source),
		Interns:       usecase.NewInterns(internRepo, attemptRepo),
		Branches:      branchRepo,
		WebhookSecret: secret,
	})
	h := srv.Router()
	auth := map[string]string{WebhookSecretHeader: secret}

	responseID := fmt.Sprintf("e2e-resp-%d", time.Now().UnixNano())
	phone := fmt.Sprintf("+99290%07d", time.Now().UnixNano()%10000000)
	body := fmt.Sprintf(`{
		"form_id": %q, "response_id": %q,
		"last_name": "Каримов", "first_name": "Далер", "middle_name": "",
		"phone": %q, "branch_code": "5100", "branch_name": "Филиал Садбарг",
		"auto_submitted": true,
		"answers": [
			{"item_id": "i1", "answer": "Да"},
			{"item_id": "i2", "answer": "развёрнутый ответ"}
		]
	}`, formID, responseID, phone)

	rec := do(t, h, http.MethodPost, "/api/v1/webhook/submission", body, auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("приём ответа: код %d, тело %s", rec.Code, rec.Body.String())
	}
	var accepted usecase.SubmissionResult
	mustJSON(t, rec.Body.Bytes(), &accepted)

	if !accepted.Scored || accepted.Score != 1 || accepted.Total != 3 {
		t.Fatalf("итог = %d из %d (scored=%v), ожидалось 1 из 3", accepted.Score, accepted.Total, accepted.Scored)
	}
	if !accepted.NeedsGrading {
		t.Fatal("открытый вопрос и шкала должны отправить попытку на ручную проверку")
	}

	rec = do(t, h, http.MethodPost, "/api/v1/webhook/submission", body, auth)
	var duplicate usecase.SubmissionResult
	mustJSON(t, rec.Body.Bytes(), &duplicate)
	if !duplicate.Duplicate || duplicate.AttemptID != accepted.AttemptID {
		t.Fatalf("повтор создал новую попытку: %+v", duplicate)
	}

	rec = do(t, h, http.MethodGet, "/api/v1/admin/grading/pending?limit=200", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("очередь: код %d", rec.Code)
	}

	rec = do(t, h, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/grading/attempts/%d?only_pending=true", accepted.AttemptID), "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("попытка для проверки: код %d, тело %s", rec.Code, rec.Body.String())
	}
	var view usecase.AttemptView
	mustJSON(t, rec.Body.Bytes(), &view)
	if len(view.Answers) != 2 {
		t.Fatalf("на проверке %d ответов, ожидалось 2", len(view.Answers))
	}

	grades := fmt.Sprintf(`{"grades":[{"answer_id":%d,"score":1},{"answer_id":%d,"score":0}]}`,
		view.Answers[0].ID, view.Answers[1].ID)
	rec = do(t, h, http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/grading/attempts/%d", accepted.AttemptID), grades, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("выставление баллов: код %d, тело %s", rec.Code, rec.Body.String())
	}
	var graded GradeResponse
	mustJSON(t, rec.Body.Bytes(), &graded)
	if graded.Status != domain.StatusGraded {
		t.Fatalf("статус = %s, ожидался graded", graded.Status)
	}
	if graded.Score != 2 || graded.Total != 3 {
		t.Fatalf("итог = %d из %d, ожидалось 2 из 3", graded.Score, graded.Total)
	}

	bad := fmt.Sprintf(`{"grades":[{"answer_id":%d,"score":5}]}`, view.Answers[0].ID)
	rec = do(t, h, http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/grading/attempts/%d", accepted.AttemptID), bad, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("балл 5: код %d, ожидался 400", rec.Code)
	}

	rec = do(t, h, http.MethodPost,
		fmt.Sprintf("/api/v1/attempts/%d/rescore", accepted.AttemptID), "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("пересчёт: код %d, тело %s", rec.Code, rec.Body.String())
	}
	var rescored usecase.RescoreResult
	mustJSON(t, rec.Body.Bytes(), &rescored)
	if rescored.Changed != 0 || rescored.Score != 2 {
		t.Fatalf("пересчёт изменил %d ответов, итог %d: ручные баллы должны сохраниться",
			rescored.Changed, rescored.Score)
	}

	rec = do(t, h, http.MethodGet, "/api/v1/interns?search="+phone[1:], "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("список стажёров: код %d", rec.Code)
	}
	var list struct {
		Items []domain.InternListItem `json:"items"`
		Total int                     `json:"total"`
	}
	mustJSON(t, rec.Body.Bytes(), &list)
	if list.Total == 0 {
		t.Fatal("стажёр не найден по номеру телефона")
	}
}

func mustJSON(t *testing.T, body []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(body, dst); err != nil {
		t.Fatalf("разбор ответа: %v (тело: %s)", err, string(body))
	}
}

const e2eSourcePage = `{
  "pageNumber": 1, "pageSize": 10, "totalPages": 1, "totalRecords": 1, "statusCode": 200,
  "data": [
    {"id": 900, "title": "Сквозной тест", "description": "", "testType": 0,
     "createdAt": "2026-01-01T00:00:00Z",
     "questions": [
       {"id": 901, "content": "Закрытый вопрос", "type": 1, "options": [
         {"id": 1, "text": "Да", "isCorrect": true,  "questionId": 901},
         {"id": 2, "text": "Нет", "isCorrect": false, "questionId": 901}]},
       {"id": 902, "content": "Открытый вопрос", "type": 2, "options": null},
       {"id": 903, "content": "Шкала", "type": 3, "options": null}
     ]}
  ]
}`

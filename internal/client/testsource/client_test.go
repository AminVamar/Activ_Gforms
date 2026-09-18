package testsource

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"gform/internal/domain"
)

const page = `{
  "pageNumber": 1, "pageSize": 10, "totalPages": 2, "totalRecords": 19, "statusCode": 200,
  "data": [
    {"id": 35, "title": "Gallup", "description": "", "testType": 0,
     "createdAt": "2026-08-03T05:11:53.135315Z",
     "questions": [
       {"id": 40, "content": "Закрытый", "type": 1, "options": [
         {"id": 54, "text": "1", "isCorrect": false, "questionId": 40},
         {"id": 55, "text": "2", "isCorrect": true,  "questionId": 40}]},
       {"id": 12, "content": "Шкала", "type": 3, "options": null}
     ]}
  ]
}`

func TestListStopsOnRepeatedPage(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	c := New(srv.URL, 50, 5*time.Second, time.Minute)
	tests, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tests) != 1 {
		t.Fatalf("получено %d тестов, ожидался 1", len(tests))
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("источник опрошен %d раз, ожидалось 2 (вторая страница — повтор первой)", got)
	}
	if len(tests[0].Questions) != 2 || tests[0].Questions[1].Type != domain.QuestionScale {
		t.Fatal("вопросы разобраны неверно")
	}
	if !tests[0].Questions[0].Options[1].IsCorrect {
		t.Fatal("признак правильного варианта потерян")
	}
}

func TestListUsesCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	c := New(srv.URL, 50, 5*time.Second, time.Minute)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := c.List(ctx); err != nil {
			t.Fatalf("List: %v", err)
		}
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("источник опрошен %d раз: кэш не работает", got)
	}
}

func TestGetFindsTestAndReportsMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	c := New(srv.URL, 50, 5*time.Second, time.Minute)
	ctx := context.Background()

	test, err := c.Get(ctx, 35)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if test.Title != "Gallup" {
		t.Fatalf("получен тест %q", test.Title)
	}

	if _, err := c.Get(ctx, 404404); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидалось «не найдено», получено %v", err)
	}
}

func TestListReportsSourceUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, 50, 5*time.Second, time.Minute)
	if _, err := c.List(context.Background()); !errors.Is(err, domain.ErrSourceUnavailable) {
		t.Fatalf("ожидалась недоступность источника, получено %v", err)
	}
}

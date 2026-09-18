package testsource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"gform/internal/domain"
)

const maxPages = 50

type Client struct {
	baseURL  string
	pageSize int
	http     *http.Client
	ttl      time.Duration

	mu       sync.Mutex
	cache    []domain.Test
	cachedAt time.Time
}

func New(baseURL string, pageSize int, timeout, cacheTTL time.Duration) *Client {
	return &Client{
		baseURL:  baseURL,
		pageSize: pageSize,
		ttl:      cacheTTL,
		http:     &http.Client{Timeout: timeout},
	}
}

type pageResponse struct {
	PageNumber   int          `json:"pageNumber"`
	PageSize     int          `json:"pageSize"`
	TotalPages   int          `json:"totalPages"`
	TotalRecords int          `json:"totalRecords"`
	StatusCode   int          `json:"statusCode"`
	Message      *string      `json:"message"`
	Data         []sourceTest `json:"data"`
}

type sourceTest struct {
	ID          int64            `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	TestType    int              `json:"testType"`
	CreatedAt   time.Time        `json:"createdAt"`
	Questions   []sourceQuestion `json:"questions"`
}

type sourceQuestion struct {
	ID      int64          `json:"id"`
	Content string         `json:"content"`
	Type    int            `json:"type"`
	Options []sourceOption `json:"options"`
}

type sourceOption struct {
	ID         int64  `json:"id"`
	Text       string `json:"text"`
	IsCorrect  bool   `json:"isCorrect"`
	QuestionID int64  `json:"questionId"`
}

func (c *Client) List(ctx context.Context) ([]domain.Test, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cache != nil && time.Since(c.cachedAt) < c.ttl {
		return c.cache, nil
	}

	tests, err := c.fetchAll(ctx)
	if err != nil {
		return nil, err
	}

	c.cache = tests
	c.cachedAt = time.Now()
	return tests, nil
}

func (c *Client) Get(ctx context.Context, id int64) (*domain.Test, error) {
	tests, err := c.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tests {
		if tests[i].ID == id {
			t := tests[i]
			return &t, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (c *Client) fetchAll(ctx context.Context) ([]domain.Test, error) {
	var (
		out  []domain.Test
		seen = make(map[int64]bool)
	)

	for page := 1; page <= maxPages; page++ {
		resp, err := c.fetchPage(ctx, page)
		if err != nil {
			return nil, err
		}

		fresh := 0
		for _, t := range resp.Data {
			if seen[t.ID] {
				continue
			}
			seen[t.ID] = true
			fresh++
			out = append(out, convertTest(t))
		}

		if fresh == 0 || resp.TotalPages <= page {
			break
		}
	}

	return out, nil
}

func (c *Client) fetchPage(ctx context.Context, page int) (*pageResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: некорректный TEST_SOURCE_URL: %v", domain.ErrSourceUnavailable, err)
	}
	q := u.Query()
	q.Set("pageNumber", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(c.pageSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrSourceUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrSourceUnavailable, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: чтение ответа: %v", domain.ErrSourceUnavailable, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: источник ответил %d", domain.ErrSourceUnavailable, res.StatusCode)
	}

	var parsed pageResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: неожиданный формат ответа: %v", domain.ErrSourceUnavailable, err)
	}
	return &parsed, nil
}

func convertTest(t sourceTest) domain.Test {
	out := domain.Test{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		TestType:    t.TestType,
		CreatedAt:   t.CreatedAt,
		Questions:   make([]domain.Question, 0, len(t.Questions)),
	}
	for _, q := range t.Questions {
		question := domain.Question{
			ID:      q.ID,
			Content: q.Content,
			Type:    q.Type,
			Options: make([]domain.Option, 0, len(q.Options)),
		}
		for _, o := range q.Options {
			question.Options = append(question.Options, domain.Option{
				ID:         o.ID,
				Text:       o.Text,
				IsCorrect:  o.IsCorrect,
				QuestionID: o.QuestionID,
			})
		}
		out.Questions = append(out.Questions, question)
	}
	return out
}

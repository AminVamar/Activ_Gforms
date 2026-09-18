package appscript

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gform/internal/domain"
)

var ErrNotConfigured = errors.New("APPSCRIPT_WEBAPP_URL не задан: создание форм недоступно")

type Client struct {
	webAppURL string
	secret    string
	http      *http.Client
}

func New(webAppURL, secret string, timeout time.Duration) *Client {
	return &Client{
		webAppURL: webAppURL,
		secret:    secret,
		http: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type envelope struct {
	Action  string      `json:"action"`
	Secret  string      `json:"secret"`
	Payload interface{} `json:"payload"`
}

type reply struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error"`
	Data  json.RawMessage `json:"data"`
}

func (c *Client) CreateForm(ctx context.Context, req domain.CreateFormRequest) (*domain.CreateFormResponse, error) {
	raw, err := c.retry(ctx, "createForm", req, retryUndelivered)
	if err != nil {
		return nil, err
	}
	var out domain.CreateFormResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("apps script вернул неожиданный ответ: %w", err)
	}
	if out.FormID == "" {
		return nil, errors.New("apps script не вернул form_id")
	}
	if out.ItemMap == nil {
		out.ItemMap = map[string]int64{}
	}
	return &out, nil
}

func (c *Client) CloseForm(ctx context.Context, formID string) error {
	_, err := c.retry(ctx, "closeForm", map[string]string{"form_id": formID}, retryAnyTransient)
	return err
}

const (
	retryUndelivered = iota
	retryAnyTransient
)

const retryAttempts = 3

type transientError struct {
	err       error
	delivered bool
}

func (e *transientError) Error() string { return e.err.Error() }
func (e *transientError) Unwrap() error { return e.err }

func (c *Client) retry(ctx context.Context, action string, payload interface{}, mode int) (json.RawMessage, error) {
	var lastErr error

	for attempt := 1; attempt <= retryAttempts; attempt++ {
		raw, err := c.call(ctx, action, payload)
		if err == nil {
			return raw, nil
		}
		lastErr = err

		var tr *transientError
		if !errors.As(err, &tr) {
			return nil, err
		}
		if mode == retryUndelivered && tr.delivered {
			return nil, err
		}
		if attempt == retryAttempts || ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}

	return nil, lastErr
}

func (c *Client) call(ctx context.Context, action string, payload interface{}) (json.RawMessage, error) {
	if c.webAppURL == "" {
		return nil, ErrNotConfigured
	}

	body, err := json.Marshal(envelope{Action: action, Secret: c.secret, Payload: payload})
	if err != nil {
		return nil, fmt.Errorf("сборка запроса к apps script: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webAppURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("запрос к apps script: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain;charset=utf-8")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, &transientError{err: fmt.Errorf("apps script недоступен: %w", err)}
	}
	if loc := res.Header.Get("Location"); res.StatusCode/100 == 3 && loc != "" {
		res.Body.Close()
		if res, err = c.fetchResult(ctx, loc); err != nil {
			return nil, &transientError{err: fmt.Errorf("apps script: %w", err), delivered: true}
		}
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, &transientError{
			err:       fmt.Errorf("чтение ответа apps script: %w", err),
			delivered: true,
		}
	}
	if res.StatusCode != http.StatusOK {
		callErr := fmt.Errorf("apps script ответил %d: %s", res.StatusCode, truncate(string(respBody), 300))
		if res.StatusCode >= 500 {
			return nil, &transientError{err: callErr, delivered: true}
		}
		return nil, callErr
	}

	var r reply
	if err := json.Unmarshal(respBody, &r); err != nil {
		if isHTML(respBody) {
			return nil, &transientError{delivered: true, err: errors.New(
				"apps script вернул HTML-страницу вместо ответа. " +
					"Проверьте: развёрнута ли НОВАЯ версия кода, стоит ли доступ «Все», " +
					"и нет ли ошибки в журнале выполнения Apps Script")}
		}
		return nil, &transientError{
			err:       fmt.Errorf("apps script вернул не JSON: %s", truncate(string(respBody), 300)),
			delivered: true,
		}
	}
	if !r.OK {
		return nil, fmt.Errorf("apps script: %s", r.Error)
	}
	return r.Data, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func isHTML(body []byte) bool {
	head := strings.ToLower(strings.TrimSpace(string(body)))
	if len(head) > 200 {
		head = head[:200]
	}
	return strings.HasPrefix(head, "<!doctype") || strings.HasPrefix(head, "<html")
}

const resultAttempts = 5

func (c *Client) fetchResult(ctx context.Context, url string) (*http.Response, error) {
	follow := &http.Client{Timeout: c.http.Timeout}
	var res *http.Response
	for attempt := 1; attempt <= resultAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		res, err = follow.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusNotFound || attempt == resultAttempts {
			return res, nil
		}
		res.Body.Close()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * 2 * time.Second):
		}
	}
	return res, nil
}

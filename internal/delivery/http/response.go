package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"gform/internal/client/appscript"
	"gform/internal/domain"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type ListResponse struct {
	Items  any `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("не удалось записать ответ", "error", err)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		slog.Warn("некорректный запрос", "error", err, "path", r.URL.Path)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: cleanMessage(err)})
	case errors.Is(err, domain.ErrUnauthorized):
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "неверный секрет"})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "не найдено"})
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrFormClosed):
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: cleanMessage(err)})
	case errors.Is(err, domain.ErrSourceUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "источник тестов недоступен"})
	case errors.Is(err, appscript.ErrNotConfigured):
		writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
	case isAppScriptFailure(err):
		writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: cleanMessage(err)})
	default:
		slog.Error("внутренняя ошибка", "error", err, "path", r.URL.Path)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка"})
	}
}

func isAppScriptFailure(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "apps script") || strings.Contains(msg, "Google-формы")
}

func cleanMessage(err error) string {
	msg := err.Error()
	for _, prefix := range []string{
		domain.ErrValidation.Error() + ": ",
		domain.ErrConflict.Error() + ": ",
	} {
		msg = strings.TrimPrefix(msg, prefix)
	}
	return msg
}

func pathID(r *http.Request, name string) (int64, error) {
	raw := muxVar(r, name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ErrNotFound
	}
	return id, nil
}

func pagination(r *http.Request) domain.Pagination {
	p := domain.Pagination{
		Limit:  queryInt(r, "limit", 50),
		Offset: queryInt(r, "offset", 0),
	}
	p.Normalize()
	return p
}

func queryInt(r *http.Request, name string, def int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func queryInt64Ptr(r *http.Request, name string) *int64 {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func queryBool(r *http.Request, name string) bool {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	return raw == "1" || strings.EqualFold(raw, "true")
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 8<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.ErrValidation
	}
	return nil
}

func decodeJSONLoose(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 8<<20))
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	return nil
}

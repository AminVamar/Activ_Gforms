package http

import (
	"net/http"
	"strings"

	"gform/internal/domain"
)

type CreateFormRequest struct {
	TestID          int64 `json:"test_id" example:"35"`
	DurationMinutes int   `json:"duration_minutes" example:"30"`
}

// @Summary		Создать форму
// @Description	Создаёт Google-форму по тесту. Стажёрам отдавать respondent_url.
// @Tags			формы
// @Accept			json
// @Produce		json
// @Param			body	body		CreateFormRequest	true	"тест и время на прохождение"
// @Success		201		{object}	domain.FormSession
// @Failure		400		{object}	ErrorResponse	"неверные данные"
// @Failure		404		{object}	ErrorResponse	"тест не найден"
// @Failure		503		{object}	ErrorResponse	"сервис недоступен"
// @Router			/api/v1/admin/forms [post]
func (s *Server) createForm(w http.ResponseWriter, r *http.Request) {
	var req CreateFormRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	form, err := s.forms.Create(r.Context(), req.TestID, req.DurationMinutes)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, form)
}

// @Summary	Список созданных форм
// @Tags		формы
// @Produce	json
// @Param		status	query		string	false	"open или closed"
// @Param		limit	query		int		false	"лимит (до 200)"
// @Param		offset	query		int		false	"смещение"
// @Success	200		{object}	ListResponse{items=[]domain.FormSession}
// @Failure	400		{object}	ErrorResponse
// @Router		/api/v1/admin/forms [get]
func (s *Server) listForms(w http.ResponseWriter, r *http.Request) {
	p := pagination(r)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	items, total, err := s.forms.List(r.Context(), status, p)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ListResponse{Items: items, Total: total, Limit: p.Limit, Offset: p.Offset})
}

// @Summary	Форма по идентификатору
// @Tags		формы
// @Produce	json
// @Param		id	path		int	true	"id"
// @Success	200	{object}	domain.FormSession
// @Failure	404	{object}	ErrorResponse
// @Router		/api/v1/admin/forms/{id} [get]
func (s *Server) getForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	form, err := s.forms.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, form)
}

// @Summary		Закрыть форму
// @Description	Форма перестаёт принимать ответы.
// @Tags			формы
// @Produce		json
// @Param			id	path		int	true	"id"
// @Success		200	{object}	domain.FormSession
// @Failure		404	{object}	ErrorResponse
// @Failure		503	{object}	ErrorResponse	"Apps Script недоступен"
// @Router			/api/v1/admin/forms/{id}/close [post]
func (s *Server) closeForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	form, err := s.forms.Close(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, form)
}

// @Summary		Приём ответа стажёра
// @Description	Вызывается из Apps Script, секрет в X-Gform-Secret.
// @Tags			вебхук
// @Accept			json
// @Produce		json
// @Param			X-Gform-Secret	header		string				true	"общий секрет"
// @Param			body			body		domain.Submission	true	"ответ стажёра"
// @Success		200				{object}	SubmissionResponse
// @Failure		400				{object}	ErrorResponse
// @Failure		401				{object}	ErrorResponse	"неверный секрет"
// @Failure		404				{object}	ErrorResponse	"форма не найдена"
// @Router			/api/v1/webhook/submission [post]
func (s *Server) webhookSubmission(w http.ResponseWriter, r *http.Request) {
	var sub domain.Submission
	if err := decodeJSONLoose(r, &sub); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := s.submissions.Accept(r.Context(), &sub)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type PingResponse struct {
	OK bool `json:"ok" example:"true"`
}

// @Summary	Проверка секрета вебхука
// @Tags		вебхук
// @Produce	json
// @Param		X-Gform-Secret	header		string	true	"общий секрет"
// @Success	200				{object}	PingResponse
// @Failure	401				{object}	ErrorResponse
// @Router		/api/v1/webhook/ping [post]
func (s *Server) webhookPing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, PingResponse{OK: true})
}

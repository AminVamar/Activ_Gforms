package http

import (
	"net/http"
	"strings"

	"gform/internal/domain"
)

// @Summary		Стажёры
// @Description	Поиск по ФИО или телефону.
// @Tags			стажёры
// @Produce		json
// @Param			search		query		string	false	"часть ФИО или телефона"
// @Param			branch_id	query		int		false	"фильтр по филиалу"
// @Param			limit		query		int		false	"лимит (до 200)"
// @Param			offset		query		int		false	"смещение"
// @Success		200			{object}	ListResponse{items=[]domain.InternListItem}
// @Router			/api/v1/interns [get]
func (s *Server) listInterns(w http.ResponseWriter, r *http.Request) {
	f := domain.InternFilter{
		Search:     strings.TrimSpace(r.URL.Query().Get("search")),
		BranchID:   queryInt64Ptr(r, "branch_id"),
		Pagination: pagination(r),
	}
	items, total, err := s.interns.List(r.Context(), f)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ListResponse{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset})
}

// @Summary	Стажёр и его попытки
// @Tags		стажёры
// @Produce	json
// @Param		id		path		int		true	"id"
// @Param		test_id	query		int		false	"фильтр по тесту"
// @Param		status	query		string	false	"graded или needs_grading"
// @Param		limit	query		int		false	"лимит"
// @Param		offset	query		int		false	"смещение"
// @Success	200		{object}	InternCardResponse
// @Failure	400		{object}	ErrorResponse
// @Failure	404		{object}	ErrorResponse
// @Router		/api/v1/interns/{id} [get]
func (s *Server) getIntern(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	f, err := attemptFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	card, err := s.interns.Card(r.Context(), id, f)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, card)
}

// @Summary		Попытка целиком
// @Description	Все ответы и итоговый балл.
// @Tags			попытки
// @Produce		json
// @Param			id	path		int	true	"id"
// @Success		200	{object}	AttemptResponse
// @Failure		404	{object}	ErrorResponse
// @Router			/api/v1/attempts/{id} [get]
func (s *Server) getAttempt(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	view, err := s.grading.Attempt(r.Context(), id, false)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// @Summary		Пересчёт автоматических баллов
// @Description	Ручные баллы не меняются.
// @Tags			попытки
// @Produce		json
// @Param			id	path		int	true	"id"
// @Success		200	{object}	RescoreResponse
// @Failure		404	{object}	ErrorResponse
// @Failure		503	{object}	ErrorResponse	"источник тестов недоступен"
// @Router			/api/v1/attempts/{id}/rescore [post]
func (s *Server) rescoreAttempt(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	res, err := s.grading.Rescore(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func attemptFilter(r *http.Request) (domain.AttemptFilter, error) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != domain.StatusGraded && status != domain.StatusNeedsGrading {
		return domain.AttemptFilter{}, domain.ErrValidation
	}
	return domain.AttemptFilter{
		TestID:     queryInt64Ptr(r, "test_id"),
		BranchID:   queryInt64Ptr(r, "branch_id"),
		Status:     status,
		Pagination: pagination(r),
	}, nil
}

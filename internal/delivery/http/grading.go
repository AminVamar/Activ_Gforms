package http

import (
	"net/http"

	"gform/internal/domain"
)

// @Summary		Очередь на ручную проверку
// @Description	Попытки с непроверенными ответами.
// @Tags			проверка
// @Produce		json
// @Param			branch_id	query		int	false	"фильтр по филиалу"
// @Param			test_id		query		int	false	"фильтр по тесту"
// @Param			limit		query		int	false	"лимит (до 200)"
// @Param			offset		query		int	false	"смещение"
// @Success		200			{object}	ListResponse{items=[]domain.Attempt}
// @Router			/api/v1/admin/grading/pending [get]
func (s *Server) pendingGrading(w http.ResponseWriter, r *http.Request) {
	f := domain.AttemptFilter{
		TestID:     queryInt64Ptr(r, "test_id"),
		BranchID:   queryInt64Ptr(r, "branch_id"),
		Pagination: pagination(r),
	}
	items, total, err := s.grading.Pending(r.Context(), f)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ListResponse{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset})
}

// @Summary		Попытка для проверки
// @Description	Ответы стажёра с правильными ответами.
// @Tags			проверка
// @Produce		json
// @Param			id				path		int		true	"id"
// @Param			only_pending	query		boolean	false	"только непроверенные"
// @Success		200				{object}	AttemptResponse
// @Failure		404				{object}	ErrorResponse
// @Router			/api/v1/admin/grading/attempts/{id} [get]
func (s *Server) gradingAttempt(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	view, err := s.grading.Attempt(r.Context(), id, queryBool(r, "only_pending"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

type GradeRequest struct {
	Grades []domain.GradeInput `json:"grades"`
}

type GradeResponse struct {
	AttemptID int64            `json:"attempt_id"`
	Changed   int              `json:"changed"`
	Status    string           `json:"status" example:"graded"`
	Score     int              `json:"score"`
	Total     int              `json:"total"`
	Attempt   *AttemptResponse `json:"attempt"`
}

// @Summary		Выставить баллы
// @Description	Балл: 0, 1 или null.
// @Tags			проверка
// @Accept			json
// @Produce		json
// @Param			id		path		int				true	"id"
// @Param			body	body		GradeRequest	true	"баллы"
// @Success		200		{object}	GradeResponse
// @Failure		400		{object}	ErrorResponse	"неверный балл"
// @Failure		404		{object}	ErrorResponse
// @Router			/api/v1/admin/grading/attempts/{id} [patch]
func (s *Server) gradeAttempt(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req GradeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	view, changed, err := s.grading.Grade(r.Context(), id, req.Grades)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, GradeResponse{
		AttemptID: id,
		Changed:   changed,
		Status:    view.Status,
		Score:     view.Score,
		Total:     view.Total,
		Attempt:   view,
	})
}

package http

import (
	"net/http"
	"strings"

	"gform/internal/domain"
)

type TestsResponse struct {
	Items []domain.Test `json:"items"`
	Total int           `json:"total"`
}

// @Summary		Все тесты из источника
// @Description	Список тестов из внешнего источника.
// @Tags			тесты
// @Produce		json
// @Success		200	{object}	TestsResponse
// @Failure		503	{object}	ErrorResponse	"источник тестов недоступен"
// @Router			/api/v1/tests [get]
func (s *Server) listTests(w http.ResponseWriter, r *http.Request) {
	tests, err := s.catalog.List(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, TestsResponse{Items: tests, Total: len(tests)})
}

// @Summary	Тест по идентификатору
// @Tags		тесты
// @Produce	json
// @Param		id	path		int	true	"id"
// @Success	200	{object}	domain.Test
// @Failure	404	{object}	ErrorResponse
// @Failure	503	{object}	ErrorResponse	"источник тестов недоступен"
// @Router		/api/v1/tests/{id} [get]
func (s *Server) getTest(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	test, err := s.catalog.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, test)
}

type BranchesResponse struct {
	Items []domain.Branch `json:"items"`
	Total int             `json:"total"`
}

// @Summary	Филиалы
// @Tags		филиалы
// @Produce	json
// @Success	200	{object}	BranchesResponse
// @Router		/api/v1/branches [get]
func (s *Server) listBranches(w http.ResponseWriter, r *http.Request) {
	branches, err := s.branches.List(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, BranchesResponse{Items: branches, Total: len(branches)})
}

type CreateBranchRequest struct {
	Code string `json:"code" example:"6900"`
	Name string `json:"name" example:"ЦБО Рудаки"`
}

// @Summary	Добавить филиал
// @Tags		филиалы
// @Accept		json
// @Produce	json
// @Param		body	body		CreateBranchRequest	true	"филиал"
// @Success	201		{object}	domain.Branch
// @Failure	400		{object}	ErrorResponse
// @Router		/api/v1/branches [post]
func (s *Server) createBranch(w http.ResponseWriter, r *http.Request) {
	var req CreateBranchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "код и название филиала обязательны"})
		return
	}
	branch, err := s.branches.Create(r.Context(), req.Code, req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, branch)
}

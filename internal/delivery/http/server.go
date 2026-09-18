package http

import (
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "gform/docs"

	"gform/internal/domain"
	"gform/internal/usecase"
)

type Server struct {
	catalog     *usecase.Catalog
	forms       *usecase.Forms
	submissions *usecase.Submissions
	grading     *usecase.Grading
	interns     *usecase.Interns
	branches    domain.BranchRepository
	secret      string
}

type Deps struct {
	Catalog       *usecase.Catalog
	Forms         *usecase.Forms
	Submissions   *usecase.Submissions
	Grading       *usecase.Grading
	Interns       *usecase.Interns
	Branches      domain.BranchRepository
	WebhookSecret string
}

func NewServer(d Deps) *Server {
	return &Server{
		catalog:     d.Catalog,
		forms:       d.Forms,
		submissions: d.Submissions,
		grading:     d.Grading,
		interns:     d.Interns,
		branches:    d.Branches,
		secret:      d.WebhookSecret,
	}
}

func (s *Server) Router() http.Handler {
	r := mux.NewRouter()
	r.Use(recoverMiddleware, loggingMiddleware)

	r.HandleFunc("/health", s.health).Methods(http.MethodGet)
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	api := r.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/tests", s.listTests).Methods(http.MethodGet)
	api.HandleFunc("/tests/{id}", s.getTest).Methods(http.MethodGet)

	api.HandleFunc("/branches", s.listBranches).Methods(http.MethodGet)
	api.HandleFunc("/branches", s.createBranch).Methods(http.MethodPost)

	api.HandleFunc("/admin/forms", s.createForm).Methods(http.MethodPost)
	api.HandleFunc("/admin/forms", s.listForms).Methods(http.MethodGet)
	api.HandleFunc("/admin/forms/{id}", s.getForm).Methods(http.MethodGet)
	api.HandleFunc("/admin/forms/{id}/close", s.closeForm).Methods(http.MethodPost)

	api.HandleFunc("/interns", s.listInterns).Methods(http.MethodGet)
	api.HandleFunc("/interns/{id}", s.getIntern).Methods(http.MethodGet)

	api.HandleFunc("/attempts/{id}", s.getAttempt).Methods(http.MethodGet)
	api.HandleFunc("/attempts/{id}/rescore", s.rescoreAttempt).Methods(http.MethodPost)

	api.HandleFunc("/admin/grading/pending", s.pendingGrading).Methods(http.MethodGet)
	api.HandleFunc("/admin/grading/attempts/{id}", s.gradingAttempt).Methods(http.MethodGet)
	api.HandleFunc("/admin/grading/attempts/{id}", s.gradeAttempt).Methods(http.MethodPatch)

	webhook := api.PathPrefix("/webhook").Subrouter()
	webhook.Use(webhookAuth(s.secret))
	webhook.HandleFunc("/submission", s.webhookSubmission).Methods(http.MethodPost)
	webhook.HandleFunc("/ping", s.webhookPing).Methods(http.MethodPost)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "не найдено"})
	})
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "метод не поддерживается"})
	})
	return r
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// @Summary	Health check
// @Tags		служебное
// @Produce	json
// @Success	200	{object}	HealthResponse
// @Router		/health [get]
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

type (
	SubmissionResponse = usecase.SubmissionResult

	AttemptResponse = usecase.AttemptView

	RescoreResponse = usecase.RescoreResult

	InternCardResponse = usecase.InternCard
)

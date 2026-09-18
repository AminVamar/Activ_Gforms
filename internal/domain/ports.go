package domain

import "context"

const MaxPageSize = 200

type Pagination struct {
	Limit  int
	Offset int
}

func (p *Pagination) Normalize() {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > MaxPageSize {
		p.Limit = MaxPageSize
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}

type InternFilter struct {
	Search   string
	BranchID *int64
	Pagination
}

type AttemptFilter struct {
	InternID *int64
	TestID   *int64
	BranchID *int64
	Status   string
	Pagination
}

type GradeInput struct {
	AnswerID int64 `json:"answer_id"`
	Score    *int  `json:"score"`
}

type SaveSubmissionInput struct {
	Form           *FormSession
	Sub            *Submission
	BranchID       *int64
	TotalQuestions int
	Answers        []Answer
}

type SaveSubmissionResult struct {
	AttemptID int64
	InternID  int64
	Duplicate bool
	Scored    int
	Total     int
}

type BranchRepository interface {
	List(ctx context.Context) ([]Branch, error)
	Create(ctx context.Context, code, name string) (*Branch, error)
	GetByCode(ctx context.Context, code string) (*Branch, error)
}

type FormRepository interface {
	Create(ctx context.Context, f *FormSession) error
	GetByID(ctx context.Context, id int64) (*FormSession, error)
	GetByFormID(ctx context.Context, formID string) (*FormSession, error)
	List(ctx context.Context, status string, p Pagination) ([]FormSession, int, error)
	Close(ctx context.Context, id int64) (*FormSession, error)
}

type InternRepository interface {
	List(ctx context.Context, f InternFilter) ([]InternListItem, int, error)
	GetByID(ctx context.Context, id int64) (*Intern, error)
}

type AttemptRepository interface {
	SaveSubmission(ctx context.Context, in SaveSubmissionInput) (*SaveSubmissionResult, error)
	GetByID(ctx context.Context, id int64) (*Attempt, error)
	List(ctx context.Context, f AttemptFilter) ([]Attempt, int, error)
	ListPending(ctx context.Context, f AttemptFilter) ([]Attempt, int, error)
	Answers(ctx context.Context, attemptID int64, onlyPending bool) ([]Answer, error)
	ApplyGrades(ctx context.Context, attemptID int64, grades []GradeInput, source string) (int, error)
	GradingLog(ctx context.Context, attemptID int64) ([]GradingLogEntry, error)
}

type TestSource interface {
	List(ctx context.Context) ([]Test, error)
	Get(ctx context.Context, id int64) (*Test, error)
}

type CreateFormRequest struct {
	TestID          int64                `json:"test_id"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	DurationMinutes int                  `json:"duration_minutes"`
	Branches        []CreateFormBranch   `json:"branches"`
	Questions       []CreateFormQuestion `json:"questions"`
	WebhookURL      string               `json:"webhook_url"`
}

type CreateFormBranch struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type CreateFormQuestion struct {
	QuestionID int64    `json:"question_id"`
	Content    string   `json:"content"`
	Type       int      `json:"type"`
	Options    []string `json:"options"`
}

type CreateFormResponse struct {
	FormID        string           `json:"form_id"`
	FormURL       string           `json:"form_url"`
	RespondentURL string           `json:"respondent_url"`
	EditURL       string           `json:"edit_url"`
	ItemMap       map[string]int64 `json:"item_map"`
}

type AppScriptClient interface {
	CreateForm(ctx context.Context, req CreateFormRequest) (*CreateFormResponse, error)
	CloseForm(ctx context.Context, formID string) error
}

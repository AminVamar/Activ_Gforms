package domain

import "time"

const (
	FormOpen   = "open"
	FormClosed = "closed"
)

const (
	GradedAuto   = "auto"
	GradedManual = "manual"
)

const (
	StatusGraded       = "graded"
	StatusNeedsGrading = "needs_grading"
)

type Branch struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type FormSession struct {
	ID              int64            `json:"id"`
	FormID          string           `json:"form_id"`
	TestID          int64            `json:"test_id"`
	TestTitle       string           `json:"test_title"`
	DurationMinutes int              `json:"duration_minutes"`
	FormURL         string           `json:"form_url"`
	RespondentURL   string           `json:"respondent_url"`
	EditURL         string           `json:"edit_url"`
	ItemMap         map[string]int64 `json:"item_map"`
	QuestionCount   int              `json:"question_count"`
	Status          string           `json:"status"`
	ClosedAt        *time.Time       `json:"closed_at,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
}

func (f *FormSession) IsOpen() bool { return f.Status == FormOpen }

type Intern struct {
	ID         int64     `json:"id"`
	Phone      string    `json:"phone"`
	LastName   string    `json:"last_name"`
	FirstName  string    `json:"first_name"`
	MiddleName string    `json:"middle_name,omitempty"`
	BranchID   *int64    `json:"branch_id,omitempty"`
	BranchCode string    `json:"branch_code,omitempty"`
	BranchName string    `json:"branch_name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type InternListItem struct {
	Intern
	AttemptsCount int  `json:"attempts_count"`
	NeedsGrading  bool `json:"needs_grading"`
}

type Attempt struct {
	ID             int64     `json:"id"`
	ResponseID     string    `json:"response_id"`
	FormSessionID  int64     `json:"form_session_id"`
	FormID         string    `json:"form_id"`
	InternID       int64     `json:"intern_id"`
	TestID         int64     `json:"test_id"`
	TestTitle      string    `json:"test_title"`
	BranchID       *int64    `json:"branch_id,omitempty"`
	TotalQuestions int       `json:"total_questions"`
	InternPhone    string    `json:"intern_phone,omitempty"`
	InternName     string    `json:"intern_name,omitempty"`
	BranchCode     string    `json:"branch_code,omitempty"`
	BranchName     string    `json:"branch_name,omitempty"`
	AutoSubmitted  bool      `json:"auto_submitted"`
	SubmittedAt    time.Time `json:"submitted_at"`
	CreatedAt      time.Time `json:"created_at"`

	Scored       int  `json:"scored"`
	PendingCount int  `json:"pending_count"`
	NeedsGrading bool `json:"needs_grading"`
}

func (a *Attempt) Status() string {
	if a.PendingCount > 0 {
		return StatusNeedsGrading
	}
	return StatusGraded
}

type Answer struct {
	ID           int64      `json:"id"`
	AttemptID    int64      `json:"attempt_id"`
	QuestionID   int64      `json:"question_id"`
	QuestionType int        `json:"question_type"`
	ItemID       string     `json:"item_id,omitempty"`
	AnswerText   string     `json:"answer_text"`
	Score        *int       `json:"score"`
	GradedBy     *string    `json:"graded_by,omitempty"`
	GradedAt     *time.Time `json:"graded_at,omitempty"`
	Position     int        `json:"position"`

	QuestionText   string   `json:"question_text,omitempty"`
	CorrectAnswers []string `json:"correct_answers,omitempty"`
}

type GradeChange struct {
	AnswerID  int64
	AttemptID int64
	OldScore  *int
	NewScore  *int
	Source    string
}

type GradingLogEntry struct {
	ID        int64     `json:"id"`
	AnswerID  int64     `json:"answer_id"`
	AttemptID int64     `json:"attempt_id"`
	OldScore  *int      `json:"old_score"`
	NewScore  *int      `json:"new_score"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type Submission struct {
	FormID        string            `json:"form_id"`
	ResponseID    string            `json:"response_id"`
	LastName      string            `json:"last_name"`
	FirstName     string            `json:"first_name"`
	MiddleName    string            `json:"middle_name"`
	Phone         string            `json:"phone"`
	BranchCode    string            `json:"branch_code"`
	BranchName    string            `json:"branch_name"`
	AutoSubmitted bool              `json:"auto_submitted"`
	SubmittedAt   *time.Time        `json:"submitted_at"`
	Answers       []SubmittedAnswer `json:"answers"`
}

type SubmittedAnswer struct {
	ItemID     string `json:"item_id"`
	QuestionID int64  `json:"question_id"`
	Answer     string `json:"answer"`
}

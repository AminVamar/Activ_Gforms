package domain

import "time"

const (
	QuestionChoice = 1
	QuestionOpen   = 2
	QuestionScale  = 3
)

type Test struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	TestType    int        `json:"test_type"`
	CreatedAt   time.Time  `json:"created_at"`
	Questions   []Question `json:"questions"`
}

type Question struct {
	ID      int64    `json:"id"`
	Content string   `json:"content"`
	Type    int      `json:"type"`
	Options []Option `json:"options"`
}

type Option struct {
	ID         int64  `json:"id"`
	Text       string `json:"text"`
	IsCorrect  bool   `json:"is_correct"`
	QuestionID int64  `json:"question_id"`
}

func IsAutoGraded(questionType int) bool { return questionType == QuestionChoice }

func (t *Test) QuestionByID(id int64) (Question, bool) {
	for i := range t.Questions {
		if t.Questions[i].ID == id {
			return t.Questions[i], true
		}
	}
	return Question{}, false
}

func (q Question) CorrectOptions() []string {
	var out []string
	for _, o := range q.Options {
		if o.IsCorrect {
			out = append(out, o.Text)
		}
	}
	return out
}

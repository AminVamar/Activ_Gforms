package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gform/internal/domain"
)

type Submissions struct {
	source   domain.TestSource
	forms    domain.FormRepository
	branches domain.BranchRepository
	attempts domain.AttemptRepository
}

func NewSubmissions(
	source domain.TestSource,
	forms domain.FormRepository,
	branches domain.BranchRepository,
	attempts domain.AttemptRepository,
) *Submissions {
	return &Submissions{source: source, forms: forms, branches: branches, attempts: attempts}
}

type SubmissionResult struct {
	AttemptID    int64 `json:"attempt_id"`
	InternID     int64 `json:"intern_id"`
	Duplicate    bool  `json:"duplicate"`
	Scored       bool  `json:"scored"`
	Score        int   `json:"score"`
	Total        int   `json:"total"`
	NeedsGrading bool  `json:"needs_grading"`
}

func (s *Submissions) Accept(ctx context.Context, sub *domain.Submission) (*SubmissionResult, error) {
	if err := validateSubmission(sub); err != nil {
		return nil, err
	}

	form, err := s.forms.GetByFormID(ctx, sub.FormID)
	if err != nil {
		return nil, err
	}

	branchID, err := s.resolveBranch(ctx, sub)
	if err != nil {
		return nil, err
	}

	test, sourceErr := s.source.Get(ctx, form.TestID)
	if sourceErr != nil && !errors.Is(sourceErr, domain.ErrSourceUnavailable) && !errors.Is(sourceErr, domain.ErrNotFound) {
		return nil, sourceErr
	}
	scoringAvailable := sourceErr == nil

	answers := s.buildAnswers(form, sub, test, scoringAvailable)

	total := form.QuestionCount
	if total == 0 {
		total = len(answers)
	}

	res, err := s.attempts.SaveSubmission(ctx, domain.SaveSubmissionInput{
		Form:           form,
		Sub:            sub,
		BranchID:       branchID,
		TotalQuestions: total,
		Answers:        answers,
	})
	if err != nil {
		return nil, err
	}

	out := &SubmissionResult{
		AttemptID: res.AttemptID,
		InternID:  res.InternID,
		Duplicate: res.Duplicate,
		Scored:    scoringAvailable,
		Score:     res.Scored,
		Total:     res.Total,
	}
	for _, a := range answers {
		if a.Score == nil {
			out.NeedsGrading = true
			break
		}
	}
	if res.Duplicate {
		out.NeedsGrading = res.Scored < res.Total
	}
	return out, nil
}

func (s *Submissions) buildAnswers(
	form *domain.FormSession,
	sub *domain.Submission,
	test *domain.Test,
	scoringAvailable bool,
) []domain.Answer {
	given := make(map[int64]domain.SubmittedAnswer, len(sub.Answers))
	for _, a := range sub.Answers {
		qid := a.QuestionID
		if qid == 0 {
			qid = form.ItemMap[a.ItemID]
		}
		if qid == 0 {
			continue
		}
		given[qid] = domain.SubmittedAnswer{ItemID: a.ItemID, QuestionID: qid, Answer: a.Answer}
	}

	if scoringAvailable && test != nil {
		out := make([]domain.Answer, 0, len(test.Questions))
		for _, q := range test.Questions {
			a := given[q.ID]
			answer := domain.Answer{
				QuestionID:   q.ID,
				QuestionType: q.Type,
				ItemID:       a.ItemID,
				AnswerText:   strings.TrimSpace(a.Answer),
			}
			if sc := ScoreAnswer(q, a.Answer); sc != nil {
				answer.Score = sc
				auto := domain.GradedAuto
				answer.GradedBy = &auto
			}
			out = append(out, answer)
		}
		return out
	}

	ids := make([]int64, 0, len(form.ItemMap))
	items := make(map[int64]string, len(form.ItemMap))
	for itemID, questionID := range form.ItemMap {
		ids = append(ids, questionID)
		items[questionID] = itemID
	}
	if len(ids) == 0 {
		for qid := range given {
			ids = append(ids, qid)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	out := make([]domain.Answer, 0, len(ids))
	for _, qid := range ids {
		a := given[qid]
		itemID := a.ItemID
		if itemID == "" {
			itemID = items[qid]
		}
		out = append(out, domain.Answer{
			QuestionID:   qid,
			QuestionType: 0,
			ItemID:       itemID,
			AnswerText:   strings.TrimSpace(a.Answer),
		})
	}
	return out
}

func (s *Submissions) resolveBranch(ctx context.Context, sub *domain.Submission) (*int64, error) {
	code := strings.TrimSpace(sub.BranchCode)
	name := strings.TrimSpace(sub.BranchName)
	if code == "" && name == "" {
		return nil, nil
	}
	if code == "" {
		code = name
	}

	branch, err := s.branches.GetByCode(ctx, code)
	if err == nil {
		return &branch.ID, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	if name == "" {
		name = code
	}
	branch, err = s.branches.Create(ctx, code, name)
	if err != nil {
		return nil, err
	}
	return &branch.ID, nil
}

func validateSubmission(sub *domain.Submission) error {
	if sub == nil {
		return fmt.Errorf("%w: пустое тело запроса", domain.ErrValidation)
	}
	if strings.TrimSpace(sub.FormID) == "" {
		return fmt.Errorf("%w: form_id обязателен", domain.ErrValidation)
	}
	if strings.TrimSpace(sub.ResponseID) == "" {
		return fmt.Errorf("%w: response_id обязателен", domain.ErrValidation)
	}
	if strings.TrimSpace(sub.Phone) == "" {
		return fmt.Errorf("%w: телефон обязателен: по нему опознаётся стажёр", domain.ErrValidation)
	}
	sub.Phone = strings.TrimSpace(sub.Phone)
	sub.LastName = strings.TrimSpace(sub.LastName)
	sub.FirstName = strings.TrimSpace(sub.FirstName)
	sub.MiddleName = strings.TrimSpace(sub.MiddleName)
	return nil
}

package usecase

import (
	"strings"

	"gform/internal/domain"
)

func ScoreAnswer(q domain.Question, answer string) *int {
	if !domain.IsAutoGraded(q.Type) {
		return nil
	}

	given := normalizeAnswer(answer)
	for _, correct := range q.CorrectOptions() {
		if given != "" && given == normalizeAnswer(correct) {
			return score(1)
		}
	}
	return score(0)
}

func normalizeAnswer(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func score(v int) *int { return &v }

func validScore(s *int) bool {
	return s == nil || *s == 0 || *s == 1
}

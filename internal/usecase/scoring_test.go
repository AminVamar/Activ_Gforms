package usecase

import (
	"testing"

	"gform/internal/domain"
)

func choiceQuestion() domain.Question {
	return domain.Question{
		ID:      1,
		Content: "Столица Таджикистана?",
		Type:    domain.QuestionChoice,
		Options: []domain.Option{
			{ID: 1, Text: "Душанбе", IsCorrect: true},
			{ID: 2, Text: "Худжанд", IsCorrect: false},
		},
	}
}

func TestScoreAnswer(t *testing.T) {
	cases := []struct {
		name     string
		question domain.Question
		answer   string
		want     *int
	}{
		{"верный вариант", choiceQuestion(), "Душанбе", score(1)},
		{"регистр не важен", choiceQuestion(), "дУшАнБе", score(1)},
		{"крайние пробелы не важны", choiceQuestion(), "  Душанбе  ", score(1)},
		{"неверный вариант", choiceQuestion(), "Худжанд", score(0)},
		{"пропущенный вопрос", choiceQuestion(), "", score(0)},
		{"открытый вопрос ждёт администратора",
			domain.Question{ID: 2, Type: domain.QuestionOpen}, "любой текст", nil},
		{"шкала ждёт администратора",
			domain.Question{ID: 3, Type: domain.QuestionScale}, "7", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ScoreAnswer(c.question, c.answer)
			if !sameScore(got, c.want) {
				t.Fatalf("ScoreAnswer = %v, ожидалось %v", str(got), str(c.want))
			}
		})
	}
}

func TestScoreAnswerEmptyOptionDoesNotMatchEmptyAnswer(t *testing.T) {
	q := domain.Question{
		ID:      4,
		Type:    domain.QuestionChoice,
		Options: []domain.Option{{Text: "   ", IsCorrect: true}},
	}
	if got := ScoreAnswer(q, ""); got == nil || *got != 0 {
		t.Fatalf("пустой ответ должен давать 0, получено %v", str(got))
	}
}

func TestValidScore(t *testing.T) {
	if !validScore(nil) || !validScore(score(0)) || !validScore(score(1)) {
		t.Fatal("0, 1 и null должны быть допустимы")
	}
	if validScore(score(2)) || validScore(score(-1)) {
		t.Fatal("любое значение кроме 0, 1 и null должно отвергаться")
	}
}

func str(v *int) string {
	if v == nil {
		return "null"
	}
	return string(rune('0' + *v))
}

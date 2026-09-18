package usecase

import (
	"context"

	"gform/internal/domain"
)

type Interns struct {
	interns  domain.InternRepository
	attempts domain.AttemptRepository
}

func NewInterns(interns domain.InternRepository, attempts domain.AttemptRepository) *Interns {
	return &Interns{interns: interns, attempts: attempts}
}

type InternCard struct {
	Intern   *domain.Intern   `json:"intern"`
	Attempts []domain.Attempt `json:"attempts"`
	Total    int              `json:"attempts_total"`
}

func (i *Interns) List(ctx context.Context, f domain.InternFilter) ([]domain.InternListItem, int, error) {
	return i.interns.List(ctx, f)
}

func (i *Interns) Card(ctx context.Context, id int64, f domain.AttemptFilter) (*InternCard, error) {
	intern, err := i.interns.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	f.InternID = &id
	attempts, total, err := i.attempts.List(ctx, f)
	if err != nil {
		return nil, err
	}
	return &InternCard{Intern: intern, Attempts: attempts, Total: total}, nil
}

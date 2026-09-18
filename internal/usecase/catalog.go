package usecase

import (
	"context"

	"gform/internal/domain"
)

type Catalog struct {
	source domain.TestSource
}

func NewCatalog(source domain.TestSource) *Catalog { return &Catalog{source: source} }

func (c *Catalog) List(ctx context.Context) ([]domain.Test, error) {
	return c.source.List(ctx)
}

func (c *Catalog) Get(ctx context.Context, id int64) (*domain.Test, error) {
	return c.source.Get(ctx, id)
}

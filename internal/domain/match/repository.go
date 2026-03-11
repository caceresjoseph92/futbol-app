package match

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository es el puerto de persistencia para partidos.
type Repository interface {
	Save(ctx context.Context, match *Match) error
	FindByID(ctx context.Context, id uuid.UUID) (*Match, error)
	FindAll(ctx context.Context) ([]*Match, error)
	FindByDateRange(ctx context.Context, from, to time.Time) ([]*Match, error)
	Update(ctx context.Context, match *Match) error
	Delete(ctx context.Context, id uuid.UUID) error
}

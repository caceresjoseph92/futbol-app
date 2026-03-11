// Package stats contiene el caso de uso para obtener estadísticas del grupo.
package stats

import (
	"context"

	"futbol-app/internal/domain/stats"
)

// Repository define qué necesita el servicio de la capa de datos.
type Repository interface {
	GetSummary(ctx context.Context) (*stats.Summary, error)
}

// Service orquesta el caso de uso de estadísticas.
type Service struct {
	repo Repository
}

// NewService crea el servicio de estadísticas.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetSummary retorna todas las estadísticas del grupo.
func (s *Service) GetSummary(ctx context.Context) (*stats.Summary, error) {
	return s.repo.GetSummary(ctx)
}

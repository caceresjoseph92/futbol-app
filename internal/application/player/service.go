// Package player (application) contiene los casos de uso relacionados con jugadores.
// Esta capa orquesta el dominio y los puertos — no contiene lógica de negocio propia.
package player

import (
	"context"

	"futbol-app/internal/domain/player"

	"github.com/google/uuid"
)

// Service implementa los casos de uso de jugadores.
//
// Principio D de SOLID: depende de la interfaz player.Repository,
// no de una implementación concreta.
type Service struct {
	repo player.Repository
}

// NewService crea el servicio con inyección de dependencia del repositorio.
func NewService(repo player.Repository) *Service {
	return &Service{repo: repo}
}

// --- DTOs (Data Transfer Objects) ---

// CreatePlayerInput contiene los datos para crear un jugador.
type CreatePlayerInput struct {
	Name             string
	PrimaryPosition  player.Position
	CanPlayPositions []player.Position
	Notes            string
	Rating           int8
}

// UpdatePlayerInput contiene los datos actualizables de un jugador.
type UpdatePlayerInput struct {
	Name             string
	PrimaryPosition  player.Position
	CanPlayPositions []player.Position
	Notes            string
	Rating           int8
}

// --- Casos de uso ---

// CreatePlayer crea un nuevo jugador y lo persiste.
func (s *Service) CreatePlayer(ctx context.Context, input CreatePlayerInput) (*player.Player, error) {
	p, err := player.New(input.Name, input.PrimaryPosition, input.Rating)
	if err != nil {
		return nil, err
	}
	p.CanPlayPositions = input.CanPlayPositions
	p.Notes = input.Notes

	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetPlayer obtiene un jugador por ID.
func (s *Service) GetPlayer(ctx context.Context, id uuid.UUID) (*player.Player, error) {
	return s.repo.FindByID(ctx, id)
}

// ListPlayers retorna todos los jugadores activos.
func (s *Service) ListPlayers(ctx context.Context) ([]*player.Player, error) {
	return s.repo.FindAll(ctx)
}

// ListAllPlayers retorna todos los jugadores incluyendo inactivos (solo admin).
func (s *Service) ListAllPlayers(ctx context.Context) ([]*player.Player, error) {
	return s.repo.FindAllIncludingInactive(ctx)
}

// UpdatePlayer actualiza los datos de un jugador existente.
func (s *Service) UpdatePlayer(ctx context.Context, id uuid.UUID, input UpdatePlayerInput) (*player.Player, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validar rating vía dominio
	if err := p.UpdateRating(input.Rating); err != nil {
		return nil, err
	}

	p.Name = input.Name
	p.PrimaryPosition = input.PrimaryPosition
	p.CanPlayPositions = input.CanPlayPositions
	p.Notes = input.Notes

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateRating actualiza solo la calificación de un jugador.
func (s *Service) UpdateRating(ctx context.Context, id uuid.UUID, rating int8) (*player.Player, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := p.UpdateRating(rating); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// DeactivatePlayer desactiva un jugador (soft delete).
func (s *Service) DeactivatePlayer(ctx context.Context, id uuid.UUID) error {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	p.Deactivate()
	return s.repo.Update(ctx, p)
}

// ActivatePlayer reactiva un jugador.
func (s *Service) ActivatePlayer(ctx context.Context, id uuid.UUID) error {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	p.Activate()
	return s.repo.Update(ctx, p)
}

// Package match (application) contiene los casos de uso del partido.
package match

import (
	"context"
	"fmt"
	"time"

	"futbol-app/internal/domain/match"
	"futbol-app/internal/domain/player"

	"github.com/google/uuid"
)

// Service implementa los casos de uso del partido.
type Service struct {
	matchRepo  match.Repository
	playerRepo player.Repository
}

// NewService crea el servicio con inyección de dependencias.
func NewService(matchRepo match.Repository, playerRepo player.Repository) *Service {
	return &Service{
		matchRepo:  matchRepo,
		playerRepo: playerRepo,
	}
}

// CreateMatchInput datos para crear un partido.
type CreateMatchInput struct {
	PlayedAt       time.Time
	GoalkeeperInfo string
	CreatedBy      uuid.UUID
}

// CreateMatch crea un partido en estado draft.
func (s *Service) CreateMatch(ctx context.Context, input CreateMatchInput) (*match.Match, error) {
	m := match.New(input.PlayedAt, input.GoalkeeperInfo, input.CreatedBy)
	if err := s.matchRepo.Save(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetMatch obtiene un partido por ID.
func (s *Service) GetMatch(ctx context.Context, id uuid.UUID) (*match.Match, error) {
	return s.matchRepo.FindByID(ctx, id)
}

// ListMatches retorna todos los partidos (historial).
func (s *Service) ListMatches(ctx context.Context) ([]*match.Match, error) {
	return s.matchRepo.FindAll(ctx)
}

// AddPlayersToMatch agrega los 12 jugadores convocados al partido.
// Recibe los IDs de los jugadores, busca sus datos actuales y guarda el snapshot de rating.
func (s *Service) AddPlayersToMatch(ctx context.Context, matchID uuid.UUID, playerIDs []uuid.UUID) (*match.Match, error) {
	if len(playerIDs) != match.RequiredPlayers {
		return nil, match.ErrInvalidPlayerCount
	}

	m, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	// Limpiar jugadores previos y agregar los nuevos
	m.Players = nil
	for _, pid := range playerIDs {
		p, err := s.playerRepo.FindByID(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("jugador %s no encontrado: %w", pid, err)
		}
		if err := m.AddPlayer(p); err != nil {
			return nil, err
		}
	}

	if err := s.matchRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// GenerateTeams ejecuta el algoritmo de balanceo automático.
func (s *Service) GenerateTeams(ctx context.Context, matchID uuid.UUID) (*match.Match, error) {
	m, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if err := m.AssignTeams(); err != nil {
		return nil, err
	}
	if err := s.matchRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// TransferPlayer mueve un jugador al equipo contrario (ajuste manual).
func (s *Service) TransferPlayer(ctx context.Context, matchID, playerID uuid.UUID) (*match.Match, error) {
	m, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if err := m.TransferPlayer(playerID); err != nil {
		return nil, err
	}
	if err := s.matchRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// SwapPlayers intercambia dos jugadores entre equipos.
func (s *Service) SwapPlayers(ctx context.Context, matchID, playerID1, playerID2 uuid.UUID) (*match.Match, error) {
	m, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if err := m.SwapPlayers(playerID1, playerID2); err != nil {
		return nil, err
	}
	if err := s.matchRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// PublishMatch publica los equipos — visible para todos.
func (s *Service) PublishMatch(ctx context.Context, matchID uuid.UUID) (*match.Match, error) {
	m, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if err := m.Publish(); err != nil {
		return nil, err
	}
	if err := s.matchRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// FinishMatch registra el resultado del partido.
func (s *Service) FinishMatch(ctx context.Context, matchID uuid.UUID, score1, score2 int) (*match.Match, error) {
	m, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if err := m.Finish(score1, score2); err != nil {
		return nil, err
	}
	if err := s.matchRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

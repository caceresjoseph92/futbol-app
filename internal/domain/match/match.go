// Package match define la entidad Match y la lógica de negocio del partido,
// incluyendo el algoritmo de balanceo de equipos.
package match

import (
	"errors"
	"sort"
	"time"

	"futbol-app/internal/domain/player"

	"github.com/google/uuid"
)

// Status representa el estado del partido en su ciclo de vida.
type Status string

const (
	StatusDraft     Status = "draft"     // armando equipos
	StatusPublished Status = "published" // equipos publicados, visible para todos
	StatusFinished  Status = "finished"  // partido terminado, con resultado
)

var (
	ErrMatchNotFound         = errors.New("partido no encontrado")
	ErrInvalidPlayerCount    = errors.New("se requieren exactamente 12 jugadores de campo")
	ErrMatchAlreadyPublished = errors.New("el partido ya fue publicado")
	ErrMatchNotFinished      = errors.New("el partido no ha terminado")
	ErrInvalidScore          = errors.New("el marcador no puede ser negativo")
	ErrPlayerAlreadyInMatch  = errors.New("el jugador ya está en este partido")
	ErrPlayerNotInMatch      = errors.New("el jugador no está en este partido")
)

const RequiredPlayers = 12

// MatchPlayer representa un jugador dentro de un partido,
// con su asignación de equipo y el snapshot de su rating en ese momento.
type MatchPlayer struct {
	PlayerID       uuid.UUID
	PlayerName     string
	PrimaryPosition player.Position
	Notes          string
	Team           int8 // 1 o 2
	RatingSnapshot int8 // rating del jugador al momento del partido
}

// Match es la entidad central del partido.
type Match struct {
	ID             uuid.UUID
	PlayedAt       time.Time
	Status         Status
	Players        []MatchPlayer
	Team1Score     *int
	Team2Score     *int
	GoalkeeperInfo string // texto libre: "Futguardian 1 y Futguardian 2"
	CreatedBy      uuid.UUID
	CreatedAt      time.Time
}

// New crea un nuevo Match en estado draft.
func New(playedAt time.Time, goalkeeperInfo string, createdBy uuid.UUID) *Match {
	return &Match{
		ID:             uuid.New(),
		PlayedAt:       playedAt,
		Status:         StatusDraft,
		GoalkeeperInfo: goalkeeperInfo,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
	}
}

// AddPlayer agrega un jugador convocado al partido (sin equipo asignado aún).
func (m *Match) AddPlayer(p *player.Player) error {
	for _, mp := range m.Players {
		if mp.PlayerID == p.ID {
			return ErrPlayerAlreadyInMatch
		}
	}
	m.Players = append(m.Players, MatchPlayer{
		PlayerID:        p.ID,
		PlayerName:      p.Name,
		PrimaryPosition: p.PrimaryPosition,
		Notes:           p.Notes,
		RatingSnapshot:  p.Rating,
		Team:            0, // sin asignar
	})
	return nil
}

// AssignTeams ejecuta el algoritmo de balanceo y asigna los jugadores a equipos.
// Requiere exactamente 12 jugadores previamente agregados.
//
// Algoritmo: greedy balancing por rating.
// Ordena jugadores de mayor a menor rating y los distribuye alternando equipos,
// asegurando que la diferencia de puntuación total sea mínima.
func (m *Match) AssignTeams() error {
	if len(m.Players) != RequiredPlayers {
		return ErrInvalidPlayerCount
	}

	// Ordenar de mayor a menor rating (snapshot)
	sorted := make([]MatchPlayer, len(m.Players))
	copy(sorted, m.Players)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RatingSnapshot > sorted[j].RatingSnapshot
	})

	// Greedy: asignar al equipo con menor suma acumulada
	var sum1, sum2 int8
	for i := range sorted {
		if sum1 <= sum2 {
			sorted[i].Team = 1
			sum1 += sorted[i].RatingSnapshot
		} else {
			sorted[i].Team = 2
			sum2 += sorted[i].RatingSnapshot
		}
	}

	m.Players = sorted
	return nil
}

// TransferPlayer mueve un jugador del equipo actual al otro equipo.
// Permite ajuste manual después del balanceo automático.
func (m *Match) TransferPlayer(playerID uuid.UUID) error {
	for i, mp := range m.Players {
		if mp.PlayerID == playerID {
			if m.Players[i].Team == 1 {
				m.Players[i].Team = 2
			} else {
				m.Players[i].Team = 1
			}
			return nil
		}
	}
	return ErrPlayerNotInMatch
}

// SwapPlayers intercambia dos jugadores entre equipos.
func (m *Match) SwapPlayers(playerID1, playerID2 uuid.UUID) error {
	idx1, idx2 := -1, -1
	for i, mp := range m.Players {
		if mp.PlayerID == playerID1 {
			idx1 = i
		}
		if mp.PlayerID == playerID2 {
			idx2 = i
		}
	}
	if idx1 == -1 || idx2 == -1 {
		return ErrPlayerNotInMatch
	}
	m.Players[idx1].Team, m.Players[idx2].Team = m.Players[idx2].Team, m.Players[idx1].Team
	return nil
}

// Publish cambia el estado a publicado.
// Solo se puede publicar si los equipos están asignados (todos con team 1 o 2).
func (m *Match) Publish() error {
	if m.Status == StatusPublished || m.Status == StatusFinished {
		return ErrMatchAlreadyPublished
	}
	for _, mp := range m.Players {
		if mp.Team == 0 {
			return errors.New("todos los jugadores deben tener equipo asignado antes de publicar")
		}
	}
	m.Status = StatusPublished
	return nil
}

// Finish registra el resultado y cierra el partido.
func (m *Match) Finish(score1, score2 int) error {
	if score1 < 0 || score2 < 0 {
		return ErrInvalidScore
	}
	m.Team1Score = &score1
	m.Team2Score = &score2
	m.Status = StatusFinished
	return nil
}

// Team1 retorna los jugadores del equipo 1.
func (m *Match) Team1() []MatchPlayer {
	return m.teamPlayers(1)
}

// Team2 retorna los jugadores del equipo 2.
func (m *Match) Team2() []MatchPlayer {
	return m.teamPlayers(2)
}

// Team1Rating retorna la suma de ratings del equipo 1.
func (m *Match) Team1Rating() int {
	return m.teamRating(1)
}

// Team2Rating retorna la suma de ratings del equipo 2.
func (m *Match) Team2Rating() int {
	return m.teamRating(2)
}

func (m *Match) teamPlayers(team int8) []MatchPlayer {
	var result []MatchPlayer
	for _, mp := range m.Players {
		if mp.Team == team {
			result = append(result, mp)
		}
	}
	return result
}

func (m *Match) teamRating(team int8) int {
	total := 0
	for _, mp := range m.Players {
		if mp.Team == team {
			total += int(mp.RatingSnapshot)
		}
	}
	return total
}

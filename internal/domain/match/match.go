// Package match define la entidad Match y la lógica de negocio del partido,
// incluyendo el algoritmo de balanceo de equipos.
package match

import (
	"errors"
	"math/rand"
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
	PlayerID              uuid.UUID
	PlayerName            string
	PrimaryPosition       player.Position
	Notes                 string
	Team                  int8    // 1 o 2
	RatingSnapshot        int8    // rating del jugador al momento del partido
	WinPctSnapshot        float64 // % victorias históricas (0-100); 0 si sin partidos
	MatchesPlayedSnapshot int     // partidos jugados al momento; 0 si sin historial
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

// SetPlayerWinPct actualiza el snapshot de win rate y partidos jugados de un jugador en el partido.
// Se llama antes de AssignTeams para enriquecer el balanceo con datos históricos.
func (m *Match) SetPlayerWinPct(playerID uuid.UUID, winPct float64, matchesPlayed int) {
	for i := range m.Players {
		if m.Players[i].PlayerID == playerID {
			m.Players[i].WinPctSnapshot = winPct
			m.Players[i].MatchesPlayedSnapshot = matchesPlayed
			return
		}
	}
}

// compositeScore calcula el score de balanceo combinado para un jugador.
// Si tiene al menos 5 partidos: 50% rating manual + 50% win rate normalizado (0-10).
// Si tiene menos de 5 partidos: usa rating manual puro (sin historial suficiente).
func compositeScore(mp MatchPlayer) float64 {
	const minMatches = 5
	rating := float64(mp.RatingSnapshot)
	if mp.MatchesPlayedSnapshot < minMatches {
		return rating
	}
	winRateNorm := mp.WinPctSnapshot / 10.0 // normaliza 0-100 → 0-10
	return rating*0.5 + winRateNorm*0.5
}

// AssignTeams ejecuta el algoritmo de balanceo automático y asigna los jugadores a equipos.
// Requiere exactamente 12 jugadores previamente agregados.
//
// Formación objetivo por equipo:
//   - Estándar (≥2 creadores disponibles): 3 defensas + 1 creador + 2 delanteros
//   - Alternativa (<2 creadores):          3 defensas + 3 delanteros
//
// Los jugadores se agrupan por posición y se procesan en bloques
// (defensas → creadores → delanteros). Dentro de cada bloque se usa greedy
// de suma acumulada para repartir jugadores de rating similar entre equipos,
// garantizando balance posicional y de nivel simultáneamente.
func (m *Match) AssignTeams() error {
	if len(m.Players) != RequiredPlayers {
		return ErrInvalidPlayerCount
	}

	byRating := func(p []MatchPlayer) {
		rand.Shuffle(len(p), func(i, j int) { p[i], p[j] = p[j], p[i] })
		sort.SliceStable(p, func(i, j int) bool { return compositeScore(p[i]) > compositeScore(p[j]) })
	}

	// Separar por posición primaria
	var defensas, creadores, delanteros []MatchPlayer
	for _, mp := range m.Players {
		switch mp.PrimaryPosition {
		case player.PositionDefensa:
			defensas = append(defensas, mp)
		case player.PositionCreador:
			creadores = append(creadores, mp)
		default:
			delanteros = append(delanteros, mp)
		}
	}
	byRating(defensas)
	byRating(creadores)
	byRating(delanteros)

	// Si hay menos de 2 creadores, los tratamos como delanteros
	if len(creadores) < 2 {
		delanteros = append(delanteros, creadores...)
		byRating(delanteros)
		creadores = nil
	}

	// Cupos: 6 defensas | 0-2 creadores | resto delanteros
	const defTarget = 6
	creTarget := 2
	if creadores == nil {
		creTarget = 0
	}

	var overflow []MatchPlayer

	// Llenar cupo de defensas (excedente va a pool delantero)
	var defSlots []MatchPlayer
	if len(defensas) >= defTarget {
		defSlots = defensas[:defTarget]
		overflow = append(overflow, defensas[defTarget:]...)
	} else {
		defSlots = defensas
	}

	// Llenar cupo de creadores (excedente va a pool delantero)
	var creSlots []MatchPlayer
	if len(creadores) >= creTarget {
		creSlots = creadores[:creTarget]
		overflow = append(overflow, creadores[creTarget:]...)
	} else {
		creSlots = creadores
	}

	// El resto del cupo lo cubren delanteros + overflow de otras posiciones
	allFwd := append(delanteros, overflow...)
	byRating(allFwd)
	fwdTarget := 12 - len(defSlots) - len(creSlots)
	var fwdSlots []MatchPlayer
	if len(allFwd) >= fwdTarget {
		fwdSlots = allFwd[:fwdTarget]
	} else {
		fwdSlots = allFwd
	}

	// Lista final: defensas → creadores → delanteros
	ordered := make([]MatchPlayer, 0, 12)
	ordered = append(ordered, defSlots...)
	ordered = append(ordered, creSlots...)
	ordered = append(ordered, fwdSlots...)

	// Fallback por si el conteo no cierra exactamente
	if len(ordered) != 12 {
		ordered = make([]MatchPlayer, len(m.Players))
		copy(ordered, m.Players)
		byRating(ordered)
	}

	// Greedy de suma acumulada procesando bloque a bloque:
	// al iterar defensas juntos, luego creadores, luego delanteros,
	// ~la mitad de cada posición cae naturalmente en cada equipo.
	var sum1, sum2 float64
	for i := range ordered {
		if sum1 <= sum2 {
			ordered[i].Team = 1
			sum1 += compositeScore(ordered[i])
		} else {
			ordered[i].Team = 2
			sum2 += compositeScore(ordered[i])
		}
	}

	m.Players = ordered
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

// CorrectScore corrige el marcador de un partido ya terminado.
// A diferencia de Finish, no requiere que el partido esté publicado.
func (m *Match) CorrectScore(score1, score2 int) error {
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

// Package stats define los tipos de datos para las estadísticas del grupo.
package stats

import (
	"time"

	"github.com/google/uuid"
)

// PlayerStat reúne todas las métricas de un jugador.
type PlayerStat struct {
	PlayerID      uuid.UUID
	PlayerName    string
	MatchesPlayed int
	Wins          int
	Losses        int
	Draws         int
	WinPct        float64 // porcentaje de victorias (0-100)
	Streak        int     // >0 racha ganadora, <0 racha perdedora, 0 = neutral
	StreakLabel   string  // "3W", "2L", "—"
}

// WinningPair representa dos jugadores y sus victorias compartidas.
type WinningPair struct {
	Player1ID      uuid.UUID
	Player1Name    string
	Player2ID      uuid.UUID
	Player2Name    string
	WinsTogether   int
	PlayedTogether int
	WinPct         float64
}

// Summary agrupa todas las estadísticas para la vista.
type Summary struct {
	TopAttendance []PlayerStat  // ordenado por partidos jugados desc
	TopWinners    []PlayerStat  // ordenado por victorias desc
	Streaks       []PlayerStat  // jugadores con racha activa (>= 2)
	WinningPairs  []WinningPair // top parejas ganadoras
}

// PlayerMatchRecord es el registro de un partido individual para un jugador.
type PlayerMatchRecord struct {
	MatchID   uuid.UUID
	PlayedAt  time.Time
	Result    string // "win", "loss", "draw"
	Score1    int
	Score2    int
	Team      int8
	Teammates []string // nombres de compañeros de equipo
	Opponents []string // nombres del equipo contrario
}

// Badge representa un logro que un jugador puede ganar.
type Badge struct {
	Icon        string // emoji
	Name        string
	Description string
}

// PlayerHistory agrupa el historial completo de un jugador.
type PlayerHistory struct {
	PlayerID   uuid.UUID
	PlayerName string
	Matches    []PlayerMatchRecord
	Wins       int
	Losses     int
	Draws      int
	WinPct     float64
	Badges     []Badge
}

// ComputeBadges calcula los logros de un jugador a partir de su historial.
// Es una función pura — no accede a la DB, solo lee los datos ya cargados.
//
// Reglas:
//   - 🔥 En racha:        3+ victorias consecutivas actuales
//   - 🧊 Racha fría:      3+ derrotas consecutivas actuales
//   - ⚽ Primeros pasos:  1–4 partidos jugados
//   - 🎯 Fiel al grupo:   15+ partidos jugados
//   - 👑 Leyenda:         30+ partidos jugados
//   - 🏆 Ganador nato:    WinPct >= 60% con mínimo 10 partidos
//   - 🧱 Inquebrantable:  20+ partidos, derrotas <= 25%
//   - 💎 Consistente:     10+ partidos, WinPct entre 45–60% (equilibrado)
func ComputeBadges(h *PlayerHistory) []Badge {
	var badges []Badge
	total := h.Wins + h.Losses + h.Draws

	// Racha actual (los Matches vienen ordenados de más reciente a más antiguo)
	streak := currentStreak(h.Matches)

	if streak >= 3 {
		badges = append(badges, Badge{"🔥", "En racha", "3 o más victorias seguidas"})
	}
	if streak <= -3 {
		badges = append(badges, Badge{"🧊", "Racha fría", "3 o más derrotas seguidas"})
	}
	if total >= 1 && total < 5 {
		badges = append(badges, Badge{"⚽", "Primeros pasos", "Menos de 5 partidos jugados"})
	}
	if total >= 15 {
		badges = append(badges, Badge{"🎯", "Fiel al grupo", "15 o más partidos jugados"})
	}
	if total >= 30 {
		badges = append(badges, Badge{"👑", "Leyenda", "30 o más partidos jugados"})
	}
	if total >= 10 && h.WinPct >= 60 {
		badges = append(badges, Badge{"🏆", "Ganador nato", "60%+ victorias con al menos 10 partidos"})
	}
	if total >= 20 && h.Losses <= total/4 {
		badges = append(badges, Badge{"🧱", "Inquebrantable", "20+ partidos con menos del 25% de derrotas"})
	}
	if total >= 10 && h.WinPct >= 45 && h.WinPct < 60 {
		badges = append(badges, Badge{"💎", "Consistente", "Siempre competitivo, cerca del 50%"})
	}
	return badges
}

// currentStreak calcula la racha actual desde el partido más reciente.
// +N = N victorias seguidas, -N = N derrotas seguidas, 0 = empate o sin partidos.
func currentStreak(matches []PlayerMatchRecord) int {
	if len(matches) == 0 {
		return 0
	}
	first := matches[0].Result
	if first == "draw" {
		return 0
	}
	count := 0
	for _, m := range matches {
		if m.Result == first {
			count++
		} else {
			break
		}
	}
	if first == "win" {
		return count
	}
	return -count
}

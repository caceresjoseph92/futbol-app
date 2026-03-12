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

// PlayerHistory agrupa el historial completo de un jugador.
type PlayerHistory struct {
	PlayerID   uuid.UUID
	PlayerName string
	Matches    []PlayerMatchRecord
	Wins       int
	Losses     int
	Draws      int
	WinPct     float64
}

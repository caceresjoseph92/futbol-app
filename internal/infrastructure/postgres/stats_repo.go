package postgres

import (
	"context"
	"fmt"
	"math"
	"sort"

	"futbol-app/internal/domain/stats"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StatsRepository calcula estadísticas del grupo a partir de partidos terminados.
type StatsRepository struct {
	pool *pgxpool.Pool
}

// NewStatsRepository crea el repositorio de estadísticas.
func NewStatsRepository(pool *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{pool: pool}
}

// GetSummary devuelve todas las estadísticas calculadas.
func (r *StatsRepository) GetSummary(ctx context.Context) (*stats.Summary, error) {
	playerStats, err := r.getPlayerStats(ctx)
	if err != nil {
		return nil, err
	}

	pairs, err := r.getWinningPairs(ctx)
	if err != nil {
		return nil, err
	}

	// TopAttendance: ordenado por partidos jugados
	attendance := make([]stats.PlayerStat, len(playerStats))
	copy(attendance, playerStats)
	sort.Slice(attendance, func(i, j int) bool {
		return attendance[i].MatchesPlayed > attendance[j].MatchesPlayed
	})

	// TopWinners: ordenado por victorias, luego % victorias
	winners := make([]stats.PlayerStat, len(playerStats))
	copy(winners, playerStats)
	sort.Slice(winners, func(i, j int) bool {
		if winners[i].Wins != winners[j].Wins {
			return winners[i].Wins > winners[j].Wins
		}
		return winners[i].WinPct > winners[j].WinPct
	})

	// Streaks: jugadores con cualquier racha activa (!= 0)
	var streaks []stats.PlayerStat
	for _, ps := range playerStats {
		if ps.Streak != 0 {
			streaks = append(streaks, ps)
		}
	}
	sort.Slice(streaks, func(i, j int) bool {
		return math.Abs(float64(streaks[i].Streak)) > math.Abs(float64(streaks[j].Streak))
	})

	return &stats.Summary{
		TopAttendance: attendance,
		TopWinners:    winners,
		Streaks:       streaks,
		WinningPairs:  pairs,
	}, nil
}

func (r *StatsRepository) getPlayerStats(ctx context.Context) ([]stats.PlayerStat, error) {
	// Paso 1: todos los jugadores activos (para que aparezcan incluso con 0 partidos)
	type playerData struct {
		name    string
		results []string // ordenados de más reciente a más antiguo
	}
	byPlayer := map[uuid.UUID]*playerData{}
	var order []uuid.UUID

	pRows, err := r.pool.Query(ctx, `SELECT id, name FROM players WHERE active = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer pRows.Close()
	for pRows.Next() {
		var pid uuid.UUID
		var name string
		if err := pRows.Scan(&pid, &name); err != nil {
			return nil, err
		}
		byPlayer[pid] = &playerData{name: name}
		order = append(order, pid)
	}
	if err := pRows.Err(); err != nil {
		return nil, err
	}

	// Paso 2: resultados de partidos terminados
	rows, err := r.pool.Query(ctx, `
		SELECT
			mp.player_id,
			CASE
				WHEN m.team1_score = m.team2_score THEN 'draw'
				WHEN (mp.team = 1 AND m.team1_score > m.team2_score)
				  OR (mp.team = 2 AND m.team2_score > m.team1_score) THEN 'win'
				ELSE 'loss'
			END AS result
		FROM match_players mp
		JOIN matches m ON m.id = mp.match_id
		WHERE m.status = 'finished'
		  AND m.team1_score IS NOT NULL
		  AND m.team2_score IS NOT NULL
		ORDER BY mp.player_id, m.played_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pid uuid.UUID
		var result string
		if err := rows.Scan(&pid, &result); err != nil {
			return nil, err
		}
		if pd, ok := byPlayer[pid]; ok {
			pd.results = append(pd.results, result)
		}
	}

	var out []stats.PlayerStat
	for _, pid := range order {
		d := byPlayer[pid]
		wins, losses, draws := 0, 0, 0
		for _, res := range d.results {
			switch res {
			case "win":
				wins++
			case "loss":
				losses++
			case "draw":
				draws++
			}
		}
		total := wins + losses + draws
		winPct := 0.0
		if total > 0 {
			winPct = math.Round(float64(wins)/float64(total)*1000) / 10
		}

		// Calcular racha desde el partido más reciente
		streak := computeStreak(d.results)
		streakLabel := formatStreak(streak)

		out = append(out, stats.PlayerStat{
			PlayerID:      pid,
			PlayerName:    d.name,
			MatchesPlayed: total,
			Wins:          wins,
			Losses:        losses,
			Draws:         draws,
			WinPct:        winPct,
			Streak:        streak,
			StreakLabel:   streakLabel,
		})
	}

	return out, nil
}

// computeStreak cuenta la racha actual desde el resultado más reciente.
// +N = N victorias consecutivas, -N = N derrotas consecutivas.
func computeStreak(results []string) int {
	if len(results) == 0 {
		return 0
	}
	first := results[0]
	if first == "draw" {
		return 0
	}
	count := 0
	for _, r := range results {
		if r == first {
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

func formatStreak(streak int) string {
	if streak == 0 {
		return "—"
	}
	if streak > 0 {
		return fmt.Sprintf("%dV", streak)
	}
	return fmt.Sprintf("%dD", -streak)
}

func (r *StatsRepository) getWinningPairs(ctx context.Context) ([]stats.WinningPair, error) {
	// Victorias compartidas
	rows, err := r.pool.Query(ctx, `
		SELECT
			CASE WHEN a.player_id < b.player_id THEN a.player_id ELSE b.player_id END AS p1_id,
			CASE WHEN a.player_id < b.player_id THEN a.player_name ELSE b.player_name END AS p1_name,
			CASE WHEN a.player_id < b.player_id THEN b.player_id ELSE a.player_id END AS p2_id,
			CASE WHEN a.player_id < b.player_id THEN b.player_name ELSE a.player_name END AS p2_name,
			COUNT(*) AS wins_together
		FROM match_players a
		JOIN match_players b ON a.match_id = b.match_id
			AND a.team = b.team
			AND a.player_id < b.player_id
		JOIN matches m ON m.id = a.match_id
		WHERE m.status = 'finished'
		  AND m.team1_score IS NOT NULL
		  AND m.team2_score IS NOT NULL
		  AND ((a.team = 1 AND m.team1_score > m.team2_score)
		    OR (a.team = 2 AND m.team2_score > m.team1_score))
		GROUP BY p1_id, p1_name, p2_id, p2_name
		ORDER BY wins_together DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type pairKey struct{ p1, p2 uuid.UUID }
	winsMap := map[pairKey]*stats.WinningPair{}
	var pairOrder []pairKey

	for rows.Next() {
		var p stats.WinningPair
		if err := rows.Scan(&p.Player1ID, &p.Player1Name, &p.Player2ID, &p.Player2Name, &p.WinsTogether); err != nil {
			return nil, err
		}
		key := pairKey{p.Player1ID, p.Player2ID}
		winsMap[key] = &p
		pairOrder = append(pairOrder, key)
	}

	if len(pairOrder) == 0 {
		return nil, nil
	}

	// Partidos juntos (misma query sin filtrar ganador)
	rows2, err := r.pool.Query(ctx, `
		SELECT
			CASE WHEN a.player_id < b.player_id THEN a.player_id ELSE b.player_id END AS p1_id,
			CASE WHEN a.player_id < b.player_id THEN b.player_id ELSE a.player_id END AS p2_id,
			COUNT(*) AS played_together
		FROM match_players a
		JOIN match_players b ON a.match_id = b.match_id
			AND a.team = b.team
			AND a.player_id < b.player_id
		JOIN matches m ON m.id = a.match_id
		WHERE m.status = 'finished'
		GROUP BY p1_id, p2_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var p1, p2 uuid.UUID
		var played int
		if err := rows2.Scan(&p1, &p2, &played); err != nil {
			return nil, err
		}
		key := pairKey{p1, p2}
		if wp, ok := winsMap[key]; ok {
			wp.PlayedTogether = played
			if played > 0 {
				wp.WinPct = math.Round(float64(wp.WinsTogether)/float64(played)*1000) / 10
			}
		}
	}

	var result []stats.WinningPair
	for _, key := range pairOrder {
		result = append(result, *winsMap[key])
	}
	return result, nil
}

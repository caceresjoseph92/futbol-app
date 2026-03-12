package postgres

import (
	"context"
	"errors"
	"time"

	"futbol-app/internal/domain/match"
	"futbol-app/internal/domain/player"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MatchRepository implementa match.Repository usando PostgreSQL.
type MatchRepository struct {
	pool *pgxpool.Pool
}

// NewMatchRepository crea el repositorio de partidos.
func NewMatchRepository(pool *pgxpool.Pool) *MatchRepository {
	return &MatchRepository{pool: pool}
}

func (r *MatchRepository) Save(ctx context.Context, m *match.Match) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO matches (id, played_at, status, team1_score, team2_score, goalkeeper_info, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, m.ID, m.PlayedAt, string(m.Status), m.Team1Score, m.Team2Score, m.GoalkeeperInfo, m.CreatedBy, m.CreatedAt)
	if err != nil {
		return err
	}

	if err := r.saveMatchPlayers(ctx, tx, m); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *MatchRepository) FindByID(ctx context.Context, id uuid.UUID) (*match.Match, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, played_at, status, team1_score, team2_score, goalkeeper_info, created_by, created_at
		FROM matches WHERE id = $1
	`, id)

	m, err := scanMatch(row)
	if err != nil {
		return nil, err
	}

	players, err := r.findMatchPlayers(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Players = players
	return m, nil
}

func (r *MatchRepository) FindAll(ctx context.Context) ([]*match.Match, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, played_at, status, team1_score, team2_score, goalkeeper_info, created_by, created_at
		FROM matches ORDER BY played_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*match.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		players, err := r.findMatchPlayers(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		m.Players = players
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

func (r *MatchRepository) FindByDateRange(ctx context.Context, from, to time.Time) ([]*match.Match, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, played_at, status, team1_score, team2_score, goalkeeper_info, created_by, created_at
		FROM matches WHERE played_at BETWEEN $1 AND $2 ORDER BY played_at DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*match.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

func (r *MatchRepository) Update(ctx context.Context, m *match.Match) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE matches SET
			played_at = $2, status = $3, team1_score = $4, team2_score = $5, goalkeeper_info = $6
		WHERE id = $1
	`, m.ID, m.PlayedAt, string(m.Status), m.Team1Score, m.Team2Score, m.GoalkeeperInfo)
	if err != nil {
		return err
	}

	// Reemplazar jugadores del partido
	if _, err := tx.Exec(ctx, "DELETE FROM match_players WHERE match_id = $1", m.ID); err != nil {
		return err
	}
	if err := r.saveMatchPlayers(ctx, tx, m); err != nil {
		return err
	}

	return tx.Commit(ctx)
}


// FindPaginated retorna una página de partidos y el total de partidos.
// offset = (page-1)*limit, limit = partidos por página.
func (r *MatchRepository) FindPaginated(ctx context.Context, offset, limit int) ([]*match.Match, int, error) {
	// Total de partidos
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM matches").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, played_at, status, team1_score, team2_score, goalkeeper_info, created_by, created_at
		FROM matches ORDER BY played_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var matches []*match.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, 0, err
		}
		players, err := r.findMatchPlayers(ctx, m.ID)
		if err != nil {
			return nil, 0, err
		}
		m.Players = players
		matches = append(matches, m)
	}
	return matches, total, rows.Err()
}

func (r *MatchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM matches WHERE id = $1", id)
	return err
}

// --- helpers ---

type dbTx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func (r *MatchRepository) saveMatchPlayers(ctx context.Context, tx dbTx, m *match.Match) error {
	for _, mp := range m.Players {
		_, err := tx.Exec(ctx, `
			INSERT INTO match_players (match_id, player_id, player_name, primary_position, notes, team, rating_snapshot)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, m.ID, mp.PlayerID, mp.PlayerName, string(mp.PrimaryPosition), mp.Notes, mp.Team, mp.RatingSnapshot)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *MatchRepository) findMatchPlayers(ctx context.Context, matchID uuid.UUID) ([]match.MatchPlayer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT player_id, player_name, primary_position, notes, team, rating_snapshot
		FROM match_players WHERE match_id = $1 ORDER BY team, player_name
	`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []match.MatchPlayer
	for rows.Next() {
		var mp match.MatchPlayer
		var posStr string
		if err := rows.Scan(&mp.PlayerID, &mp.PlayerName, &posStr, &mp.Notes, &mp.Team, &mp.RatingSnapshot); err != nil {
			return nil, err
		}
		mp.PrimaryPosition = player.Position(posStr)
		players = append(players, mp)
	}
	return players, rows.Err()
}

func scanMatch(row pgx.Row) (*match.Match, error) {
	var m match.Match
	var statusStr string
	err := row.Scan(&m.ID, &m.PlayedAt, &statusStr, &m.Team1Score, &m.Team2Score, &m.GoalkeeperInfo, &m.CreatedBy, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, match.ErrMatchNotFound
		}
		return nil, err
	}
	m.Status = match.Status(statusStr)
	return &m, nil
}

package postgres

import (
	"context"
	"errors"

	"futbol-app/internal/domain/player"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlayerRepository implementa player.Repository usando PostgreSQL.
// Es un ADAPTADOR — traduce entre el dominio y la base de datos.
type PlayerRepository struct {
	pool *pgxpool.Pool
}

// NewPlayerRepository crea el repositorio con el pool de conexiones.
func NewPlayerRepository(pool *pgxpool.Pool) *PlayerRepository {
	return &PlayerRepository{pool: pool}
}

// Save inserta un nuevo jugador en la base de datos.
func (r *PlayerRepository) Save(ctx context.Context, p *player.Player) error {
	query := `
		INSERT INTO players (id, name, primary_position, can_play_positions, notes, rating, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	positions := positionsToStrings(p.CanPlayPositions)
	_, err := r.pool.Exec(ctx, query,
		p.ID, p.Name, string(p.PrimaryPosition), positions,
		p.Notes, p.Rating, p.Active, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

// FindByID busca un jugador por su UUID.
func (r *PlayerRepository) FindByID(ctx context.Context, id uuid.UUID) (*player.Player, error) {
	query := `
		SELECT id, name, primary_position, can_play_positions, notes, rating, active, created_at, updated_at
		FROM players WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	return scanPlayer(row)
}

// FindAll retorna todos los jugadores activos.
func (r *PlayerRepository) FindAll(ctx context.Context) ([]*player.Player, error) {
	query := `
		SELECT id, name, primary_position, can_play_positions, notes, rating, active, created_at, updated_at
		FROM players WHERE active = true ORDER BY name
	`
	return r.queryPlayers(ctx, query)
}

// FindAllIncludingInactive retorna todos los jugadores incluyendo inactivos.
func (r *PlayerRepository) FindAllIncludingInactive(ctx context.Context) ([]*player.Player, error) {
	query := `
		SELECT id, name, primary_position, can_play_positions, notes, rating, active, created_at, updated_at
		FROM players ORDER BY name
	`
	return r.queryPlayers(ctx, query)
}

// Update actualiza los datos de un jugador.
func (r *PlayerRepository) Update(ctx context.Context, p *player.Player) error {
	query := `
		UPDATE players SET
			name = $2, primary_position = $3, can_play_positions = $4,
			notes = $5, rating = $6, active = $7, updated_at = $8
		WHERE id = $1
	`
	positions := positionsToStrings(p.CanPlayPositions)
	_, err := r.pool.Exec(ctx, query,
		p.ID, p.Name, string(p.PrimaryPosition), positions,
		p.Notes, p.Rating, p.Active, p.UpdatedAt,
	)
	return err
}

// Delete elimina un jugador permanentemente.
func (r *PlayerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM players WHERE id = $1", id)
	return err
}

// --- helpers privados ---

func (r *PlayerRepository) queryPlayers(ctx context.Context, query string, args ...any) ([]*player.Player, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []*player.Player
	for rows.Next() {
		p, err := scanPlayer(rows)
		if err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

func scanPlayer(row pgx.Row) (*player.Player, error) {
	var p player.Player
	var posStr string
	var canPlayStrs []string

	err := row.Scan(
		&p.ID, &p.Name, &posStr, &canPlayStrs,
		&p.Notes, &p.Rating, &p.Active, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, player.ErrPlayerNotFound
		}
		return nil, err
	}

	p.PrimaryPosition = player.Position(posStr)
	p.CanPlayPositions = stringsToPositions(canPlayStrs)
	return &p, nil
}

func positionsToStrings(positions []player.Position) []string {
	result := make([]string, len(positions))
	for i, p := range positions {
		result[i] = string(p)
	}
	return result
}

func stringsToPositions(strs []string) []player.Position {
	result := make([]player.Position, len(strs))
	for i, s := range strs {
		result[i] = player.Position(s)
	}
	return result
}

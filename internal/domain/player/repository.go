package player

import (
	"context"

	"github.com/google/uuid"
)

// Repository es el PUERTO (interfaz) que define cómo persistir jugadores.
//
// Principio D de SOLID: la capa de aplicación depende de esta abstracción,
// no de una implementación concreta (Postgres, MySQL, memoria, etc.).
//
// Cualquier implementación que satisfaga esta interfaz es válida.
type Repository interface {
	// Save persiste un nuevo jugador.
	Save(ctx context.Context, player *Player) error

	// FindByID busca un jugador por su ID. Retorna ErrPlayerNotFound si no existe.
	FindByID(ctx context.Context, id uuid.UUID) (*Player, error)

	// FindAll retorna todos los jugadores activos.
	FindAll(ctx context.Context) ([]*Player, error)

	// FindAllIncludingInactive retorna todos los jugadores incluyendo los inactivos.
	FindAllIncludingInactive(ctx context.Context) ([]*Player, error)

	// Update actualiza los datos de un jugador existente.
	Update(ctx context.Context, player *Player) error

	// Delete elimina un jugador permanentemente (usar con cuidado, preferir Deactivate).
	Delete(ctx context.Context, id uuid.UUID) error
}

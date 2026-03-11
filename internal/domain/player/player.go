// Package player define la entidad Player y las reglas de negocio del dominio.
// Esta capa no tiene dependencias externas: ni base de datos, ni HTTP, ni frameworks.
package player

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Position representa la posición de un jugador en el campo.
type Position string

const (
	PositionDelantero Position = "delantero"
	PositionDefensa   Position = "defensa"
	PositionCreador   Position = "creador"
)

// Errores del dominio — reglas de negocio puras.
var (
	ErrEmptyName        = errors.New("el nombre del jugador no puede estar vacío")
	ErrInvalidRating    = errors.New("la calificación debe estar entre 1 y 10")
	ErrInvalidPosition  = errors.New("posición inválida: debe ser delantero, defensa o creador")
	ErrPlayerNotFound   = errors.New("jugador no encontrado")
	ErrPlayerInactive   = errors.New("el jugador está inactivo")
)

// Player es la entidad central del dominio.
// Contiene los datos y las reglas de negocio del jugador.
type Player struct {
	ID               uuid.UUID
	Name             string
	PrimaryPosition  Position
	CanPlayPositions []Position // posiciones secundarias
	Notes            string    // ej: "puede subir", "versátil"
	Rating           int8      // 1-10, solo visible para admins
	Active           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// New crea un nuevo Player aplicando las reglas de negocio del dominio.
// Es el constructor oficial — garantiza que ningún Player inválido exista.
func New(name string, pos Position, rating int8) (*Player, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if err := validateRating(rating); err != nil {
		return nil, err
	}
	if err := validatePosition(pos); err != nil {
		return nil, err
	}

	return &Player{
		ID:              uuid.New(),
		Name:            name,
		PrimaryPosition: pos,
		Rating:          rating,
		Active:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

// UpdateRating actualiza la calificación del jugador.
// Aplica la regla de negocio: rating debe estar entre 1 y 10.
func (p *Player) UpdateRating(rating int8) error {
	if err := validateRating(rating); err != nil {
		return err
	}
	p.Rating = rating
	p.UpdatedAt = time.Now()
	return nil
}

// Deactivate desactiva el jugador sin eliminarlo (soft delete).
func (p *Player) Deactivate() {
	p.Active = false
	p.UpdatedAt = time.Now()
}

// Activate reactiva un jugador previamente desactivado.
func (p *Player) Activate() {
	p.Active = true
	p.UpdatedAt = time.Now()
}

// CanPlay retorna true si el jugador puede jugar en la posición dada.
func (p *Player) CanPlay(pos Position) bool {
	if p.PrimaryPosition == pos {
		return true
	}
	for _, secondary := range p.CanPlayPositions {
		if secondary == pos {
			return true
		}
	}
	return false
}

// --- validaciones internas del dominio ---

func validateName(name string) error {
	if name == "" {
		return ErrEmptyName
	}
	return nil
}

func validateRating(rating int8) error {
	if rating < 1 || rating > 10 {
		return ErrInvalidRating
	}
	return nil
}

func validatePosition(pos Position) error {
	switch pos {
	case PositionDelantero, PositionDefensa, PositionCreador:
		return nil
	default:
		return ErrInvalidPosition
	}
}

// Package user define la entidad User y sus reglas de negocio.
package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Role define los niveles de acceso del sistema.
type Role string

const (
	RoleAdmin  Role = "admin"  // acceso total: edición y consulta
	RoleViewer Role = "viewer" // solo consulta: partidos e historial
)

var (
	ErrEmptyName       = errors.New("el nombre no puede estar vacío")
	ErrEmptyEmail      = errors.New("el email no puede estar vacío")
	ErrEmptyPassword   = errors.New("la contraseña no puede estar vacía")
	ErrInvalidRole     = errors.New("rol inválido: debe ser admin o viewer")
	ErrUserNotFound    = errors.New("usuario no encontrado")
	ErrEmailTaken      = errors.New("el email ya está registrado")
)

// User representa un usuario del sistema con su rol y credenciales.
type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

// New crea un nuevo User con validaciones del dominio.
// La contraseña que se recibe ya debe estar hasheada (responsabilidad de la capa de aplicación).
func New(name, email, passwordHash string, role Role) (*User, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	if email == "" {
		return nil, ErrEmptyEmail
	}
	if passwordHash == "" {
		return nil, ErrEmptyPassword
	}
	if err := validateRole(role); err != nil {
		return nil, err
	}

	return &User{
		ID:           uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
	}, nil
}

// IsAdmin retorna true si el usuario tiene rol administrador.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func validateRole(role Role) error {
	switch role {
	case RoleAdmin, RoleViewer:
		return nil
	default:
		return ErrInvalidRole
	}
}

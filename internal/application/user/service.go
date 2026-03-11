// Package user (application) contiene los casos de uso de autenticación y gestión de usuarios.
package user

import (
	"context"
	"fmt"

	"futbol-app/internal/domain/user"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service implementa los casos de uso de usuarios.
type Service struct {
	repo user.Repository
}

// NewService crea el servicio con inyección de dependencia.
func NewService(repo user.Repository) *Service {
	return &Service{repo: repo}
}

// CreateUserInput datos para crear un usuario.
type CreateUserInput struct {
	Name     string
	Email    string
	Password string
	Role     user.Role
}

// CreateUser registra un nuevo usuario hasheando su contraseña.
func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (*user.User, error) {
	// Verificar si el email ya existe
	existing, err := s.repo.FindByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return nil, user.ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hasheando contraseña: %w", err)
	}

	u, err := user.New(input.Name, input.Email, string(hash), input.Role)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Authenticate verifica credenciales y retorna el usuario si son válidas.
func (s *Service) Authenticate(ctx context.Context, email, password string) (*user.User, error) {
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, user.ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, user.ErrUserNotFound // no revelar si es email o password incorrecto
	}

	return u, nil
}

// GetUser obtiene un usuario por ID.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return s.repo.FindByID(ctx, id)
}

// ListUsers retorna todos los usuarios.
func (s *Service) ListUsers(ctx context.Context) ([]*user.User, error) {
	return s.repo.FindAll(ctx)
}

// DeleteUser elimina un usuario.
func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

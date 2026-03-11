package postgres

import (
	"context"
	"errors"

	"futbol-app/internal/domain/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository implementa user.Repository usando PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository crea el repositorio de usuarios.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, role, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query, u.ID, u.Name, u.Email, u.PasswordHash, string(u.Role), u.CreatedAt)
	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	query := `SELECT id, name, email, password_hash, role, created_at FROM users WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	return scanUser(row)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `SELECT id, name, email, password_hash, role, created_at FROM users WHERE email = $1`
	row := r.pool.QueryRow(ctx, query, email)
	return scanUser(row)
}

func (r *UserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, email, password_hash, role, created_at FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	query := `UPDATE users SET name = $2, email = $3, role = $4 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, u.ID, u.Name, u.Email, string(u.Role))
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

func scanUser(row pgx.Row) (*user.User, error) {
	var u user.User
	var roleStr string
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	u.Role = user.Role(roleStr)
	return &u, nil
}

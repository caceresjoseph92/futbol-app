// Package postgres contiene los adaptadores de base de datos.
// Implementa las interfaces (puertos) definidas en el dominio.
package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool crea un pool de conexiones a PostgreSQL usando la variable DATABASE_URL.
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL no está configurada")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("error conectando a PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error en ping a PostgreSQL: %w", err)
	}

	return pool, nil
}

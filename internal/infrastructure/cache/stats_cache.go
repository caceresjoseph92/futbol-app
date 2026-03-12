// Package cache implementa un caché en memoria para resultados costosos de calcular.
package cache

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"futbol-app/internal/domain/stats"
)

// StatsSource es la interfaz del origen de datos real (la DB).
// Al definir la interfaz aquí, el caché no depende de un paquete concreto.
// Esto se llama "dependency inversion" — dependemos de abstracciones, no implementaciones.
type StatsSource interface {
	GetSummary(ctx context.Context) (*stats.Summary, error)
}

// StatsCache envuelve un StatsSource y guarda el resultado en memoria por un TTL.
//
// Cómo funciona:
//  1. Primera llamada → va a la DB, guarda el resultado, anota la hora de expiración
//  2. Llamadas siguientes (dentro del TTL) → devuelve el resultado guardado sin ir a DB
//  3. Cuando expira (o se invalida) → vuelve a consultar la DB
//
// sync.RWMutex permite lecturas concurrentes (RLock) cuando el caché está caliente,
// y bloqueo exclusivo (Lock) solo cuando hay que refrescarlo.
type StatsCache struct {
	mu     sync.RWMutex
	data   *stats.Summary
	expiry time.Time
	ttl    time.Duration
	source StatsSource
}

// NewStatsCache crea el caché. ttl define cuánto tiempo son válidos los datos.
// Valor típico en producción: 5 minutos. En dev: 30 segundos para ver cambios rápido.
func NewStatsCache(source StatsSource, ttl time.Duration) *StatsCache {
	return &StatsCache{source: source, ttl: ttl}
}

// GetSummary devuelve las stats desde caché si están frescas, o las recalcula si no.
func (c *StatsCache) GetSummary(ctx context.Context) (*stats.Summary, error) {
	// Intento de lectura rápida con RLock (no bloquea otras lecturas concurrentes)
	c.mu.RLock()
	if c.data != nil && time.Now().Before(c.expiry) {
		data := c.data
		c.mu.RUnlock()
		slog.Debug("stats cache: HIT")
		return data, nil
	}
	c.mu.RUnlock()

	// Cache miss: necesitamos refrescar — Lock exclusivo
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check: otro goroutine puede haber refrescado mientras esperábamos el Lock
	if c.data != nil && time.Now().Before(c.expiry) {
		slog.Debug("stats cache: HIT (after lock)")
		return c.data, nil
	}

	slog.Info("stats cache: MISS — recalculando desde DB")
	data, err := c.source.GetSummary(ctx)
	if err != nil {
		return nil, err
	}

	c.data = data
	c.expiry = time.Now().Add(c.ttl)
	return c.data, nil
}

// Invalidate descarta el caché inmediatamente.
// Se llama cuando los datos cambian (ej: se registra el resultado de un partido).
func (c *StatsCache) Invalidate() {
	c.mu.Lock()
	c.data = nil
	c.mu.Unlock()
	slog.Info("stats cache: invalidado")
}

// Package main es el punto de entrada de la aplicación.
// Aquí se realiza la Inyección de Dependencias (Dependency Injection):
// se construyen todos los componentes en el orden correcto y se conectan.
//
// Principio: main.go es el único lugar que conoce todas las capas.
// Ninguna otra capa importa a otra directamente — todo se conecta aquí.
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	infrapostgres "futbol-app/internal/infrastructure/postgres"
	infracache "futbol-app/internal/infrastructure/cache"
	appplayer "futbol-app/internal/application/player"
	appmatch "futbol-app/internal/application/match"
	appstats "futbol-app/internal/application/stats"
	appuser "futbol-app/internal/application/user"
	httphandler "futbol-app/internal/interface/http"

	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno desde .env (solo en desarrollo)
	if err := godotenv.Load(); err != nil {
		log.Println("Archivo .env no encontrado, usando variables del sistema")
	}

	// Configurar slog con JSON estructurado (en producción) o texto legible (en dev)
	if os.Getenv("ENV") == "production" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	}

	ctx := context.Background()

	// ── Infraestructura ─────────────────────────────────────────────────────
	pool, err := infrapostgres.NewPool(ctx)
	if err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}
	defer pool.Close()

	// ── Repositorios (adaptadores) ───────────────────────────────────────────
	playerRepo := infrapostgres.NewPlayerRepository(pool)
	matchRepo  := infrapostgres.NewMatchRepository(pool)
	userRepo   := infrapostgres.NewUserRepository(pool)
	statsRepo  := infrapostgres.NewStatsRepository(pool)

	// ── Caché de estadísticas (5 min TTL) ───────────────────────────────────
	statsCache := infracache.NewStatsCache(statsRepo, 5*time.Minute)

	// ── Servicios de aplicación (casos de uso) ───────────────────────────────
	playerService := appplayer.NewService(playerRepo)
	matchService  := appmatch.NewService(matchRepo, playerRepo, statsRepo)
	userService   := appuser.NewService(userRepo)
	statsService  := appstats.NewService(statsCache)

	// ── Templates HTML ───────────────────────────────────────────────────────
	renderer, err := httphandler.NewRenderer()
	if err != nil {
		log.Fatalf("Error cargando templates: %v", err)
	}

	// ── SSE Hub (tiempo real) ────────────────────────────────────────────────
	sseHub := httphandler.NewSSEHub()

	// ── Handlers HTTP ────────────────────────────────────────────────────────
	authHandler   := httphandler.NewAuthHandler(userService, renderer)
	playerHandler := httphandler.NewPlayerHandler(playerService, statsService, renderer)
	matchHandler  := httphandler.NewMatchHandler(matchService, playerService, renderer, sseHub, statsCache)
	userHandler   := httphandler.NewUserHandler(userService, renderer)
	statsHandler  := httphandler.NewStatsHandler(statsService, renderer)

	// ── Router ───────────────────────────────────────────────────────────────
	router := httphandler.NewRouter(authHandler, playerHandler, matchHandler, userHandler, statsHandler, sseHub)

	// ── Servidor ─────────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor iniciado en http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}

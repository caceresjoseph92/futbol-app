// Package main es el punto de entrada de la aplicación.
// Aquí se realiza la Inyección de Dependencias (Dependency Injection):
// se construyen todos los componentes en el orden correcto y se conectan.
//
// Principio: main.go es el único lugar que conoce todas las capas.
// Ninguna otra capa importa a otra directamente — todo se conecta aquí.
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"

	infrapostgres "futbol-app/internal/infrastructure/postgres"
	appplayer "futbol-app/internal/application/player"
	appmatch "futbol-app/internal/application/match"
	appuser "futbol-app/internal/application/user"
	httphandler "futbol-app/internal/interface/http"

	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno desde .env (solo en desarrollo)
	if err := godotenv.Load(); err != nil {
		log.Println("Archivo .env no encontrado, usando variables del sistema")
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

	// ── Servicios de aplicación (casos de uso) ───────────────────────────────
	playerService := appplayer.NewService(playerRepo)
	matchService  := appmatch.NewService(matchRepo, playerRepo)
	userService   := appuser.NewService(userRepo)

	// ── Templates HTML ───────────────────────────────────────────────────────
	tmpl, err := template.ParseGlob("internal/interface/templates/**/*.html")
	if err != nil {
		log.Fatalf("Error cargando templates: %v", err)
	}

	// ── Handlers HTTP ────────────────────────────────────────────────────────
	authHandler   := httphandler.NewAuthHandler(userService, tmpl)
	playerHandler := httphandler.NewPlayerHandler(playerService, tmpl)
	matchHandler  := httphandler.NewMatchHandler(matchService, playerService, tmpl)
	userHandler   := httphandler.NewUserHandler(userService, tmpl)

	// ── Router ───────────────────────────────────────────────────────────────
	router := httphandler.NewRouter(authHandler, playerHandler, matchHandler, userHandler)

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

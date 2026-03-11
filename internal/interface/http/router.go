package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter configura todas las rutas de la aplicación.
// Aplica middlewares globales y define los grupos de rutas por rol.
func NewRouter(
	authHandler *AuthHandler,
	playerHandler *PlayerHandler,
	matchHandler *MatchHandler,
	userHandler *UserHandler,
) http.Handler {
	r := chi.NewRouter()

	// Middlewares globales
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)

	// Archivos estáticos
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Rutas públicas
	r.Get("/login", authHandler.ShowLogin)
	r.Post("/login", authHandler.Login)
	r.Post("/logout", authHandler.Logout)

	// Rutas autenticadas (admin y viewer)
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		// Vista pública del partido activo (sin ratings)
		r.Get("/", matchHandler.ShowCurrent)
		r.Get("/matches/{id}", matchHandler.ShowMatch)
		r.Get("/matches/{id}/share", matchHandler.ShareView) // vista para copiar a WhatsApp
		r.Get("/history", matchHandler.History)
	})

	// Rutas solo admin
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Use(AdminOnly)

		// Jugadores
		r.Get("/admin/players", playerHandler.List)
		r.Get("/admin/players/new", playerHandler.ShowCreate)
		r.Post("/admin/players", playerHandler.Create)
		r.Get("/admin/players/{id}/edit", playerHandler.ShowEdit)
		r.Put("/admin/players/{id}", playerHandler.Update)
		r.Patch("/admin/players/{id}/rating", playerHandler.UpdateRating)
		r.Patch("/admin/players/{id}/deactivate", playerHandler.Deactivate)
		r.Patch("/admin/players/{id}/activate", playerHandler.Activate)

		// Partidos
		r.Get("/admin/matches/new", matchHandler.ShowCreate)
		r.Post("/admin/matches", matchHandler.Create)
		r.Get("/admin/matches/{id}/edit", matchHandler.ShowEdit)
		r.Post("/admin/matches/{id}/players", matchHandler.AddPlayers)
		r.Post("/admin/matches/{id}/generate", matchHandler.GenerateTeams)
		r.Post("/admin/matches/{id}/transfer/{playerID}", matchHandler.TransferPlayer)
		r.Post("/admin/matches/{id}/swap", matchHandler.SwapPlayers)
		r.Post("/admin/matches/{id}/publish", matchHandler.Publish)
		r.Post("/admin/matches/{id}/finish", matchHandler.Finish)

		// Usuarios
		r.Get("/admin/users", userHandler.List)
		r.Get("/admin/users/new", userHandler.ShowCreate)
		r.Post("/admin/users", userHandler.Create)
		r.Delete("/admin/users/{id}", userHandler.Delete)
	})

	return r
}

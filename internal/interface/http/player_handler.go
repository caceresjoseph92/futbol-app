package http

import (
	"net/http"
	"strconv"

	appplayer "futbol-app/internal/application/player"
	"futbol-app/internal/domain/player"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PlayerHandler maneja las rutas de jugadores.
type PlayerHandler struct {
	service *appplayer.Service
	tmpl    *Renderer
}

// NewPlayerHandler crea el handler de jugadores.
func NewPlayerHandler(service *appplayer.Service, tmpl *Renderer) *PlayerHandler {
	return &PlayerHandler{service: service, tmpl: tmpl}
}

// List muestra la lista de todos los jugadores (solo admin ve ratings).
func (h *PlayerHandler) List(w http.ResponseWriter, r *http.Request) {
	players, err := h.service.ListAllPlayers(r.Context())
	if err != nil {
		http.Error(w, "Error cargando jugadores", http.StatusInternalServerError)
		return
	}
	h.tmpl.ExecuteTemplate(w, "players/list.html", withFlash(w, r, map[string]any{
		"Players": players,
	}))
}

// ShowCreate muestra el formulario de creación.
func (h *PlayerHandler) ShowCreate(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "players/form.html", withFlash(w, r, map[string]any{
		"Action": "create",
	}))
}

// Create procesa la creación de un jugador.
func (h *PlayerHandler) Create(w http.ResponseWriter, r *http.Request) {
	input, err := parsePlayerForm(r)
	if err != nil {
		h.tmpl.ExecuteTemplate(w, "players/form.html", withFlash(w, r, map[string]any{
			"Error":  err.Error(),
			"Action": "create",
		}))
		return
	}

	if _, err := h.service.CreatePlayer(r.Context(), input); err != nil {
		h.tmpl.ExecuteTemplate(w, "players/form.html", withFlash(w, r, map[string]any{
			"Error":  err.Error(),
			"Action": "create",
		}))
		return
	}

	setFlash(w, "success", "Jugador creado correctamente")
	http.Redirect(w, r, "/admin/players", http.StatusSeeOther)
}

// ShowEdit muestra el formulario de edición.
func (h *PlayerHandler) ShowEdit(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	p, err := h.service.GetPlayer(r.Context(), id)
	if err != nil {
		http.Error(w, "Jugador no encontrado", http.StatusNotFound)
		return
	}

	canPlay := map[string]bool{}
	for _, pos := range p.CanPlayPositions {
		canPlay[string(pos)] = true
	}
	h.tmpl.ExecuteTemplate(w, "players/form.html", withFlash(w, r, map[string]any{
		"Player":  p,
		"Action":  "edit",
		"CanPlay": canPlay,
	}))
}

// Update procesa la actualización de un jugador.
func (h *PlayerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	input, err := parsePlayerForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := h.service.UpdatePlayer(r.Context(), id, appplayer.UpdatePlayerInput{
		Name:            input.Name,
		PrimaryPosition: input.PrimaryPosition,
		CanPlayPositions: input.CanPlayPositions,
		Notes:           input.Notes,
		Rating:          input.Rating,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	setFlash(w, "success", "Jugador actualizado correctamente")
	http.Redirect(w, r, "/admin/players", http.StatusSeeOther)
}

// UpdateRating actualiza solo el rating (HTMX inline).
func (h *PlayerHandler) UpdateRating(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	ratingStr := r.FormValue("rating")
	rating, err := strconv.ParseInt(ratingStr, 10, 8)
	if err != nil {
		http.Error(w, "Rating inválido", http.StatusBadRequest)
		return
	}

	p, err := h.service.UpdateRating(r.Context(), id, int8(rating))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// HTMX: retorna solo el fragmento actualizado
	h.tmpl.ExecuteTemplate(w, "partials/player_rating.html", p)
}

// Deactivate desactiva un jugador.
func (h *PlayerHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	if err := h.service.DeactivatePlayer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setFlash(w, "success", "Jugador desactivado")
	http.Redirect(w, r, "/admin/players", http.StatusSeeOther)
}

// Activate reactiva un jugador.
func (h *PlayerHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	if err := h.service.ActivatePlayer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setFlash(w, "success", "Jugador activado")
	http.Redirect(w, r, "/admin/players", http.StatusSeeOther)
}

// --- helpers ---

func parsePlayerForm(r *http.Request) (appplayer.CreatePlayerInput, error) {
	r.ParseForm()
	ratingStr := r.FormValue("rating")
	rating, err := strconv.ParseInt(ratingStr, 10, 8)
	if err != nil {
		return appplayer.CreatePlayerInput{}, player.ErrInvalidRating
	}

	var canPlay []player.Position
	for _, p := range r.Form["can_play_positions"] {
		canPlay = append(canPlay, player.Position(p))
	}

	return appplayer.CreatePlayerInput{
		Name:             r.FormValue("name"),
		PrimaryPosition:  player.Position(r.FormValue("primary_position")),
		CanPlayPositions: canPlay,
		Notes:            r.FormValue("notes"),
		Rating:           int8(rating),
	}, nil
}

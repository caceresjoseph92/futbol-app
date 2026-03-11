package http

import (
	"net/http"
	"strconv"
	"time"

	appmatch "futbol-app/internal/application/match"
	appplayer "futbol-app/internal/application/player"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// MatchHandler maneja las rutas de partidos.
type MatchHandler struct {
	matchService  *appmatch.Service
	playerService *appplayer.Service
	tmpl          *Renderer
}

// NewMatchHandler crea el handler de partidos.
func NewMatchHandler(ms *appmatch.Service, ps *appplayer.Service, tmpl *Renderer) *MatchHandler {
	return &MatchHandler{matchService: ms, playerService: ps, tmpl: tmpl}
}

// ShowCurrent muestra el partido activo (publicado más reciente).
func (h *MatchHandler) ShowCurrent(w http.ResponseWriter, r *http.Request) {
	matches, err := h.matchService.ListMatches(r.Context())
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	var current interface{}
	for _, m := range matches {
		if m.Status == "published" || m.Status == "draft" {
			current = m
			break
		}
	}

	h.tmpl.ExecuteTemplate(w, "matches/current.html", withFlash(w, r, map[string]any{
		"Match":   current,
		"IsAdmin": IsAdmin(r.Context()),
	}))
}

// ShowMatch muestra un partido específico.
func (h *MatchHandler) ShowMatch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	m, err := h.matchService.GetMatch(r.Context(), id)
	if err != nil {
		http.Error(w, "Partido no encontrado", http.StatusNotFound)
		return
	}

	h.tmpl.ExecuteTemplate(w, "matches/detail.html", withFlash(w, r, map[string]any{
		"Match":   m,
		"IsAdmin": IsAdmin(r.Context()),
	}))
}

// ShareView muestra la vista limpia para copiar a WhatsApp (sin ratings).
func (h *MatchHandler) ShareView(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	m, err := h.matchService.GetMatch(r.Context(), id)
	if err != nil {
		http.Error(w, "Partido no encontrado", http.StatusNotFound)
		return
	}

	h.tmpl.ExecuteTemplate(w, "matches/share.html", map[string]any{
		"Match": m,
		"Team1": m.Team1(),
		"Team2": m.Team2(),
	})
}

// History muestra el historial de partidos.
func (h *MatchHandler) History(w http.ResponseWriter, r *http.Request) {
	matches, err := h.matchService.ListMatches(r.Context())
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	h.tmpl.ExecuteTemplate(w, "matches/history.html", withFlash(w, r, map[string]any{
		"Matches": matches,
		"IsAdmin": IsAdmin(r.Context()),
	}))
}

// ShowCreate muestra el formulario para crear un partido.
func (h *MatchHandler) ShowCreate(w http.ResponseWriter, r *http.Request) {
	players, err := h.playerService.ListPlayers(r.Context())
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	h.tmpl.ExecuteTemplate(w, "matches/form.html", withFlash(w, r, map[string]any{
		"Players": players,
	}))
}

// Create crea un nuevo partido en estado draft.
func (h *MatchHandler) Create(w http.ResponseWriter, r *http.Request) {
	dateStr := r.FormValue("played_at")
	playedAt, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		playedAt = time.Now()
	}

	userIDStr, _ := r.Context().Value(contextKeyUser).(string)
	userID, _ := uuid.Parse(userIDStr)

	m, err := h.matchService.CreateMatch(r.Context(), appmatch.CreateMatchInput{
		PlayedAt:       playedAt,
		GoalkeeperInfo: r.FormValue("goalkeeper_info"),
		CreatedBy:      userID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setFlash(w, "success", "Partido creado correctamente")
	http.Redirect(w, r, "/admin/matches/"+m.ID.String()+"/edit", http.StatusSeeOther)
}

// ShowEdit muestra la pantalla de edición del partido (asignar jugadores, equipos).
func (h *MatchHandler) ShowEdit(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	m, err := h.matchService.GetMatch(r.Context(), id)
	if err != nil {
		http.Error(w, "Partido no encontrado", http.StatusNotFound)
		return
	}

	players, err := h.playerService.ListPlayers(r.Context())
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	h.tmpl.ExecuteTemplate(w, "matches/edit.html", withFlash(w, r, map[string]any{
		"Match":   m,
		"Players": players,
		"Team1":   m.Team1(),
		"Team2":   m.Team2(),
	}))
}

// AddPlayers agrega los 12 jugadores convocados al partido.
func (h *MatchHandler) AddPlayers(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	r.ParseForm()
	var playerIDs []uuid.UUID
	for _, pidStr := range r.Form["player_ids"] {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			continue
		}
		playerIDs = append(playerIDs, pid)
	}

	if _, err := h.matchService.AddPlayersToMatch(r.Context(), id, playerIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	setFlash(w, "success", "Convocados guardados")
	http.Redirect(w, r, "/admin/matches/"+id.String()+"/edit", http.StatusSeeOther)
}

// GenerateTeams ejecuta el algoritmo de balanceo automático.
func (h *MatchHandler) GenerateTeams(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	if _, err := h.matchService.GenerateTeams(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	setFlash(w, "success", "Equipos generados automáticamente")
	http.Redirect(w, r, "/admin/matches/"+id.String()+"/edit", http.StatusSeeOther)
}

// TransferPlayer mueve un jugador al equipo contrario.
func (h *MatchHandler) TransferPlayer(w http.ResponseWriter, r *http.Request) {
	matchID, _ := uuid.Parse(chi.URLParam(r, "id"))
	playerID, _ := uuid.Parse(chi.URLParam(r, "playerID"))

	m, err := h.matchService.TransferPlayer(r.Context(), matchID, playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// HTMX: retorna solo la sección de equipos actualizada
	h.tmpl.ExecuteTemplate(w, "partials/teams.html", map[string]any{
		"Match": m,
		"Team1": m.Team1(),
		"Team2": m.Team2(),
	})
}

// SwapPlayers intercambia dos jugadores entre equipos.
func (h *MatchHandler) SwapPlayers(w http.ResponseWriter, r *http.Request) {
	matchID, _ := uuid.Parse(chi.URLParam(r, "id"))
	playerID1, _ := uuid.Parse(r.FormValue("player1_id"))
	playerID2, _ := uuid.Parse(r.FormValue("player2_id"))

	m, err := h.matchService.SwapPlayers(r.Context(), matchID, playerID1, playerID2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.tmpl.ExecuteTemplate(w, "partials/teams.html", map[string]any{
		"Match": m,
		"Team1": m.Team1(),
		"Team2": m.Team2(),
	})
}

// Publish publica el partido.
func (h *MatchHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	if _, err := h.matchService.PublishMatch(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "success", "Equipos publicados")
	http.Redirect(w, r, "/matches/"+id.String(), http.StatusSeeOther)
}

// UpdateDate actualiza la fecha y arqueros del partido.
func (h *MatchHandler) UpdateDate(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	dateStr := r.FormValue("played_at")
	playedAt, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Fecha inválida", http.StatusBadRequest)
		return
	}
	if _, err := h.matchService.UpdateMatchDate(r.Context(), id, playedAt, r.FormValue("goalkeeper_info")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setFlash(w, "success", "Fecha y arqueros actualizados")
	http.Redirect(w, r, "/admin/matches/"+id.String()+"/edit", http.StatusSeeOther)
}

// Finish registra el resultado del partido.
func (h *MatchHandler) Finish(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	score1, _ := strconv.Atoi(r.FormValue("score1"))
	score2, _ := strconv.Atoi(r.FormValue("score2"))

	if _, err := h.matchService.FinishMatch(r.Context(), id, score1, score2); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "success", "Resultado registrado")
	http.Redirect(w, r, "/matches/"+id.String(), http.StatusSeeOther)
}

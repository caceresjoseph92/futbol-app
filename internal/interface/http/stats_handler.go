package http

import (
	"net/http"

	appstats "futbol-app/internal/application/stats"
)

// StatsHandler maneja la vista de estadísticas del grupo.
type StatsHandler struct {
	service *appstats.Service
	tmpl    *Renderer
}

// NewStatsHandler crea el handler de estadísticas.
func NewStatsHandler(service *appstats.Service, tmpl *Renderer) *StatsHandler {
	return &StatsHandler{service: service, tmpl: tmpl}
}

// Show muestra el resumen de estadísticas.
func (h *StatsHandler) Show(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetSummary(r.Context())
	if err != nil {
		http.Error(w, "Error cargando estadísticas", http.StatusInternalServerError)
		return
	}
	h.tmpl.ExecuteTemplate(w, "stats/index.html", withFlash(w, r, map[string]any{
		"Summary": summary,
		"IsAdmin": IsAdmin(r.Context()),
	}))
}

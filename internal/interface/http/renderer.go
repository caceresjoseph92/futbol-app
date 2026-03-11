package http

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

const templatesDir = "internal/interface/templates"

// funcMap contiene funciones auxiliares disponibles en todos los templates.
var funcMap = template.FuncMap{
	"add": func(a, b int) int { return a + b },
}

// Renderer compila un set de templates por página, evitando el problema
// de múltiples {{define "content"}} sobreescribiéndose en un set global.
type Renderer struct {
	templates map[string]*template.Template
}

func NewRenderer() (*Renderer, error) {
	r := &Renderer{templates: make(map[string]*template.Template)}

	layout := templatesDir + "/layout.html"
	teams  := templatesDir + "/partials/teams.html"

	// Páginas que usan el layout (tienen {{define "content"}})
	pages := []string{
		"matches/current.html",
		"matches/edit.html",
		"matches/history.html",
		"matches/detail.html",
		"matches/form.html",
		"players/list.html",
		"players/form.html",
		"users/list.html",
		"users/form.html",
		"stats/index.html",
	}
	for _, page := range pages {
		t, err := template.New("").Funcs(funcMap).ParseFiles(layout, teams, templatesDir+"/"+page)
		if err != nil {
			return nil, fmt.Errorf("error cargando %s: %w", page, err)
		}
		r.templates[page] = t
	}

	// Templates standalone (sin layout)
	standalones := []string{
		"auth/login.html",
		"matches/share.html",
	}
	for _, s := range standalones {
		t, err := template.New("").Funcs(funcMap).ParseFiles(templatesDir + "/" + s)
		if err != nil {
			return nil, fmt.Errorf("error cargando %s: %w", s, err)
		}
		r.templates[s] = t
	}

	// Partials (respuestas HTMX — tienen su propio {{define}})
	partials := []string{
		"partials/teams.html",
		"partials/player_rating.html",
	}
	for _, p := range partials {
		t, err := template.New("").Funcs(funcMap).ParseFiles(templatesDir + "/" + p)
		if err != nil {
			return nil, fmt.Errorf("error cargando %s: %w", p, err)
		}
		r.templates[p] = t
	}

	return r, nil
}

// ExecuteTemplate renderiza el template correcto según su tipo.
func (r *Renderer) ExecuteTemplate(w io.Writer, name string, data any) error {
	t, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("template no encontrado: %s", name)
	}
	// Partials: ejecutar el bloque definido por nombre
	if strings.HasPrefix(name, "partials/") {
		return t.ExecuteTemplate(w, name, data)
	}
	// Standalone: ejecutar directamente
	if name == "auth/login.html" || name == "matches/share.html" {
		return t.Execute(w, data)
	}
	// Páginas con layout
	return t.ExecuteTemplate(w, "layout.html", data)
}

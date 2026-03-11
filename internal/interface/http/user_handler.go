package http

import (
	"net/http"

	appuser "futbol-app/internal/application/user"
	"futbol-app/internal/domain/user"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// UserHandler maneja la gestión de usuarios (solo admin).
type UserHandler struct {
	service *appuser.Service
	tmpl    *Renderer
}

// NewUserHandler crea el handler de usuarios.
func NewUserHandler(service *appuser.Service, tmpl *Renderer) *UserHandler {
	return &UserHandler{service: service, tmpl: tmpl}
}

// List muestra la lista de usuarios.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	h.tmpl.ExecuteTemplate(w, "users/list.html", withFlash(w, r, map[string]any{
		"Users": users,
	}))
}

// ShowCreate muestra el formulario de creación de usuario.
func (h *UserHandler) ShowCreate(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "users/form.html", withFlash(w, r, map[string]any{}))
}

// Create crea un nuevo usuario.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	input := appuser.CreateUserInput{
		Name:     r.FormValue("name"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
		Role:     user.Role(r.FormValue("role")),
	}

	if _, err := h.service.CreateUser(r.Context(), input); err != nil {
		h.tmpl.ExecuteTemplate(w, "users/form.html", withFlash(w, r, map[string]any{
			"Error": err.Error(),
		}))
		return
	}

	setFlash(w, "success", "Usuario creado correctamente")
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// Delete elimina un usuario.
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteUser(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

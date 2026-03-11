package http

import (
	"html/template"
	"net/http"
	"os"
	"time"

	appuser "futbol-app/internal/application/user"
	"futbol-app/internal/domain/user"

	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler maneja autenticación (login/logout).
type AuthHandler struct {
	userService *appuser.Service
	tmpl        *template.Template
}

// NewAuthHandler crea el handler de autenticación.
func NewAuthHandler(userService *appuser.Service, tmpl *template.Template) *AuthHandler {
	return &AuthHandler{userService: userService, tmpl: tmpl}
}

// ShowLogin muestra el formulario de login.
func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "auth/login.html", nil)
}

// Login procesa las credenciales y emite el JWT en una cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	u, err := h.userService.Authenticate(r.Context(), email, password)
	if err != nil {
		h.tmpl.ExecuteTemplate(w, "auth/login.html", map[string]string{
			"Error": "Credenciales inválidas",
		})
		return
	}

	token, err := generateToken(u.ID.String(), string(u.Role))
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout elimina la cookie de sesión.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func generateToken(userID, role string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	claims := &Claims{
		UserID: userID,
		Role:   user.Role(role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}


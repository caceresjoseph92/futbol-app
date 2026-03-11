package http

import (
	"context"
	"net/http"
	"os"
	"strings"

	"futbol-app/internal/domain/user"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey tipo privado para evitar colisiones en el contexto HTTP.
type contextKey string

const (
	contextKeyUser contextKey = "user"
	contextKeyRole contextKey = "role"
)

// Claims define la estructura del JWT.
type Claims struct {
	UserID string    `json:"user_id"`
	Role   user.Role `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware verifica el JWT en la cookie o header Authorization.
// Si el token es válido, inyecta el usuario en el contexto.
// Si no hay token o es inválido, redirige a /login.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		claims, err := parseToken(tokenStr)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyUser, claims.UserID)
		ctx = context.WithValue(ctx, contextKeyRole, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware intenta leer el JWT pero nunca redirige.
// Si hay token válido, inyecta el usuario en el contexto (permite saber si es admin).
// Si no hay token o es inválido, continúa sin usuario en el contexto.
func OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr != "" {
			if claims, err := parseToken(tokenStr); err == nil {
				ctx := context.WithValue(r.Context(), contextKeyUser, claims.UserID)
				ctx = context.WithValue(ctx, contextKeyRole, claims.Role)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// AdminOnly permite solo a usuarios con rol admin.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(contextKeyRole).(user.Role)
		if !ok || role != user.RoleAdmin {
			http.Error(w, "Acceso denegado", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserRole extrae el rol del usuario desde el contexto.
func GetUserRole(ctx context.Context) user.Role {
	role, _ := ctx.Value(contextKeyRole).(user.Role)
	return role
}

// IsAdmin retorna true si el usuario en el contexto es admin.
func IsAdmin(ctx context.Context) bool {
	return GetUserRole(ctx) == user.RoleAdmin
}

func extractToken(r *http.Request) string {
	// Primero buscar en cookie
	cookie, err := r.Cookie("token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	// Luego en header Authorization: Bearer <token>
	bearer := r.Header.Get("Authorization")
	if strings.HasPrefix(bearer, "Bearer ") {
		return strings.TrimPrefix(bearer, "Bearer ")
	}
	return ""
}

func parseToken(tokenStr string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

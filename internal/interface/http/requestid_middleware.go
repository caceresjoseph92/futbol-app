package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const contextKeyRequestID contextKey = "request_id"

// RequestIDMiddleware genera un ID único por request y lo propaga en:
//   - El contexto HTTP (para que otros middlewares y handlers lo lean)
//   - El header de respuesta X-Request-ID (para correlación desde el cliente)
//
// Si el cliente ya envía X-Request-ID, lo reutilizamos. Esto permite rastrear
// una cadena de requests entre microservicios (distributed tracing básico).
//
// ¿Por qué es útil?
//   - En los logs, cada línea tiene el mismo request_id → filtrás por él y ves
//     todo lo que pasó en esa request específica.
//   - Cuando un usuario reporta un error, le pedís el X-Request-ID del header
//     y en segundos encontrás los logs relevantes.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}

		// Inyectar en contexto para que LoggingMiddleware y handlers lo lean
		ctx := context.WithValue(r.Context(), contextKeyRequestID, id)

		// Devolver el ID en la respuesta para correlación del lado del cliente
		w.Header().Set("X-Request-ID", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package http

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter captura el status code que el handler escribe.
// Necesitamos esto porque http.ResponseWriter no expone el status por defecto.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// LoggingMiddleware registra cada request con método, ruta, status y duración.
// Usa slog (stdlib desde Go 1.21) que produce JSON estructurado — ideal para
// servicios en producción donde los logs se parsean con herramientas externas.
//
// Log de ejemplo:
//
//	{"time":"...","level":"INFO","msg":"request","method":"GET","path":"/stats","status":200,"ms":12}
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Envolvemos el ResponseWriter para interceptar el status code
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"ms", time.Since(start).Milliseconds(),
		)
	})
}

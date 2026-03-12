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
// Lee el request_id del contexto (inyectado por RequestIDMiddleware) para
// que cada línea de log quede vinculada a su request específico.
//
// Log de ejemplo:
//
//	{"time":"...","level":"INFO","msg":"request","request_id":"abc-123","method":"GET","path":"/stats","status":200,"ms":12}
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Envolvemos el ResponseWriter para interceptar el status code
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)

		// El request_id fue inyectado por RequestIDMiddleware (que corre antes)
		reqID, _ := r.Context().Value(contextKeyRequestID).(string)
		slog.Info("request",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"ms", time.Since(start).Milliseconds(),
		)
	})
}

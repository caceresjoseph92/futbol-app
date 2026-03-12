package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

// SSEHub gestiona todos los clientes conectados via Server-Sent Events.
//
// SSE es una tecnología del navegador para recibir eventos del servidor en
// tiempo real usando una conexión HTTP normal (no WebSocket). El servidor
// mantiene la conexión abierta y envía "eventos" en texto plano.
//
// Protocolo SSE:
//
//	event: nombre_del_evento\n
//	data: contenido\n
//	\n   ← línea vacía = fin del evento
//
// El hub usa un map de canales: cada cliente conectado tiene su canal.
// sync.Mutex protege el map contra accesos concurrentes (varios usuarios
// conectándose/desconectándose al mismo tiempo).
type SSEHub struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
}

// NewSSEHub crea el hub. Se crea UNA vez en main.go y se pasa a los handlers.
func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[chan string]struct{})}
}

// Subscribe registra un nuevo cliente y retorna su canal de eventos.
// El canal tiene buffer 4 para evitar bloquear el Broadcast si el cliente es lento.
func (h *SSEHub) Subscribe() chan string {
	ch := make(chan string, 4)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	slog.Debug("sse: cliente conectado", "total", len(h.clients))
	return ch
}

// Unsubscribe elimina el cliente del hub y cierra su canal.
// Se llama cuando el cliente se desconecta (navegador cierra la pestaña, etc.).
func (h *SSEHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
	slog.Debug("sse: cliente desconectado", "total", len(h.clients))
}

// Broadcast envía un evento a TODOS los clientes conectados.
// El select con default evita bloquearse si el canal de algún cliente está lleno.
func (h *SSEHub) Broadcast(event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- event:
		default: // cliente lento — se pierde el evento (no bloqueamos)
		}
	}
	slog.Info("sse: broadcast enviado", "event", event, "clientes", len(h.clients))
}

// ServeSSE es el handler HTTP del endpoint /events.
// El navegador se conecta aquí y queda esperando eventos indefinidamente.
func (h *SSEHub) ServeSSE(w http.ResponseWriter, r *http.Request) {
	// Headers obligatorios para SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Evitar que Nginx/proxies almacenen en buffer la respuesta
	w.Header().Set("X-Accel-Buffering", "no")

	// http.Flusher permite enviar datos parciales sin cerrar la conexión
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming no soportado", http.StatusInternalServerError)
		return
	}

	ch := h.Subscribe()
	defer h.Unsubscribe(ch)

	// Evento inicial para confirmar conexión
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	// Esperar eventos o desconexión del cliente
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: match_updated\ndata: %s\n\n", event)
			flusher.Flush()
		case <-r.Context().Done():
			// El cliente cerró la conexión (navegador, timeout, etc.)
			return
		}
	}
}

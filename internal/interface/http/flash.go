package http

import (
	"net/http"
	"net/url"
)

const flashCookie = "flash"

// withFlash inyecta el flash (si existe) en el mapa de datos del template.
func withFlash(w http.ResponseWriter, r *http.Request, data map[string]any) map[string]any {
	kind, msg := getFlash(w, r)
	data["Flash"] = msg
	data["FlashKind"] = kind
	return data
}

// setFlash guarda un mensaje de flash en una cookie de un solo uso.
func setFlash(w http.ResponseWriter, kind, message string) {
	val := url.QueryEscape(kind + "|" + message)
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookie,
		Value:    val,
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// getFlash lee y borra el mensaje de flash. Retorna (kind, message).
func getFlash(w http.ResponseWriter, r *http.Request) (string, string) {
	c, err := r.Cookie(flashCookie)
	if err != nil {
		return "", ""
	}
	// Borrar la cookie
	http.SetCookie(w, &http.Cookie{
		Name:   flashCookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	decoded, err := url.QueryUnescape(c.Value)
	if err != nil {
		return "", ""
	}
	for i, ch := range decoded {
		if ch == '|' {
			return decoded[:i], decoded[i+1:]
		}
	}
	return "", decoded
}

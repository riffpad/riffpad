package hub

import (
	"net/http"
	"strings"

	"github.com/riffpad/riffpad/packages/webui"
)

// apiOnly reports whether this vhost is an API-only host (e.g. api.riffpad.ai):
// API and WebSocket routes keep working, but the web UI is not served here.
func (h *Hub) apiOnly(r *http.Request) bool {
	host := strings.ToLower(strings.Split(r.Host, ":")[0])
	for _, hh := range h.apiHosts {
		if strings.ToLower(hh) == host {
			return true
		}
	}
	return false
}

func (h *Hub) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/device" {
		http.NotFound(w, r)
		return
	}
	if h.apiOnly(r) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	raw, err := webui.IndexHTML()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "webui not built"})
		return
	}
	html := strings.Replace(string(raw), "</head>", "<script>window.RIFFPAD_RELAY=1;</script></head>", 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func (h *Hub) handleAsset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	data, err := webui.Asset(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", webui.ContentType(name))
	_, _ = w.Write(data)
}

func (h *Hub) handleStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	hosts, sessions := len(h.hosts), len(h.sessions)
	h.mu.Unlock()
	var users, devices int64
	h.store.db.Model(&User{}).Count(&users)
	h.store.db.Model(&Device{}).Count(&devices)
	writeJSON(w, http.StatusOK, map[string]any{
		"name": "riffpad-relay", "version": version,
		"hosts": hosts, "sessions": sessions, "users": users, "devices": devices,
	})
}

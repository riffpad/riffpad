package hub

import (
	"net/http"
	"strings"
)

// handleDevices lists devices paired to the authenticated user's hosts.
func (h *Hub) handleDevices(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	list, err := h.store.DevicesForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": list})
}

// handleDeviceDelete revokes one paired device and disconnects it immediately.
func (h *Hub) handleDeviceDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if err := h.store.DeleteDevice(id, u.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	n := h.disconnectViewers(func(v *viewerConn) bool { return v.deviceID == id })
	h.log.Printf("revoked device=%s user=%s disconnected=%d", id, u.Username, n)
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true, "disconnected": n})
}

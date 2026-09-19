package hub

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Hub) handleRegisterHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	rec, err := h.store.CreateHost(u.ID, req.Name, newSecret())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "register host failed")
		return
	}
	h.log.Printf("host registered id=%s name=%s owner=%s", rec.ID, rec.Name, u.Username)
	writeJSON(w, http.StatusOK, map[string]any{"hostId": rec.ID, "hostSecret": rec.Secret})
}

// handleHosts lists the hosts owned by the authenticated user. The daemon CLI
// uses it after login to detect stale host credentials from a previous
// account.
func (h *Hub) handleHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	hosts, err := h.store.HostsForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hosts": hosts})
}

// handleHostKillswitch revokes every device paired to a host and disconnects
// all of its viewers (one-action kill switch).
func (h *Hub) handleHostKillswitch(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	hostID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/hosts/"), "/killswitch")
	host, err := h.store.GetHost(hostID)
	if err != nil || host.OwnerID != u.ID {
		writeError(w, http.StatusNotFound, "host not found")
		return
	}
	if err := h.store.DeleteDevicesForHost(hostID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	n := h.disconnectViewers(func(v *viewerConn) bool { return v.host.id == hostID })
	h.log.Printf("killswitch host=%s user=%s devices-revoked disconnected=%d", hostID, u.Username, n)
	writeJSON(w, http.StatusOK, map[string]any{"killed": true, "disconnected": n})
}

// disconnectViewers closes matching viewer connections and removes them.
func (h *Hub) disconnectViewers(match func(*viewerConn) bool) int {
	h.mu.Lock()
	var targets []*viewerConn
	for _, v := range h.viewers {
		if match(v) {
			targets = append(targets, v)
		}
	}
	for _, v := range targets {
		delete(h.viewers, v.id)
		h.hostSend(v.host, hostFrame{Kind: "leave", ViewerID: v.id})
		v.closeDone()
		_ = v.conn.Close()
	}
	h.mu.Unlock()
	return len(targets)
}

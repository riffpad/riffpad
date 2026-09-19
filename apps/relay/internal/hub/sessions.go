package hub

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

// SessionView is a host-announced SessionMeta with the account's client-side
// meta layered on top (custom display name / hidden state). The raw
// SessionMeta is a persisted table, so the client-only fields live on this
// response type instead of on the model.
type SessionView struct {
	SessionMeta
	DisplayName string `json:"displayName"`
	Hidden      bool   `json:"hidden"`
}

func (h *Hub) handleSessions(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	hostIDs, err := h.store.HostIDsForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	clientMeta, err := h.store.SessionClientMetaForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	metaByKey := map[string]SessionClientMeta{}
	for _, m := range clientMeta {
		metaByKey[m.HostID+"|"+m.SessionID] = m
	}
	h.mu.Lock()
	live := map[string]bool{}
	for id := range h.hosts {
		live[id] = true
	}
	// Serve the live snapshot announced by each host, not the accumulated
	// database history: sessions that exited (or hosts that restarted) must
	// not appear in the client list. hostIDs keeps owner isolation intact.
	owners := map[string]bool{}
	for _, id := range hostIDs {
		owners[id] = true
	}
	liveSessions := make([]SessionView, 0, len(h.sessions))
	for _, s := range h.sessions {
		if owners[s.HostID] && live[s.HostID] {
			v := SessionView{SessionMeta: s}
			if m, ok := metaByKey[s.HostID+"|"+s.ID]; ok {
				v.DisplayName = m.DisplayName
				v.Hidden = m.Hidden
			}
			liveSessions = append(liveSessions, v)
		}
	}
	// Stable order for the client: most recently active first, zero
	// timestamps last, id as a tiebreaker. Ranging over a Go map yields a
	// random order per request, which made the client list reshuffle on
	// every 5s poll (#249).
	sort.Slice(liveSessions, func(i, j int) bool {
		a, b := liveSessions[i], liveSessions[j]
		if a.LastSeenAt.IsZero() != b.LastSeenAt.IsZero() {
			return !a.LastSeenAt.IsZero()
		}
		if !a.LastSeenAt.Equal(b.LastSeenAt) {
			return a.LastSeenAt.After(b.LastSeenAt)
		}
		return a.ID < b.ID
	})
	// hostOnline lets the client tell "daemon offline" (empty list because no
	// host is connected) apart from "no sessions" (host connected, nothing
	// running): the two need very different empty states (#174).
	hostOnline := false
	for _, id := range hostIDs {
		if live[id] {
			hostOnline = true
			break
		}
	}
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"sessions": liveSessions, "hostOnline": hostOnline})
}

func (h *Hub) handleSessionMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w)
		return
	}
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[3] != "meta" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	sessionID := parts[2]
	var req struct {
		HostID      string  `json:"hostId"`
		DisplayName *string `json:"displayName"`
		Hidden      *bool   `json:"hidden"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HostID == "" {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	hostIDs, err := h.store.HostIDsForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	owns := false
	for _, hid := range hostIDs {
		if hid == req.HostID {
			owns = true
			break
		}
	}
	if !owns {
		writeError(w, http.StatusForbidden, "host not owned")
		return
	}

	meta := SessionClientMeta{SessionID: sessionID, HostID: req.HostID, UserID: u.ID}
	existing, err := h.store.GetSessionClientMeta(sessionID, req.HostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if existing != nil {
		meta.DisplayName = existing.DisplayName
		meta.Hidden = existing.Hidden
	}
	if req.DisplayName != nil {
		meta.DisplayName = *req.DisplayName
	}
	if req.Hidden != nil {
		meta.Hidden = *req.Hidden
	}
	if err := h.store.UpsertSessionClientMeta(meta); err != nil {
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId":   sessionID,
		"hostId":      req.HostID,
		"displayName": meta.DisplayName,
		"hidden":      meta.Hidden,
	})
}

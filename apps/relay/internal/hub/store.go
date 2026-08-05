package hub

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// loadStore restores persistent host and device records from disk so a relay
// restart does not invalidate registered hosts or paired devices.
func (h *Hub) loadStore() {
	if h.dataDir == "" {
		return
	}
	_ = os.MkdirAll(h.dataDir, 0o700)
	h.mu.Lock()
	defer h.mu.Unlock()
	if data, err := os.ReadFile(filepath.Join(h.dataDir, "hosts.json")); err == nil {
		var list []HostRecord
		if json.Unmarshal(data, &list) == nil {
			for _, rec := range list {
				h.hostRecords[rec.ID] = rec
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(h.dataDir, "devices.json")); err == nil {
		var list []Device
		if json.Unmarshal(data, &list) == nil {
			for _, d := range list {
				h.devices[d.ID] = d
			}
		}
	}
}

func (h *Hub) saveHosts() {
	if h.dataDir == "" {
		return
	}
	h.mu.Lock()
	list := make([]HostRecord, 0, len(h.hostRecords))
	for _, rec := range h.hostRecords {
		list = append(list, rec)
	}
	h.mu.Unlock()
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(h.dataDir, "hosts.json"), data, 0o600)
}

func (h *Hub) saveDevices() {
	if h.dataDir == "" {
		return
	}
	h.mu.Lock()
	list := make([]Device, 0, len(h.devices))
	for _, d := range h.devices {
		list = append(list, d)
	}
	h.mu.Unlock()
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(h.dataDir, "devices.json"), data, 0o600)
}

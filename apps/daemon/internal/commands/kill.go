package commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

// KillCmd triggers the daemon kill switch: stops all agent sessions and
// revokes all paired devices (local + cloud).
func KillCmd(base string) error {
	resp, err := cliutil.DaemonDo(nil, http.MethodPost, base+"/api/killswitch", nil)
	if err != nil {
		return fmt.Errorf("daemon not reachable at %s: %w", base, err)
	}
	defer resp.Body.Close()
	var out struct {
		Killed   bool `json:"killed"`
		Sessions int  `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	fmt.Println(t.T("kill_done", out.Sessions))
	return nil
}

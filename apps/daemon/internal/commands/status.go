package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

func StatusCmd(base string) error {
	resp, err := cliutil.DaemonDo(nil, http.MethodGet, base+"/api/status", nil)
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out bytes.Buffer
	if json.Indent(&out, body, "", "  ") != nil {
		out.Write(body)
	}
	fmt.Println(out.String())
	return nil
}

func SessionsCmd(base string) error {
	resp, err := cliutil.DaemonDo(nil, http.MethodGet, base+"/api/sessions", nil)
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out bytes.Buffer
	if json.Indent(&out, body, "", "  ") != nil {
		out.Write(body)
	}
	fmt.Println(out.String())
	return nil
}

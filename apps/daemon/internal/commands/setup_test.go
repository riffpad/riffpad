package commands

// Characterization tests for setupCmd (unit-file install/remove with all
// external effects stubbed) and extractLangFlag, added before the #282 split.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stubSystemctl records systemctlFn calls for the duration of a test.
func stubSystemctl(t *testing.T) *[]string {
	t.Helper()
	calls := &[]string{}
	old := systemctlFn
	systemctlFn = func(args ...string) ([]byte, error) {
		*calls = append(*calls, strings.Join(args, " "))
		return nil, nil
	}
	t.Cleanup(func() { systemctlFn = old })
	return calls
}

// TestSetupCmdRemove: `riffpad setup --remove` disables the service and
// removes the unit file, ignoring a missing file.
func TestSetupCmdRemove(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows uses the task-scheduler path")
	}
	calls := stubSystemctl(t)
	t.Setenv("HOME", t.TempDir())

	if err := SetupCmd([]string{"--remove"}, t.TempDir()); err != nil {
		t.Fatalf("setup --remove failed: %v", err)
	}
	if len(*calls) != 1 || (*calls)[0] != "disable --now riffpad.service" {
		t.Fatalf("unexpected systemctl calls: %v", *calls)
	}
}

// TestSetupCmdInstallsUnitFile: `riffpad setup` writes the unit file with the
// systemd-specifier-escaped PATH and runs daemon-reload + enable.
func TestSetupCmdInstallsUnitFile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	calls := stubSystemctl(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:50%off")
	exe := withFakeSelfExecutable(t)

	if err := SetupCmd(nil, t.TempDir()); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".config", "systemd", "user", "riffpad.service"))
	if err != nil {
		t.Fatal(err)
	}
	unit := string(raw)
	if !strings.Contains(unit, "ExecStart="+exe+" _daemon") {
		t.Fatalf("unit missing ExecStart: %q", unit)
	}
	if !strings.Contains(unit, "Environment=PATH=/usr/bin:50%%off") {
		t.Fatalf("unit PATH not escaped for systemd specifiers: %q", unit)
	}
	if len(*calls) != 2 || (*calls)[0] != "daemon-reload" || (*calls)[1] != "enable --now riffpad.service" {
		t.Fatalf("unexpected systemctl calls: %v", *calls)
	}
}

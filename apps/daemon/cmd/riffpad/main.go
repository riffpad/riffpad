// Command riffpad is the single Riffpad binary: user-facing CLI plus the
// hidden `_daemon` subcommand that runs the background daemon. Command
// implementations live in internal/commands; this file only parses --lang,
// dispatches, and exits.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
	"github.com/riffpad/riffpad/apps/daemon/internal/commands"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/console"
	"github.com/riffpad/riffpad/apps/daemon/internal/i18n"
	"github.com/riffpad/riffpad/apps/daemon/internal/version"
)

// t is the active language bundle, initialized in main from --lang.
var t = i18n.New(i18n.DefaultLang)

func main() {
	langFlag, args := extractLangFlag(os.Args[1:])
	t = i18n.New(i18n.Detect(langFlag))
	commands.SetBundle(t)
	console.SetBundle(t)
	os.Args = append([]string{os.Args[0]}, args...)
	// Corrupted state files are backed up and rebuilt automatically (#172);
	// warn instead of dying at startup. RunDaemon overrides this to also log.
	config.OnHeal = func(h config.Heal) {
		fmt.Fprintln(os.Stderr, t.T("config_file_healed", h.Path, h.Backup))
		if h.Kind == "keys" {
			fmt.Fprintln(os.Stderr, t.T("keys_regenerated_warn"))
		}
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] == "_daemon" {
		os.Exit(commands.RunDaemon(os.Args[2:]))
	}
	base := os.Getenv("RIFFPAD_URL")
	if base == "" {
		base = "http://127.0.0.1:8787"
	}
	dataDir := os.Getenv("RIFFPAD_DIR")
	if dataDir == "" {
		d, err := config.DefaultDataDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, t.T("resolve_data_dir", err))
			os.Exit(1)
		}
		dataDir = d
	}
	cliutil.SetDataDir(dataDir)

	var err error
	switch os.Args[1] {
	case "daemon":
		err = commands.DaemonCmd(os.Args[2:], base, dataDir)
	case "status":
		err = commands.StatusCmd(base)
	case "pair":
		err = commands.WithDaemon(func() error { return commands.PairCmd(base, os.Args[2:]) }, base, dataDir)
	case "sessions":
		err = commands.WithDaemon(func() error { return commands.SessionsCmd(base) }, base, dataDir)
	case "run":
		err = commands.WithDaemon(func() error { return commands.RunCmd(os.Args[2:], base) }, base, dataDir)
	case "logs":
		err = commands.LogsCmd(dataDir)
	case "attach":
		err = commands.WithDaemon(func() error { return commands.AttachCmd(base) }, base, dataDir)
	case "detach":
		err = commands.DetachCmd()
	case "auth":
		err = commands.AuthCmd(dataDir)
	case "login":
		err = commands.LoginCmd(os.Args[2:], dataDir)
	case "logout":
		err = commands.LogoutCmd(dataDir)
	case "relay":
		err = commands.RelayCmd(os.Args[2:], dataDir)
	case "setup":
		err = commands.SetupCmd(os.Args[2:], dataDir)
	case "kill":
		err = commands.KillCmd(base)
	case "update":
		err = commands.UpdateCmd(os.Args[2:], dataDir)
	case "version":
		fmt.Println("riffpad", version.Version)
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "riffpad:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, t.T("usage"))
}

// extractLangFlag pulls a global --lang/-lang value out of args so it works
// before any subcommand, regardless of position.
func extractLangFlag(args []string) (lang string, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--lang" || a == "-lang":
			if i+1 < len(args) {
				lang = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--lang="):
			lang = strings.TrimPrefix(a, "--lang=")
		case strings.HasPrefix(a, "-lang="):
			lang = strings.TrimPrefix(a, "-lang=")
		default:
			rest = append(rest, a)
		}
	}
	return lang, rest
}

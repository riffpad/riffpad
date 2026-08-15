package commands

import (
	"fmt"

	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
)

func LogsCmd(dataDir string) error {
	out, err := logging.Tail(dataDir, 200)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

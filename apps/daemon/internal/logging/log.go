package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// New creates a logger that writes to both the daemon log file and stderr.
func New(dataDir string) (*log.Logger, io.Closer, error) {
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, nil, err
	}
	path := filepath.Join(logDir, "daemon.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, err
	}
	logger := log.New(io.MultiWriter(f, os.Stderr), "", log.LstdFlags|log.LUTC)
	return logger, f, nil
}

// Tail returns the last n lines of the daemon log.
func Tail(dataDir string, n int) (string, error) {
	path := filepath.Join(dataDir, "logs", "daemon.log")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// Simple last-n-lines extraction.
	all := string(data)
	if n <= 0 {
		return all, nil
	}
	idx := []int{-1}
	for i, c := range all {
		if c == '\n' {
			idx = append(idx, i)
		}
	}
	start := 0
	if len(idx) > n {
		start = idx[len(idx)-n] + 1
	}
	return fmt.Sprintf("== %s ==\n%s", time.Now().Format(time.RFC3339), all[start:]), nil
}

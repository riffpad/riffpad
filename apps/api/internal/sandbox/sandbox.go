package sandbox

import (
	"context"
	"time"
)

type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
}

type ExecOptions struct {
	Timeout time.Duration
}

type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

type Sandbox interface {
	ID() string
	WorkspaceID() string
	ReadFile(ctx context.Context, path string) (string, error)
	WriteFile(ctx context.Context, path string, content string) error
	ListFiles(ctx context.Context, dir string) ([]FileInfo, error)
	Exec(ctx context.Context, command string, opts ExecOptions) (ExecResult, error)
	Destroy() error
}

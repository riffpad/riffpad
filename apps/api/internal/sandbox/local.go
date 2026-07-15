package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type LocalSandbox struct {
	id          string
	workspaceID string
	root        string
}

func NewLocal(workspaceID string) (*LocalSandbox, error) {
	root := filepath.Join(os.TempDir(), "riffpad-sandbox", workspaceID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create sandbox root: %w", err)
	}

	return &LocalSandbox{
		id:          workspaceID,
		workspaceID: workspaceID,
		root:        root,
	}, nil
}

func (s *LocalSandbox) ID() string          { return s.id }
func (s *LocalSandbox) WorkspaceID() string { return s.workspaceID }

func (s *LocalSandbox) resolve(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}

	abs := filepath.Clean(filepath.Join(s.root, path))
	// Ensure the resolved path is inside the sandbox root
	if !strings.HasPrefix(abs, s.root) || abs == s.root && path != "." {
		return "", fmt.Errorf("path %q escapes sandbox", path)
	}
	return abs, nil
}

func (s *LocalSandbox) ReadFile(ctx context.Context, path string) (string, error) {
	resolved, err := s.resolve(path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(data), nil
}

func (s *LocalSandbox) WriteFile(ctx context.Context, path string, content string) error {
	resolved, err := s.resolve(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(resolved, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (s *LocalSandbox) ListFiles(ctx context.Context, dir string) ([]FileInfo, error) {
	resolved, err := s.resolve(dir)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, fmt.Errorf("list directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(s.root, filepath.Join(resolved, entry.Name()))
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    rel,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
	}
	return files, nil
}

func (s *LocalSandbox) Exec(ctx context.Context, command string, opts ExecOptions) (ExecResult, error) {
	if err := validateCommand(command); err != nil {
		return ExecResult{}, err
	}

	shell := "/bin/sh"
	if path, err := exec.LookPath("bash"); err == nil {
		shell = path
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, "-c", command)
	cmd.Dir = s.root
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + s.root,
	}

	// Prevent network by default
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	output, err := cmd.CombinedOutput()
	result := ExecResult{
		Stdout:   string(output),
		Stderr:   "",
		ExitCode: 0,
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			result.ExitCode = -1
			result.Stderr = "command timed out"
		} else {
			result.ExitCode = -1
			result.Stderr = err.Error()
		}
	}

	return result, nil
}

func (s *LocalSandbox) Destroy() error {
	return os.RemoveAll(s.root)
}

func validateCommand(command string) error {
	lower := strings.ToLower(command)
	dangerous := []string{
		"rm -rf /",
		"mkfs.",
		":(){ :|:& };:",
		"dd if=/dev/zero",
		"> /dev/sda",
	}
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			return fmt.Errorf("dangerous command blocked: %q", command)
		}
	}
	return nil
}

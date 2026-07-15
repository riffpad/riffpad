package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalSandboxReadWrite(t *testing.T) {
	box, err := NewLocal("test-read-write")
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	defer box.Destroy()

	ctx := context.Background()
	if err := box.WriteFile(ctx, "hello.txt", "world"); err != nil {
		t.Fatalf("write file: %v", err)
	}

	content, err := box.ReadFile(ctx, "hello.txt")
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if content != "world" {
		t.Fatalf("expected 'world', got %q", content)
	}
}

func TestLocalSandboxListFiles(t *testing.T) {
	box, err := NewLocal("test-list")
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	defer box.Destroy()

	ctx := context.Background()
	_ = box.WriteFile(ctx, "a.txt", "a")
	_ = box.WriteFile(ctx, "dir/b.txt", "b")

	files, err := box.ListFiles(ctx, ".")
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(files))
	}
}

func TestLocalSandboxPathEscape(t *testing.T) {
	box, err := NewLocal("test-escape")
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	defer box.Destroy()

	ctx := context.Background()
	if _, err := box.ReadFile(ctx, "../outside.txt"); err == nil {
		t.Fatal("expected error for path escape")
	}
}

func TestLocalSandboxExec(t *testing.T) {
	box, err := NewLocal("test-exec")
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	defer box.Destroy()

	ctx := context.Background()
	result, err := box.Exec(ctx, "echo hello", ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Stdout != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %q", result.Stdout)
	}
}

func TestLocalSandboxDestroy(t *testing.T) {
	box, err := NewLocal("test-destroy")
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	root := box.root
	if err := box.Destroy(); err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("expected sandbox root to be removed")
	}
}

func TestValidateCommand(t *testing.T) {
	if err := validateCommand("rm -rf /"); err == nil {
		t.Fatal("expected dangerous command to be blocked")
	}
	if err := validateCommand("echo hello"); err != nil {
		t.Fatalf("expected safe command to pass, got %v", err)
	}
}

// expose root for tests
func (s *LocalSandbox) Root() string { return s.root }

func TestMain(m *testing.M) {
	// Clean up any leftover test sandboxes
	_ = os.RemoveAll(filepath.Join(os.TempDir(), "riffpad-sandbox"))
	os.Exit(m.Run())
}

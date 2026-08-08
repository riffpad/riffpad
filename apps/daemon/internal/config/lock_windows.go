//go:build windows

package config

import (
	"os"

	"golang.org/x/sys/windows"
)

func lockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0, 1, 0, ol,
	)
}

func unlockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
}

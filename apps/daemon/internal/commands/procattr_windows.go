//go:build windows

package commands

import "syscall"

func daemonProcAttr() *syscall.SysProcAttr {
	return nil
}

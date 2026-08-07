//go:build !windows

package main

import "syscall"

func daemonProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

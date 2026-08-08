//go:build !windows

package acp

import "syscall"

// setProcGroup configures the subprocess to run in its own process group (Unix).
func setProcGroup(attr *syscall.SysProcAttr) {
	attr.Setpgid = true
}

// killProcessGroup sends SIGTERM to the process group.
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGTERM)
}

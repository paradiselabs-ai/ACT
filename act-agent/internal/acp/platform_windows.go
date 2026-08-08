//go:build windows

package acp

import "syscall"

// setProcGroup is a no-op on Windows.
func setProcGroup(_ *syscall.SysProcAttr) {}

// killProcessGroup is unsupported on Windows; always returns a non-nil error.
func killProcessGroup(_ int) error {
	return syscall.EWINDOWS
}

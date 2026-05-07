//go:build windows

package runner

import "syscall"

// setProcGroup is a no-op on Windows — process groups work differently;
// individual Process.Kill() is used instead.
func setProcGroup(_ *syscall.SysProcAttr) {}

// killProcessGroup is unsupported on Windows; always returns a non-nil error
// so callers fall back to Process.Kill().
func killProcessGroup(_ int) error {
	return syscall.EWINDOWS
}

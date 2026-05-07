//go:build !windows

package runner

import "syscall"

// setProcGroup sets Setpgid so each runner gets its own process group,
// allowing the parent to kill the whole subtree by signaling the negative pgid.
func setProcGroup(attr *syscall.SysProcAttr) {
	attr.Setpgid = true
}

// killProcessGroup sends SIGTERM to the entire process group rooted at pid.
// Returns an error if the signal fails; callers should fall back to Process.Kill().
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGTERM)
}

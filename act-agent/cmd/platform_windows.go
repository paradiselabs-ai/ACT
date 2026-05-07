//go:build windows

package cmd

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// terminalSize returns 0,0 on Windows — the TUI auto-resizes via WindowSizeMsg.
func terminalSize(_ *os.File) (int, int) {
	return 0, 0
}

// notifyOSSignals subscribes sigCh to SIGTERM on Windows (SIGHUP unavailable).
func notifyOSSignals(sigCh chan os.Signal) {
	signal.Notify(sigCh, syscall.SIGTERM)
}

// execCLIProc runs bin+args as a child process and exits with its exit code.
// Windows has no execve, so we spawn a child and forward the exit code.
func execCLIProc(bin string, args []string) error {
	cmd := exec.Command(bin, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	os.Exit(0)
	return nil
}

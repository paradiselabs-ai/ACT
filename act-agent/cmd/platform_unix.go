//go:build !windows

package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

// terminalSize returns (cols, rows) via TIOCGWINSZ. Returns 0,0 on failure.
func terminalSize(f *os.File) (int, int) {
	ws := struct {
		Row, Col, Xpixel, Ypixel uint16
	}{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0, 0
	}
	return int(ws.Col), int(ws.Row)
}

// notifyOSSignals subscribes sigCh to SIGHUP and SIGTERM (Unix).
func notifyOSSignals(sigCh chan os.Signal) {
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)
}

// execCLIProc replaces the current process with bin+args via execve.
// On Unix this is a true exec-replace; on Windows it falls back to
// running a child process.
func execCLIProc(bin string, args []string) error {
	return syscall.Exec(bin, args, os.Environ())
}

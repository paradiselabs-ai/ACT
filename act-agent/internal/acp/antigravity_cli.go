package acp

import (
	"os"
	"path/filepath"
	"strings"
)

// antigravityCLIDefaults returns the spawn argv for the bundled agy ACP shim.
//
// env is cfg.Env from the ACPConfig that withTier1ShimPath already enriched —
// binDir (the act-agent install directory) is the first entry in env["PATH"].
// We read it from there directly rather than re-deriving it from os.Executable.
//
// agy itself is NOT bundled — users install it separately:
//
//	npm i -g @google/antigravity-cli
func antigravityCLIDefaults(env map[string]string) (command string, args []string) {
	return "node", []string{agyShimPath(env)}
}

// agyShimPath resolves the absolute path of agy-acp.mjs.
//
// Two strategies, tried in order:
//  1. Read binDir from cfg.Env["PATH"] (set by withTier1ShimPath). Works when
//     os.Executable resolved the real binary path without symlinks.
//  2. Re-derive binDir from os.Executable with filepath.EvalSymlinks applied.
//     Handles the common case where the user runs act-agent via a symlink in
//     ~/.local/bin or /opt/homebrew/bin — os.Executable returns the symlink
//     path, so strategy 1 looks in the wrong directory.
func agyShimPath(env map[string]string) string {
	// Strategy 1: use binDir from withTier1ShimPath's PATH.
	if p := env["PATH"]; p != "" {
		binDir, _, _ := strings.Cut(p, string(os.PathListSeparator))
		if binDir != "" {
			candidate := filepath.Join(binDir, "agy-acp.mjs")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	// Strategy 2: resolve the real binary location (following symlinks).
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		candidate := filepath.Join(filepath.Dir(exe), "agy-acp.mjs")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "agy-acp.mjs"
}

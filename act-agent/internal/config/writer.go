package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteAgentBackend mutates ~/.act.json (or .opencode.json fallback) to set
// the `backend` field for a given agent role. Used by slash commands and CLI
// subcommands so users can change backends without manually editing JSON.
//
// Atomic: writes to a temp file in the same directory and renames over the
// original. Preserves all other fields by decoding to a generic map.
func WriteAgentBackend(role, backend string) error {
	path, err := userConfigPath()
	if err != nil {
		return err
	}

	// Read existing config (or start empty)
	raw, err := os.ReadFile(path)
	var data map[string]any
	if err == nil {
		if jerr := json.Unmarshal(raw, &data); jerr != nil {
			return fmt.Errorf("parse %s: %w", path, jerr)
		}
	} else if os.IsNotExist(err) {
		data = map[string]any{}
	} else {
		return fmt.Errorf("read %s: %w", path, err)
	}

	// Ensure agents.<role> exists
	agentsRaw, ok := data["agents"].(map[string]any)
	if !ok {
		agentsRaw = map[string]any{}
		data["agents"] = agentsRaw
	}
	roleRaw, ok := agentsRaw[role].(map[string]any)
	if !ok {
		roleRaw = map[string]any{}
		agentsRaw[role] = roleRaw
	}
	roleRaw["backend"] = backend

	// Marshal pretty
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	out = append(out, '\n')

	// Atomic write: temp file + rename
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".act.json.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// userConfigPath returns the path to the user-level ACT config. Prefers
// ~/.act.json; falls back to ~/.opencode.json if only the legacy file exists.
// Always returns a path; the caller decides whether to read or create.
func userConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	actPath := filepath.Join(home, ".act.json")
	if _, err := os.Stat(actPath); err == nil {
		return actPath, nil
	}
	legacyPath := filepath.Join(home, ".opencode.json")
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath, nil
	}
	return actPath, nil // will be created
}

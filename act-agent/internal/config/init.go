package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// InitFlagFilename is the name of the file that indicates whether the project has been initialized
	InitFlagFilename = "init"
)

// ProjectInitFlag represents the initialization status for a project directory
type ProjectInitFlag struct {
	Initialized bool `json:"initialized"`
}

// ShouldShowInitDialog checks if the initialization dialog should be shown for the current directory
func ShouldShowInitDialog() (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("config not loaded")
	}

	// Create the flag file path
	flagFilePath := filepath.Join(cfg.Data.Directory, InitFlagFilename)

	// Check if the flag file exists
	_, err := os.Stat(flagFilePath)
	if err == nil {
		// File exists, don't show the dialog
		return false, nil
	}

	// If the error is not "file not found", return the error
	if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to check init flag file: %w", err)
	}

	// File doesn't exist, show the dialog
	return true, nil
}

// IsFirstRun checks if this is the first time ACT is being used on this machine.
// Returns true if neither .act/ nor .opencode/ exists in the user's home directory.
func IsFirstRun() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	// Check for .act/ (new name) or .opencode/ (legacy)
	if _, err := os.Stat(filepath.Join(home, ".act")); err == nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(home, ".opencode")); err == nil {
		return false
	}
	return true
}

// HasConfigFile checks if an ACT config file (.act.json or .opencode.json) exists.
func HasConfigFile() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(home, ".act.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(home, ".opencode.json")); err == nil {
		return true
	}
	return false
}

// MarkProjectInitialized marks the current project as initialized
func MarkProjectInitialized() error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}
	// Create the flag file path
	flagFilePath := filepath.Join(cfg.Data.Directory, InitFlagFilename)

	// Create an empty file to mark the project as initialized
	file, err := os.Create(flagFilePath)
	if err != nil {
		return fmt.Errorf("failed to create init flag file: %w", err)
	}
	defer file.Close()

	return nil
}


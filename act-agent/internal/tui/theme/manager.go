package theme

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

// Manager handles theme registration, selection, and retrieval.
// It maintains a registry of available themes and tracks the currently active theme.
type Manager struct {
	themes      map[string]Theme
	currentName string
	mu          sync.RWMutex
}

// Global instance of the theme manager
var globalManager = &Manager{
	themes:      make(map[string]Theme),
	currentName: "",
}

// registerOnce guards ensureRegistered. Theme registration used to happen
// in nine package init() functions; any Go binary that merely linked this
// package (including every test binary in packages that transitively import
// it) executed them before main/testing.Main — and on some Windows hosts
// that init path blocked indefinitely, hanging the entire test suite at
// startup (audit P4-hang). Registration is now lazy: the first public API
// call triggers it once.
var registerOnce sync.Once

// RegisterBuiltins registers all built-in themes exactly once. Safe to call
// directly (app.New does, to keep startup deterministic); every public
// accessor also calls it via registerOnce.
func RegisterBuiltins() {
	registerOnce.Do(func() {
		RegisterTheme("act", NewACTTheme())
		RegisterTheme("opencode", NewACTTheme()) // backward-compat alias
		RegisterTheme("catppuccin", NewCatppuccinTheme())
		RegisterTheme("dracula", NewDraculaTheme())
		RegisterTheme("flexoki", NewFlexokiTheme())
		RegisterTheme("gruvbox", NewGruvboxTheme())
		RegisterTheme("monokai", NewMonokaiProTheme())
		RegisterTheme("onedark", NewOneDarkTheme())
		RegisterTheme("tokyonight", NewTokyoNightTheme())
		RegisterTheme("tron", NewTronTheme())
	})
}

// RegisterTheme adds a new theme to the registry.
// If this is the first theme registered, it becomes the default.
func RegisterTheme(name string, theme Theme) {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	globalManager.themes[name] = theme

	// If this is the first theme, make it the default
	if globalManager.currentName == "" {
		globalManager.currentName = name
	}
}

// SetTheme changes the active theme to the one with the specified name.
// Returns an error if the theme doesn't exist.
func SetTheme(name string) error {
	RegisterBuiltins()
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	delete(styles.Registry, "charm")
	if _, exists := globalManager.themes[name]; !exists {
		return fmt.Errorf("theme '%s' not found", name)
	}

	globalManager.currentName = name

	// Update the config file using viper
	if err := updateConfigTheme(name); err != nil {
		// Log the error but don't fail the theme change
		logging.Warn("Warning: Failed to update config file with new theme", "err", err)
	}

	return nil
}

// CurrentTheme returns the currently active theme.
// If no theme is set, it returns nil.
func CurrentTheme() Theme {
	RegisterBuiltins()
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	if globalManager.currentName == "" {
		return nil
	}

	return globalManager.themes[globalManager.currentName]
}

// CurrentThemeName returns the name of the currently active theme.
func CurrentThemeName() string {
	RegisterBuiltins()
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	return globalManager.currentName
}

// AvailableThemes returns a list of all registered theme names.
func AvailableThemes() []string {
	RegisterBuiltins()
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	names := make([]string, 0, len(globalManager.themes))
	for name := range globalManager.themes {
		names = append(names, name)
	}
	slices.SortFunc(names, func(a, b string) int {
		if a == "act" {
			return -1
		} else if b == "act" {
			return 1
		}
		return strings.Compare(a, b)
	})
	return names
}

// GetTheme returns a specific theme by name.
// Returns nil if the theme doesn't exist.
func GetTheme(name string) Theme {
	RegisterBuiltins()
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	return globalManager.themes[name]
}

// updateConfigTheme updates the theme setting in the configuration file
func updateConfigTheme(themeName string) error {
	// Use the config package to update the theme
	return config.UpdateTheme(themeName)
}

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// stripCommentKeys returns raw with every map key whose name starts with
// "_comment" removed at any nesting depth. JSON has no native comment syntax,
// so .act.example.json (and any ~/.act.json a user creates by copying it)
// carries documentation as "_comment" / "_comment_<topic>" string values
// alongside real fields. mapstructure cannot fit a string into a typed struct
// field, so without this strip step the agents map fails to unmarshal when
// the example documents fields like `"agents": { "_comment": "...", "planner": {...} }`.
func stripCommentKeys(raw []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return json.Marshal(stripWalk(v))
}

func stripWalk(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, vv := range x {
			if strings.HasPrefix(k, "_comment") {
				continue
			}
			out[k] = stripWalk(vv)
		}
		return out
	case []interface{}:
		for i := range x {
			x[i] = stripWalk(x[i])
		}
		return x
	}
	return v
}

// readSanitizedGlobalConfig finds the user's global config file using the same
// path precedence viper's AddConfigPath chain would, strips _comment keys,
// and feeds the result into the global viper instance. Returns nil with no
// side-effects when no config file exists (defaults apply).
func readSanitizedGlobalConfig() error {
	for _, p := range globalConfigSearchPaths() {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		cleaned, err := stripCommentKeys(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", p, err)
		}
		viper.SetConfigType("json")
		viper.SetConfigFile(p)
		return viper.ReadConfig(bytes.NewReader(cleaned))
	}
	return nil
}

// readSanitizedLocalConfig mirrors the global function for the per-project
// .act.json that mergeLocalConfig consumes. It returns a fresh *viper.Viper
// populated from sanitized bytes, or nil if no local config is present.
func readSanitizedLocalConfig(workingDir string) *viper.Viper {
	candidates := []string{
		filepath.Join(workingDir, fmt.Sprintf(".%s.json", appName)),
		filepath.Join(workingDir, ".opencode.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		cleaned, err := stripCommentKeys(data)
		if err != nil {
			continue
		}
		local := viper.New()
		local.SetConfigType("json")
		if err := local.ReadConfig(bytes.NewReader(cleaned)); err != nil {
			continue
		}
		return local
	}
	return nil
}

func globalConfigSearchPaths() []string {
	home, _ := os.UserHomeDir()
	name := fmt.Sprintf(".%s.json", appName)
	legacy := ".opencode.json"
	paths := []string{}
	if home != "" {
		paths = append(paths, filepath.Join(home, name))
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, appName, name))
	}
	if home != "" {
		paths = append(paths,
			filepath.Join(home, ".config", appName, name),
			filepath.Join(home, legacy),
		)
	}
	return paths
}

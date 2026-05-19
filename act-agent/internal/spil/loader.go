// Package spil loads SPIL-formatted prompt files at startup and exposes them
// to the rest of the act-agent binary.
//
// SPIL files live in act-agent/prompts/*.spil and are embedded via go:embed so
// the binary is self-contained. The loader parses each file once into a
// section-keyed map; callers fetch either the full assembled body or a single
// section by name.
//
// This is ACT's Phase 1 of SPIL integration per docs/Vault/.../SPIL.md:
// role-identity prompts move out of Go string constants into .spil files. The
// orchestrator stays in Go; only the prompt content moves.
package spil

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"sync"
)

// File holds a parsed SPIL document.
type File struct {
	Name     string            // filename without extension (e.g. "planner")
	Body     string            // full document text as authored
	Order    []string          // section names in document order
	Sections map[string]string // section name (without @) → section body (includes the leading @keyword line)
}

// Section returns the body of one named section. Returns the empty string if
// the section is not present; callers should check ok before using.
func (f *File) Section(name string) (string, bool) {
	if f == nil {
		return "", false
	}
	body, ok := f.Sections[strings.TrimPrefix(name, "@")]
	return body, ok
}

// Manifest returns a manifest string: one line per section, formatted as
// "@name — <first non-empty line of section body, trimmed>".
//
// Useful for Phase 2 lazy injection — the caller can paste the manifest into a
// system prompt and use Section() to fetch bodies on demand.
func (f *File) Manifest() string {
	if f == nil {
		return ""
	}
	var lines []string
	for _, name := range f.Order {
		body := f.Sections[name]
		hint := firstContentLine(body)
		lines = append(lines, fmt.Sprintf("@%s — %s", name, hint))
	}
	return strings.Join(lines, "\n")
}

func firstContentLine(body string) string {
	for _, ln := range strings.Split(body, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "@") || strings.HasPrefix(t, "//") {
			continue
		}
		// Strip leading list-item dashes for cleaner hints
		t = strings.TrimPrefix(t, "- ")
		// Strip surrounding quotes
		if len(t) >= 2 && t[0] == '"' && t[len(t)-1] == '"' {
			t = t[1 : len(t)-1]
		}
		// Truncate to keep manifests cheap
		if len(t) > 100 {
			t = t[:97] + "..."
		}
		return t
	}
	return "(section body)"
}

//go:embed all:prompts
var promptsFS embed.FS

var (
	loadOnce sync.Once
	files    map[string]*File
	loadErr  error
)

// Load reads every act-agent/prompts/*.spil file once and caches them.
// Called automatically on first access; safe for concurrent callers.
func Load() error {
	loadOnce.Do(func() {
		files = make(map[string]*File)
		entries, err := fs.ReadDir(promptsFS, "prompts")
		if err != nil {
			loadErr = fmt.Errorf("spil: read prompts dir: %w", err)
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".spil") {
				continue
			}
			data, err := promptsFS.ReadFile(path.Join("prompts", e.Name()))
			if err != nil {
				loadErr = fmt.Errorf("spil: read %s: %w", e.Name(), err)
				return
			}
			name := strings.TrimSuffix(e.Name(), ".spil")
			files[name] = parse(name, string(data))
		}
	})
	return loadErr
}

// Get returns the parsed SPIL file by name (e.g. "planner"). Triggers load on
// first call. Returns (nil, false) if no such file exists.
func Get(name string) (*File, bool) {
	if err := Load(); err != nil {
		// Embedded reads fail only on programmer error; surface via empty result
		return nil, false
	}
	f, ok := files[name]
	return f, ok
}

// MustGet returns the file or panics. Use only in package init or test code
// where the file is known to exist.
func MustGet(name string) *File {
	f, ok := Get(name)
	if !ok {
		panic(fmt.Sprintf("spil: required file %q not found in embedded prompts", name))
	}
	return f
}

// Body returns the full assembled body of a role's SPIL file, or empty if
// missing. Convenience for callers replacing inline Go string constants.
func Body(name string) string {
	f, ok := Get(name)
	if !ok {
		return ""
	}
	return f.Body
}

// List returns the names of all loaded SPIL files (for diagnostics / tests).
func List() []string {
	_ = Load()
	out := make([]string, 0, len(files))
	for n := range files {
		out = append(out, n)
	}
	return out
}

// ---- parser ----

// headerRe matches @keyword section headers at column 0. Both styles are
// accepted: `@keyword "value"` (inline) and `@keyword:` (block).
var headerRe = regexp.MustCompile(`^@(\w+)(?:\s+.*|:?)\s*$`)

func parse(name, text string) *File {
	f := &File{
		Name:     name,
		Body:     text,
		Sections: make(map[string]string),
	}

	lines := strings.Split(text, "\n")
	var curName string
	var curLines []string

	flush := func() {
		if curName == "" {
			return
		}
		// trim trailing blank lines
		for len(curLines) > 0 && strings.TrimSpace(curLines[len(curLines)-1]) == "" {
			curLines = curLines[:len(curLines)-1]
		}
		f.Order = append(f.Order, curName)
		f.Sections[curName] = strings.Join(curLines, "\n")
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "@") {
			if m := headerRe.FindStringSubmatch(line); m != nil {
				flush()
				curName = m[1]
				curLines = []string{line}
				continue
			}
		}
		if curName != "" {
			curLines = append(curLines, line)
		}
		// lines before the first @section are dropped from per-section indexing
		// but remain in f.Body for full-body callers
	}
	flush()
	return f
}

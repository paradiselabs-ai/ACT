package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContextFromPaths(t *testing.T) {
	// Not parallel: config.Load and getContextFromPaths use package-level
	// globals (cfg, contextLoaded, contextContent). Running in parallel
	// with another test that touches these would race.

	// Isolate from the developer's real ~/.act.json. Without this, the
	// global config loader walks $HOME and pulls in whatever providers/
	// agents happen to be configured, then Validate fires against fields
	// the test never asked for — flake-by-environment.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// The Load singleton (cfg) may have been populated by an earlier test
	// in the same binary. Reset before each call so contextPaths overrides
	// below take effect on a fresh load.
	config.ResetForTests()
	InvalidateContextCache()

	_, err := config.Load(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg := config.Get()
	cfg.WorkingDir = tmpDir
	cfg.ContextPaths = []string{
		"file.txt",
		"directory/",
	}
	testFiles := []string{
		"file.txt",
		"directory/file_a.txt",
		"directory/file_b.txt",
		"directory/file_c.txt",
	}

	createTestFiles(t, tmpDir, testFiles)

	context := getContextFromPaths()
	expectedContext := fmt.Sprintf("# From:%s/file.txt\nfile.txt: test content\n# From:%s/directory/file_a.txt\ndirectory/file_a.txt: test content\n# From:%s/directory/file_b.txt\ndirectory/file_b.txt: test content\n# From:%s/directory/file_c.txt\ndirectory/file_c.txt: test content", tmpDir, tmpDir, tmpDir, tmpDir)
	assert.Equal(t, expectedContext, context)
}

// TestProcessContextPaths_DeterministicOrdering — audit Fix 14 (entry
// 8.2). Two consecutive calls on the same files must produce byte-
// identical output. Pre-fix the goroutine fan-out + channel collection
// could produce different orderings → different bytes → provider-side
// prompt-cache breakpoint hit a different key each turn.
func TestProcessContextPaths_DeterministicOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{
		"a/one.txt",
		"a/two.txt",
		"a/three.txt",
		"b/x.txt",
		"b/y.txt",
		"root.txt",
	}
	createTestFiles(t, tmpDir, files)

	paths := []string{"root.txt", "a/", "b/"}
	first := processContextPaths(tmpDir, paths)
	if first == "" {
		t.Fatalf("first run produced empty output")
	}
	for i := 0; i < 8; i++ {
		next := processContextPaths(tmpDir, paths)
		if next != first {
			t.Fatalf("run %d produced different bytes:\nfirst:\n%s\n\nrun:\n%s", i+2, first, next)
		}
	}
}

// TestGetContextFromPaths_HashSkipReusesContent — audit Fix 14 (entry
// 2.4). After Invalidate + rebuild on unchanged files, the returned
// string must be byte-identical to the prior cached value (the cache
// retains contextContent + contextHash and reuses them when the
// rebuild's hash matches).
func TestGetContextFromPaths_HashSkipReusesContent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	config.ResetForTests()
	InvalidateContextCache()
	contextHash = "" // also reset hash since the prior test may have set it
	contextContent = ""

	_, err := config.Load(tmpDir, false)
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	cfg := config.Get()
	cfg.WorkingDir = tmpDir
	cfg.ContextPaths = []string{"hashtest.txt"}
	createTestFiles(t, tmpDir, []string{"hashtest.txt"})

	first := getContextFromPaths()
	if first == "" {
		t.Fatalf("first call produced empty")
	}
	hashAfterFirst := contextHash

	// Invalidate but don't change the file. Rebuild should detect
	// identical content via hash and reuse the cached string.
	InvalidateContextCache()
	second := getContextFromPaths()
	if second != first {
		t.Errorf("hash-skip path failed — second call must return identical bytes\nfirst:\n%s\n\nsecond:\n%s", first, second)
	}
	if contextHash != hashAfterFirst {
		t.Errorf("contextHash must be stable across no-change invalidate cycles; got %q vs %q", contextHash, hashAfterFirst)
	}

	// Now mutate the file. Rebuild MUST produce different content + new hash.
	if err := os.WriteFile(filepath.Join(tmpDir, "hashtest.txt"), []byte("hashtest.txt: CHANGED content"), 0644); err != nil {
		t.Fatalf("file rewrite: %v", err)
	}
	InvalidateContextCache()
	third := getContextFromPaths()
	if third == first {
		t.Errorf("after file change, rebuild must produce new content; got identical bytes")
	}
	if contextHash == hashAfterFirst {
		t.Errorf("after file change, contextHash must update; still %q", contextHash)
	}
}

func createTestFiles(t *testing.T, tmpDir string, testFiles []string) {
	t.Helper()
	for _, path := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if path[len(path)-1] == '/' {
			err := os.MkdirAll(fullPath, 0755)
			require.NoError(t, err)
		} else {
			dir := filepath.Dir(fullPath)
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
			err = os.WriteFile(fullPath, []byte(path+": test content"), 0644)
			require.NoError(t, err)
		}
	}
}

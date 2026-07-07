package prompt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

func GetAgentPrompt(agentName config.AgentName, provider models.ModelProvider) string {
	basePrompt := ""
	switch agentName {
	// Tier 1 — Interactive (NesTTY window)
	case config.RolePlanner:
		basePrompt = PlannerPrompt(provider)
	case config.RoleObserver:
		basePrompt = ObserverPrompt(provider)
	case config.RoleAssurance:
		basePrompt = AssurancePrompt(provider)
	case config.RoleQASynthesizer:
		basePrompt = QASynthesizerPrompt(provider)
	// Tier 2 — The Swarm (headless)
	case config.RoleDeveloper:
		basePrompt = DeveloperPrompt(provider)
	case config.RoleFrontendDev:
		basePrompt = FrontendDevPrompt(provider)
	case config.RoleBackendDev:
		basePrompt = BackendDevPrompt(provider)
	case config.RoleQAEngineer:
		basePrompt = QAEngineerPrompt(provider)
	case config.RoleResearcher:
		basePrompt = ResearcherPrompt(provider)
	default:
		// Utility agents (title/summarizer) legitimately use the generic
		// developer prompt. Any OTHER unrecognized name means a role string
		// never got a case above — warn loudly rather than letting it silently
		// masquerade as a developer (the role-swap anti-pattern).
		if agentName != config.AgentTitle && agentName != config.AgentSummarizer {
			logging.Warn("prompt_role_unrecognized",
				"agent_name", string(agentName),
				"action", "falling back to developer prompt — this role has no case in GetAgentPrompt",
			)
		}
		basePrompt = DeveloperPrompt(provider)
	}

	// All roles get project-specific context (CLAUDE.md, ACT.md, etc.)
	if agentName != config.AgentTitle && agentName != config.AgentSummarizer {
		contextContent := getContextFromPaths()
		logging.Debug("Context content", "Context", contextContent)
		if contextContent != "" {
			return fmt.Sprintf("%s\n\n# Project-Specific Context\nFollow the instructions in the context below:\n%s", basePrompt, contextContent)
		}
	}
	return basePrompt
}

var (
	contextMu      sync.Mutex
	contextLoaded  bool
	contextContent string
	// contextHash is the SHA-256 of the most-recently-built contextContent.
	// On rebuild (post-InvalidateContextCache), we recompute the hash and
	// reuse the prior contextContent string if the hash matches — keeps the
	// provider-side prompt-cache breakpoint (Anthropic ephemeral cache,
	// anthropic.go:190) stable when InvalidateContextCache fires but the
	// files didn't actually change. Audit Fix 14 (entry 2.4).
	contextHash string
)

func getContextFromPaths() string {
	contextMu.Lock()
	defer contextMu.Unlock()
	if contextLoaded {
		return contextContent
	}
	cfg := config.Get()
	rebuilt := processContextPaths(cfg.WorkingDir, cfg.ContextPaths)
	rebuiltHash := hashString(rebuilt)
	if contextHash != "" && rebuiltHash == contextHash {
		// Content unchanged from last cached state — discard the new
		// string and reuse the prior one. Same bytes either way; reusing
		// keeps any downstream identity-equality (provider cache keys) stable.
		contextLoaded = true
		return contextContent
	}
	contextContent = rebuilt
	contextHash = rebuiltHash
	contextLoaded = true
	return contextContent
}

// InvalidateContextCache clears the loaded-flag so the next call to
// getContextFromPaths re-reads every file. NOT cleared: contextContent and
// contextHash, so the rebuild can compare against them and reuse the prior
// string if the files happen to be unchanged (audit Fix 14, entry 2.4). The
// orchestrator calls this after writing AGENTS.md from a fresh PROJECT_BRIEF
// so Tier 1 agents whose system messages were baked at TUI startup before
// AGENTS.md existed can pick up the new content via RebindSystemPrompt.
func InvalidateContextCache() {
	contextMu.Lock()
	defer contextMu.Unlock()
	contextLoaded = false
}

// hashString returns a hex-encoded SHA-256 of s. Used by the cache to skip
// unnecessary content-string churn when a rebuild produces identical bytes.
func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// processContextPaths reads every path (file or directory) under workDir
// and assembles their contents into one string. Output is deterministic:
// (1) paths visited in argument order, (2) directory walks sorted by
// file path, (3) duplicates suppressed case-insensitively. Determinism
// matters because the assembled string lands in the system prompt and
// hits the provider-side prompt-cache; non-deterministic ordering poisons
// the cache key and forces a full re-process every turn. Audit Fix 14
// (entry 8.2) — used to fan out across goroutines with channel-order
// collection, producing different bytes for the same files on different
// runs.
//
// The parallelism that was here didn't buy much: context paths is
// typically 2-3 entries (per CLAUDE.md "defaultContextPaths reduced to
// ['ACT.md', 'ACT.local.md']"). Sequential read is simpler AND
// deterministic by construction.
func processContextPaths(workDir string, paths []string) string {
	processed := make(map[string]bool) // case-insensitive seen-set
	var parts []string

	consume := func(fullPath string) {
		key := strings.ToLower(fullPath)
		if processed[key] {
			return
		}
		processed[key] = true
		if result := processFile(fullPath); result != "" {
			parts = append(parts, result)
		}
	}

	for _, p := range paths {
		if strings.HasSuffix(p, "/") {
			root := filepath.Join(workDir, p)
			// Collect file paths first, sort, then read in order — keeps
			// directory contributions deterministic even if the OS returns
			// filesystem-order entries (typical on macOS/Linux).
			var files []string
			_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				files = append(files, path)
				return nil
			})
			sort.Strings(files)
			for _, f := range files {
				consume(f)
			}
			continue
		}
		consume(filepath.Join(workDir, p))
	}

	return strings.Join(parts, "\n")
}

func processFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return "# From:" + filePath + "\n" + string(content)
}

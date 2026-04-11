package prompt

import (
	"fmt"
	"os"
	"path/filepath"
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
		// Fallback to developer prompt — "coder" is an OpenCode internal, not an ACT role
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
	onceContext    sync.Once
	contextContent string
)

func getContextFromPaths() string {
	onceContext.Do(func() {
		var (
			cfg          = config.Get()
			workDir      = cfg.WorkingDir
			contextPaths = cfg.ContextPaths
		)

		contextContent = processContextPaths(workDir, contextPaths)
	})

	return contextContent
}

func processContextPaths(workDir string, paths []string) string {
	var (
		wg       sync.WaitGroup
		resultCh = make(chan string)
	)

	// Track processed files to avoid duplicates
	processedFiles := make(map[string]bool)
	var processedMutex sync.Mutex

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			if strings.HasSuffix(p, "/") {
				filepath.WalkDir(filepath.Join(workDir, p), func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !d.IsDir() {
						// Check if we've already processed this file (case-insensitive)
						processedMutex.Lock()
						lowerPath := strings.ToLower(path)
						if !processedFiles[lowerPath] {
							processedFiles[lowerPath] = true
							processedMutex.Unlock()

							if result := processFile(path); result != "" {
								resultCh <- result
							}
						} else {
							processedMutex.Unlock()
						}
					}
					return nil
				})
			} else {
				fullPath := filepath.Join(workDir, p)

				// Check if we've already processed this file (case-insensitive)
				processedMutex.Lock()
				lowerPath := strings.ToLower(fullPath)
				if !processedFiles[lowerPath] {
					processedFiles[lowerPath] = true
					processedMutex.Unlock()

					result := processFile(fullPath)
					if result != "" {
						resultCh <- result
					}
				} else {
					processedMutex.Unlock()
				}
			}
		}(path)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make([]string, 0)
	for result := range resultCh {
		results = append(results, result)
	}

	return strings.Join(results, "\n")
}

func processFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return "# From:" + filePath + "\n" + string(content)
}

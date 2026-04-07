package runner

// DefaultCapabilities is the per-role capability tag list used by the Spawner
// when starting a swarm Runner. The ACT server uses these tags to route tasks
// (`requiredCapabilities` on a task is matched against an agent's declared
// capabilities). Users can override per-Runner via the
// ACT_CAPABILITIES_<ROLE> environment variable.
//
// These defaults are intentionally broad — they describe what each role is
// equipped to handle, not a strict allowlist. The Planner picks the most
// specific match.
var DefaultCapabilities = map[string][]string{
	"developer": {
		"general", "typescript", "javascript", "python", "go", "rust",
		"bash", "git", "shell", "scripting", "cli", "refactor",
	},
	"frontend_dev": {
		"frontend", "ui", "ux", "react", "vue", "svelte", "html", "css",
		"tailwind", "typescript", "javascript", "accessibility", "a11y",
		"responsive", "components",
	},
	"backend_dev": {
		"backend", "api", "rest", "graphql", "database", "sql", "postgres",
		"sqlite", "mysql", "redis", "auth", "authentication", "authorization",
		"middleware", "python", "go", "node", "rust",
	},
	"qa_engineer": {
		"testing", "qa", "unit-tests", "integration-tests", "e2e",
		"pytest", "jest", "vitest", "mocha", "playwright", "cypress",
		"coverage", "fixtures", "mocking", "tdd",
	},
	"researcher": {
		"research", "analysis", "documentation", "investigation",
		"survey", "comparison", "benchmarking", "literature-review",
		"requirements", "discovery",
	},
}

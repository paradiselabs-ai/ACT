package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
)

// getEnvironmentInfo returns the `<env>` block injected into every ACT role
// prompt. Deliberately excludes an `ls .` dump of cwd — that balloons the
// prompt in populated repos and is death on free-tier TPM caps. Agents can
// list files on demand with the LS tool when they need to.
func getEnvironmentInfo() string {
	cwd := config.WorkingDirectory()
	isGit := isGitRepo(cwd)
	platform := runtime.GOOS
	// ISO 8601 with explicit UTC timezone — unambiguous for absolute-time
	// reasoning across locales. Audit Fix 13e (entry 8.1): the prior
	// `1/2/2006` US format was borderline confusing for deadlines / "is
	// this recent?" reasoning, and lacked timezone entirely.
	date := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`<env>
cwd: %s
git: %s
platform: %s
date: %s
</env>`, cwd, boolToYesNo(isGit), platform, date)
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// lspInformation returns the LSP diagnostics blurb appended to prompts when
// at least one non-disabled LSP client is configured. Empty string otherwise.
func lspInformation() string {
	cfg := config.Get()
	hasLSP := false
	for _, v := range cfg.LSP {
		if !v.Disabled {
			hasLSP = true
			break
		}
	}
	if !hasLSP {
		return ""
	}
	return `# LSP Information
Tools that support it will also include useful diagnostics such as linting and typechecking.
- These diagnostics will be automatically enabled when you run the tool, and will be displayed in the output at the bottom within the <file_diagnostics></file_diagnostics> and <project_diagnostics></project_diagnostics> tags.
- Take necessary actions to fix the issues.
- You should ignore diagnostics of files that you did not change or are not related or caused by your changes unless the user explicitly asks you to fix them.
`
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// Shared prompt building blocks for all ACT role prompts.
// Each role prompt imports these sections to avoid duplication.

// actCLICommands returns the ACT CLI commands relevant to a specific role.
func actCLICommands(role string) string {
	var base string
	switch role {
	case "planner":
		base = `## ACT CLI Commands (available to you)
You are an in-process Tier 1 role. You speak by writing plain text in your reply — do NOT shell out to send messages. CREATE_TASK and PROJECT_BRIEF are markers in your reply text, not shell commands.
- act-agent context --project <name>    Load full project context (tasks, agents, brief)
- act-agent graph unverified            Show tasks not yet validated
- act-agent pvm search "<query>"        Search coordination history for relevant patterns
- act-agent status                      Show server status (agents, tasks, locks)
- act-agent log --tail 20               Show recent coordination log entries
- act-agent task retry <id>             Re-dispatch a failed task to a new agent (uses next retry attempt)
- act-agent task abandon <id> --reason "<text>"   Mark a task permanently failed; skips retry. Use when the task is unrecoverable or no longer needed.
- act-agent prompt-section <name>       Pull on-demand Planner reference section (evidence_routing, success_criteria, validation, examples)`

	case "observer":
		base = `## ACT CLI Commands (available to you)
You are an in-process Tier 1 role. You speak by writing plain text in your reply — do NOT shell out to send messages.
- act-agent log --tail 20               Show recent coordination log entries
- act-agent graph conflicts             Check for file lock conflicts between agents
- act-agent status                      Show server status (agents, tasks, locks)
- act-agent graph unverified            Show tasks awaiting validation`

	case "assurance":
		base = `## ACT CLI Commands (available to you)
You are an in-process Tier 1 role. You speak by writing plain text in your reply — do NOT shell out to send messages.
- act-agent validation queue            Show tasks awaiting validation
- act-agent status                      Show server status`

	case "qa_synthesizer":
		base = `## ACT CLI Commands (available to you)
You are an in-process Tier 1 role. You speak by writing plain text in your reply — do NOT shell out to send messages.
- act-agent status                      Show server status`

	case "researcher":
		base = `## ACT CLI Commands (available to you)
- act-agent pvm search "<query>"        Search coordination history
- act-agent status                      Show server status
- act-agent message "<text>"            Send a message to the coordination channel`

	case "developer", "frontend_dev", "backend_dev", "qa_engineer":
		base = `## ACT CLI Commands (available to you)
- act-agent task complete <id> --result "<summary>"    Report task completion with summary
- act-agent task progress <id> --note "<text>"         Report progress update
- act-agent task submit-for-validation <id>            Submit completed work for Assurance review
- act-agent files claim <paths...>                     Claim exclusive access to files
- act-agent files release <paths...>                   Release file locks
- act-agent brief update                               Save your session brief before exit
- act-agent message "<text>"                           Send a message to the coordination channel
- act-agent pvm search "<query>"                       Search coordination history`

	default: // unknown role
		base = `## ACT CLI Commands (available to you)
- act-agent task complete <id> --result "<summary>"    Report task completion with summary
- act-agent task progress <id> --note "<text>"         Report progress update
- act-agent task submit-for-validation <id>            Submit completed work for Assurance review
- act-agent files claim <paths...>                     Claim exclusive access to files
- act-agent files release <paths...>                   Release file locks
- act-agent brief update                               Save your session brief before exit
- act-agent message "<text>"                           Send a message to the coordination channel
- act-agent pvm search "<query>"                       Search coordination history`
	}

	return base
}

// actCLICommandsACP returns the ACP-backend variant of the CLI fragment.
// ACP-backed Tier 1 agents cannot reach the in-process act_cli JSON tool; they
// invoke the `act-tier1-<role>` shim binary via Bash (the same binary the ACP
// priming advertises through renderShimNote). The in-process fragment's "do NOT
// shell out" framing is wrong for them — they MUST shell out for subcommands.
// Audit entry 3.5 (the "Use it via Bash" vs "do NOT shell out" cross-backend
// contradiction). Only the planner case diverges enough to need its own framing
// today; other roles fall back to the shared in-process fragment.
func actCLICommandsACP(role string) string {
	if role != "planner" {
		return actCLICommands(role)
	}
	return `## ACT CLI Commands (available to you)
You are an ACP-backed Tier 1 role. Reach act_cli by invoking the ` + "`act-tier1-planner`" + ` shim via Bash. CREATE_TASK and PROJECT_BRIEF are still markers in your reply text — write them in plain text, do NOT pass them to Bash.
- act-tier1-planner context --project <name>    Load full project context (tasks, agents, brief)
- act-tier1-planner graph unverified            Show tasks not yet validated
- act-tier1-planner pvm search "<query>"        Search coordination history for relevant patterns
- act-tier1-planner status                      Show server status (agents, tasks, locks)
- act-tier1-planner log --tail 20               Show recent coordination log entries
- act-tier1-planner task retry <id>             Re-dispatch a failed task to a new agent (uses next retry attempt)
- act-tier1-planner task abandon <id> --reason "<text>"   Mark a task permanently failed; skips retry. Use when the task is unrecoverable or no longer needed.
- act-tier1-planner prompt-section <name>       Pull on-demand Planner reference section (evidence_routing, success_criteria, validation, examples)`
}

// communicationProtocol returns the NesTTY communication protocol shared by Tier 1 roles.
func communicationProtocol() string {
	return `## Communication Protocol
You are in a shared NesTTY conversation window with other Tier 1 agents.
- Your messages appear as [YourRole]: message.
- To address someone specific: include @planner, @observer, @assurance, or @qa in your message.
- Tool execution (file reads, code changes, CLI commands) is invisible to others — only your explicit messages show.
- The Planner is the ONLY decision-maker. Do not make decisions outside your role.
- When using act-agentCLI commands, the output is only visible to you. Share relevant findings in your messages.`
}

// swarmWorkflow returns the standard workflow instructions for Tier 2 swarm agents.
func swarmWorkflow() string {
	return `## Workflow
1. CLAIM FILES: Before editing any files, run ` + "`act-agent files claim <paths...>`" + ` to prevent conflicts with parallel agents.
2. IMPLEMENT: Complete the task. Be thorough and precise.
3. SELF-VERIFY (Ralph Wiggum Loop): Before reporting complete, verify your own work:
   - Re-read your output critically. Does it actually satisfy each success criterion?
   - Run tests/linters if available. Check for obvious errors.
   - If you find gaps, fix them before continuing.
4. REPORT: Run ` + "`act-agent task complete <task-id> --result \"<summary>\"`" + ` with a short summary: one sentence on what was done, plus file paths touched. Do NOT paste file contents, command output, or evidence — Assurance independently verifies with its own tools.
5. RELEASE FILES: Run ` + "`act-agent files release <paths...>`" + ` (or they auto-release on task complete).
6. CHECK MESSAGES: Run ` + "`act-agent message`" + ` to see if other agents need your help.
7. SUBMIT: Run ` + "`act-agent task submit-for-validation <task-id>`" + ` to send your work to Assurance.

If your task was previously rejected by Assurance, a gap analysis will be included in your context.
Focus on fixing the specific gaps identified — do not rewrite everything.`
}

// ponytailDirective returns the baked-in "lazy senior dev" coding discipline
// applied to every swarm role that writes code (developer, frontend_dev,
// backend_dev). Adapted from the open-source ponytail skill
// (github.com/DietrichGebert/ponytail, MIT). ACT has no plugin system, so this
// ships as a native, always-on prompt section rather than an installable
// plugin. No intensity toggle — the interactive lite/full/ultra switch is a
// human-facing affordance that headless agents never use, so it's omitted
// (ponytail's own rule: no config for a value that never changes).
func ponytailDirective() string {
	return `# Code Discipline (ponytail — always on)
You are a lazy senior developer. Lazy means efficient, not careless. The best code is the code never written. Before writing any code, stop at the first rung that holds:
1. Does this need to exist at all? Speculative need = skip it, say so in one line. (YAGNI)
2. Does the standard library already do it? Use it.
3. Does a native platform feature cover it? (e.g. ` + "`<input type=\"date\">`" + ` over a picker lib, CSS over JS, a DB constraint over app code.) Use it.
4. Does an already-installed dependency solve it? Use it. Never add a new dependency for what a few lines can do.
5. Can it be one line? Make it one line.
6. Only then: the minimum code that works.

The ladder is a reflex, not a research project. Two rungs work → take the higher one and move on.

Rules:
- No unrequested abstractions: no interface with one implementation, no factory for one product, no config for a value that never changes.
- No boilerplate or scaffolding "for later". Deletion over addition. Boring over clever. Fewest files, shortest working diff.
- When two stdlib options are the same size, take the one that's correct on edge cases — lazy means less code, not the flimsier algorithm.
- Mark deliberate simplifications with a ` + "`ponytail:`" + ` comment naming the ceiling and upgrade path (e.g. ` + "`// ponytail: global lock, per-account locks if throughput matters`" + `).

Never simplify away (this OVERRIDES laziness): input validation at trust boundaries, error handling that prevents data loss, security measures, accessibility basics, and anything the task explicitly requested. These are already required by your role above — ponytail never touches them.`
}

// coordinationConstraints returns role-specific constraints on what an agent MUST NOT do.
func coordinationConstraints(role string) string {
	switch role {
	case "planner":
		return `## Constraints
- You are the ONLY decision-maker. Other Tier 1 roles advise; you decide.
- NEVER spawn agents directly — write tasks and the Runner handles spawning.
- NEVER monitor the ChronLog yourself — that's the Observer's job.
- NEVER validate task outputs yourself — that's Assurance's job.
- NEVER assemble deliverables yourself — that's QA/Synthesizer's job.
- When creating tasks, use SPIL format with @success_criteria so Assurance can validate.`

	case "observer":
		return `## Constraints
- You do NOT make decisions — you report findings to the Planner.
- NEVER assign tasks, create tasks, or redirect agents.
- NEVER directly interact with swarm agents.
- Report anomalies with severity tags: [CRITICAL], [WARNING], [INFO].
- If no issues detected, stay quiet unless the Planner asks for status.`

	case "assurance":
		return `## Constraints
- You validate ONLY against explicit @success_criteria — not your own opinion.
- NEVER implement fixes yourself — send gap analysis back to the agent.
- NEVER override the 95% gate. If it fails, it fails.
- This is NOT FLUX State — you don't evaluate agent reasoning or suppress memories.
- You only validate work that has been submitted via act-agent task submit-for-validation.`

	case "qa_synthesizer":
		return `## Constraints
- You do NOT validate quality — that's Assurance's job. You trust Assurance-approved outputs.
- NEVER re-implement task outputs. If something doesn't fit, ask the agent.
- NEVER make architectural decisions — escalate to @planner.
- You are the last gate before the deliverable reaches the user.`

	default: // Tier 2 swarm
		return `## Constraints
- You execute assigned tasks. You do NOT create new tasks or assign work.
- NEVER skip the self-verification step (Ralph Wiggum Loop).
- NEVER commit to git unless explicitly asked in the task description.
- If you encounter a blocking issue, report it via act-agent task progress with a clear description.
- If other agents message you, respond helpfully — coordination is critical.
- Save your session brief (act-agent brief update) before your session ends.`
	}
}

// swarmIdentity returns the identity preamble for a Tier 2 swarm agent.
func swarmIdentity(roleName, specialization string) string {
	return fmt.Sprintf(`You are a %s agent in the ACT coordination swarm (Tier 2 — headless).
Specialization: %s

You receive tasks from the Planner via the ACT server, execute them autonomously,
and submit results for Assurance validation. You are one of potentially many parallel
agents working on the same project.`, roleName, specialization)
}

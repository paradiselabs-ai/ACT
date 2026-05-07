package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/permission"
)

// ActCLIToolName is the name the LLM sees in its tool catalog. A narrow,
// named action (not "bash") so the model learns the allowed surface from
// the tool schema alone.
const ActCLIToolName = "act_cli"

// Timeout bounds (milliseconds) for a single act CLI invocation. These are
// deliberately tighter than BashTool's limits — Tier 1 agents should be
// making quick, purpose-built calls, not long-running shell work.
const (
	ActCLIDefaultTimeout = 30 * 1000  // 30 seconds
	ActCLIMaxTimeout     = 120 * 1000 // 2 minutes
)

// Shell metacharacters that should never appear in args. We invoke via
// os/exec without a shell, so these can't actually inject — but Tier 1
// agents passing them in is a sign of confusion (treating act_cli like
// bash). Reject with a loud message so the model corrects course.
var actCLIBannedArgSubstrings = []string{";", "|", "&&", "||", "$(", "`"}

// Subcommands that mutate shared state. These require a permission prompt
// before running. All other allowed subcommands are read-only and fire
// without prompting.
var actCLIMutatingSubcommands = map[string]bool{
	"message": true, // broadcasts to other agents — treat as side-effect
}

// ActCLIParams is the JSON input shape the LLM sends.
type ActCLIParams struct {
	Subcommand string   `json:"subcommand"`
	Args       []string `json:"args,omitempty"`
	Timeout    int      `json:"timeout,omitempty"`
}

type actCLIPermissionsParams struct {
	Role       string   `json:"role"`
	Subcommand string   `json:"subcommand"`
	Args       []string `json:"args"`
}

type actCLITool struct {
	permissions permission.Service
	role        string          // owning Tier 1 role; drives whitelist
	allowed     map[string]bool // precomputed whitelist for O(1) check
	allowedList []string        // preserved order for Info().Description
}

// NewActCLITool returns a BaseTool that runs the `act` coordination CLI with
// the given Tier 1 role's subcommand whitelist. Subcommands outside the
// whitelist are rejected before exec. Shell metacharacters in args are
// rejected. Every invocation logs role + subcommand for auditability.
//
// role must be one of the keys in RoleSubcommands (see act_cli_whitelist.go).
// An unknown role returns a tool whose Run always errors — caller should
// treat "tools not correctly wired" as a config bug and fix it rather than
// fail at runtime.
func NewActCLITool(role string, permissions permission.Service) BaseTool {
	list := AllowedFor(role)
	allowed := make(map[string]bool, len(list))
	for _, s := range list {
		allowed[s] = true
	}
	return &actCLITool{
		permissions: permissions,
		role:        role,
		allowed:     allowed,
		allowedList: list,
	}
}

func (t *actCLITool) Info() ToolInfo {
	allowedStr := strings.Join(t.allowedList, ", ")
	desc := fmt.Sprintf(`Call the ACT coordination CLI. Runs "act-agent <subcommand> [args]" and returns stdout.

ALLOWED subcommands for role %q: %s

The subcommand you pass is the FIRST positional token after "act-agent". Sub-sub commands (like "pvm search" or "graph unverified") go into args. Examples:
- Check system status: {"subcommand":"status"}
- Get log: {"subcommand":"log","args":["--limit","20"]}
- PVM search: {"subcommand":"pvm","args":["search","markdown URL extraction"]}
- Graph unverified tasks: {"subcommand":"graph","args":["unverified"]}
- Project context: {"subcommand":"context","args":["--project","checklinks"]}

Do NOT attempt subcommands outside the allowed list — they will be rejected. Do NOT include shell metacharacters (;, |, &, $(), backticks) in args.

Timeout defaults to 30000ms (30s), max 120000ms (2m).`, t.role, allowedStr)

	return ToolInfo{
		Name:        ActCLIToolName,
		Description: desc,
		Parameters: map[string]any{
			"subcommand": map[string]any{
				"type":        "string",
				"description": "The act CLI subcommand (one of: " + allowedStr + ")",
				"enum":        t.allowedList,
			},
			"args": map[string]any{
				"type":        "array",
				"description": "Positional arguments and flags for the subcommand. E.g. for \"act-agent pvm search X\" use args=[\"search\",\"X\"].",
				"items": map[string]any{
					"type": "string",
				},
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Optional timeout in milliseconds (default 30000, max 120000).",
			},
		},
		Required: []string{"subcommand"},
	}
}

func (t *actCLITool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ActCLIParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	if params.Subcommand == "" {
		return NewTextErrorResponse("missing subcommand"), nil
	}

	// Whitelist check — this is the core of KI-02. Subcommands outside the
	// role's whitelist are rejected with an actionable error so the model
	// can correct on the same turn.
	if !t.allowed[params.Subcommand] {
		return NewTextErrorResponse(fmt.Sprintf(
			"subcommand %q is not allowed for role %q. Allowed: %s. "+
				"Do NOT attempt ls, cat, go, git, or other shell commands — "+
				"this tool only runs the act coordination CLI.",
			params.Subcommand, t.role, strings.Join(t.allowedList, ", "),
		)), nil
	}

	// Defensive metachar guard on args. Since we exec via os/exec without a
	// shell, these can't actually inject — but passing them is a sign the
	// model is confusing act_cli with bash. Fail loudly so it course-corrects.
	for _, arg := range params.Args {
		for _, banned := range actCLIBannedArgSubstrings {
			if strings.Contains(arg, banned) {
				return NewTextErrorResponse(fmt.Sprintf(
					"arg %q contains shell metacharacter %q — act_cli does not run a shell. "+
						"Pass plain arg tokens (no quoting, no $(), no pipes).",
					arg, banned,
				)), nil
			}
		}
	}

	// Permission prompt for mutating subcommands (currently just "message").
	// Read-only subcommands run without prompting to keep Tier 1 turns fast.
	if actCLIMutatingSubcommands[params.Subcommand] {
		sessionID, _ := GetContextValues(ctx)
		if sessionID == "" {
			return ToolResponse{}, fmt.Errorf("session ID required for mutating act_cli subcommand")
		}
		ok := t.permissions.Request(permission.CreatePermissionRequest{
			SessionID: sessionID,
			Path:      config.WorkingDirectory(),
			ToolName:  ActCLIToolName,
			Action:    "execute",
			Description: fmt.Sprintf("Run: act-agent %s %s",
				params.Subcommand, strings.Join(params.Args, " ")),
			Params: actCLIPermissionsParams{
				Role:       t.role,
				Subcommand: params.Subcommand,
				Args:       params.Args,
			},
		})
		if !ok {
			return ToolResponse{}, permission.ErrorPermissionDenied
		}
	}

	// Timeout bounds.
	timeoutMs := params.Timeout
	if timeoutMs <= 0 {
		timeoutMs = ActCLIDefaultTimeout
	}
	if timeoutMs > ActCLIMaxTimeout {
		timeoutMs = ActCLIMaxTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	logging.Info("act_cli_invoke",
		"role", t.role,
		"subcommand", params.Subcommand,
		"args_count", len(params.Args),
		"timeout_ms", timeoutMs,
	)

	startTime := time.Now()
	cmdArgs := append([]string{params.Subcommand}, params.Args...)
	cmd := exec.CommandContext(execCtx, "act-agent", cmdArgs...)
	cmd.Dir = config.WorkingDirectory()

	out, err := cmd.CombinedOutput()
	elapsed := time.Since(startTime)
	stdout := truncateOutput(string(out))

	metadata := BashResponseMetadata{
		StartTime: startTime.UnixMilli(),
		EndTime:   time.Now().UnixMilli(),
	}

	if err != nil {
		exitCode := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		// Context timeout vs genuine exit error — distinguish for the model.
		if execCtx.Err() == context.DeadlineExceeded {
			logging.Warn("act_cli_timeout",
				"role", t.role,
				"subcommand", params.Subcommand,
				"timeout_ms", timeoutMs,
			)
			return WithResponseMetadata(NewTextErrorResponse(fmt.Sprintf(
				"act-agent %s timed out after %dms. Output so far:\n%s",
				params.Subcommand, timeoutMs, stdout,
			)), metadata), nil
		}
		logging.Info("act_cli_nonzero_exit",
			"role", t.role,
			"subcommand", params.Subcommand,
			"exit_code", exitCode,
			"elapsed_ms", elapsed.Milliseconds(),
		)
		// Non-zero exit — still return output so the model can inspect stderr.
		if stdout == "" {
			stdout = fmt.Sprintf("act-agent %s exited %d with no output (error: %s)",
				params.Subcommand, exitCode, err.Error())
		}
		return WithResponseMetadata(NewTextErrorResponse(stdout), metadata), nil
	}

	if stdout == "" {
		stdout = "(no output)"
	}
	return WithResponseMetadata(NewTextResponse(stdout), metadata), nil
}

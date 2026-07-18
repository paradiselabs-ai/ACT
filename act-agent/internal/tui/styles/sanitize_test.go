package styles

import (
	"strings"
	"testing"
)

func TestRewriteFileLinks_RelativePath(t *testing.T) {
	root := "C:/Users/zadka/Desktop/ACT/act-agent/"
	content := "[cli/](file:///C:/Users/zadka/Desktop/ACT/act-agent/cli)"
	got := rewriteFileLinks(content, root)
	if got != "cli" {
		t.Errorf("expected 'cli', got %q", got)
	}
}

func TestRewriteFileLinks_PreservesLabel(t *testing.T) {
	root := "C:/Users/zadka/Desktop/ACT/act-agent/"
	content := "[internal](file:///C:/Users/zadka/Desktop/ACT/act-agent/internal)"
	got := rewriteFileLinks(content, root)
	if got != "internal" {
		t.Errorf("expected 'internal', got %q", got)
	}
}

func TestRewriteFileLinks_NonRelativizable(t *testing.T) {
	root := "C:/Users/other/project/"
	content := "[label](file:///C:/Users/zadka/Desktop/ACT/act-agent/cli)"
	got := rewriteFileLinks(content, root)
	if got != "label" {
		t.Errorf("expected 'label', got %q", got)
	}
}

func TestRewriteFileLinks_MultipleLinks(t *testing.T) {
	root := "C:/Users/zadka/Desktop/ACT/act-agent/"
	content := "See [cli/](file:///C:/Users/zadka/Desktop/ACT/act-agent/cli) and [cmd/](file:///C:/Users/zadka/Desktop/ACT/act-agent/cmd)"
	got := rewriteFileLinks(content, root)
	expected := "See cli and cmd"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestLinkCorruption_URLExceedsWrapWidthLabelDoesnt(t *testing.T) {
	root := "C:/Users/zadka/Desktop/ACT/act-agent/"
	content := "Check [cli](file:///C:/Users/zadka/Desktop/ACT/act-agent/internal/very/deep/nested/path/cli) for details."
	sanitized := SanitizeForTUI(content, 60, root, false)

	if strings.Contains(sanitized, "file:///") {
		t.Errorf("file:/// URL should have been rewritten, got: %q", sanitized)
	}
	if !strings.Contains(sanitized, "for details.") {
		t.Errorf("trailing text after link was swallowed, got: %q", sanitized)
	}
	if strings.Count(sanitized, "[") != strings.Count(sanitized, "]") {
		t.Errorf("unbalanced brackets in: %q", sanitized)
	}
	if strings.Count(sanitized, "(") != strings.Count(sanitized, ")") {
		t.Errorf("unbalanced parens in: %q", sanitized)
	}
}

func TestTruncateLongTokens_ShortTokensUnchanged(t *testing.T) {
	content := "hello world short tokens"
	got := processContentLines(content, 80, false)
	if got != content {
		t.Errorf("short tokens should be unchanged, got %q", got)
	}
}

func TestTruncateLongTokens_LongBareToken(t *testing.T) {
	token := strings.Repeat("a", 100)
	content := "before " + token + " after"
	got := processContentLines(content, 60, false)
	if strings.Contains(got, token) {
		t.Errorf("100-char token should have been truncated, got: %q", got)
	}
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Errorf("surrounding words should survive truncation, got: %q", got)
	}
	if !strings.Contains(got, "…") {
		t.Errorf("truncated token should contain '…', got: %q", got)
	}
}

func TestTruncateLongTokens_PreservesFencedCodeBlocks(t *testing.T) {
	longToken := strings.Repeat("x", 100)
	content := "```\n" + longToken + "\n```"
	got := processContentLines(content, 60, false)
	if !strings.Contains(got, longToken) {
		t.Errorf("content inside fenced code block should be preserved, got: %q", got)
	}
}

func TestTruncateLongTokens_PreservesInlineCode(t *testing.T) {
	longPath := strings.Repeat("a", 80)
	content := "Use `" + longPath + "` for reference"
	got := processContentLines(content, 60, false)
	if !strings.Contains(got, longPath) {
		t.Errorf("inline code should be preserved, got: %q", got)
	}
}

func TestTabToSpace(t *testing.T) {
	content := "line1\tindented\n\t\tline2"
	got := SanitizeForTUI(content, 80, "/tmp", false)
	if strings.Contains(got, "\t") {
		t.Errorf("tabs should be converted to spaces, got: %q", got)
	}
	if !strings.Contains(got, "    ") {
		t.Errorf("tabs should become 4 spaces, got: %q", got)
	}
}

func TestCapContentHeight_UnderLimit(t *testing.T) {
	content := "line1\nline2\nline3"
	got := CapContentHeight(content, 10)
	if got != content {
		t.Errorf("content under limit should be unchanged, got %q", got)
	}
}

func TestCapContentHeight_OverLimit(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	content := strings.Join(lines, "\n")
	got := CapContentHeight(content, 10)
	resultLines := strings.Split(got, "\n")
	if len(resultLines) != 11 {
		t.Errorf("expected 11 lines (10 + footer), got %d", len(resultLines))
	}
	if !strings.Contains(got, "90 more lines") {
		t.Errorf("footer should mention 90 remaining lines, got: %q", resultLines[len(resultLines)-1])
	}
}

func TestSanitizeForTUI_WindowsBackslashPaths(t *testing.T) {
	root := `C:\Users\zadka\Desktop\ACT\act-agent`
	content := `[cli/](file:///C:\Users\zadka\Desktop\ACT\act-agent\cli)`
	got := SanitizeForTUI(content, 80, root, false)
	if strings.Contains(got, "file:///") {
		t.Errorf("Windows paths should be handled, got: %q", got)
	}
}

func TestBulletPointReformatting(t *testing.T) {
	content := "• **Tech Stack**: Go 1.25.8\n- **Database**: Postgres"
	got := SanitizeForTUI(content, 80, "/tmp", false)
	expected := "• Tech Stack: **Go 1.25.8**\n- Database: **Postgres**"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestBulletPointReformatting_BoldCollision(t *testing.T) {
	// If the value already contains bolding, we should NOT wrap it in bold again
	content := "• **Config**: Use **strict** mode"
	got := SanitizeForTUI(content, 80, "/tmp", false)
	expected := "• Config: Use **strict** mode"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestQuestionOptionHighlighting_AwaitingInput(t *testing.T) {
	content := "Ready to start?\n1. Confirm database\n2. Retry task"
	got := SanitizeForTUI(content, 80, "/tmp", true)
	expected := "❯ **Ready to start?**\n1. ❯ **Confirm database**\n2. ❯ **Retry task**"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestQuestionOptionHighlighting_OverlappingNumberedQuestion(t *testing.T) {
	// A line like "1. Should we deploy to prod?" satisfies both the option regex
	// and standalone question regex. We must process it as option first and NOT double-wrap.
	content := "1. Should we deploy to prod?"
	got := SanitizeForTUI(content, 80, "/tmp", true)
	expected := "1. ❯ **Should we deploy to prod?**"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestJaccardDeduplication(t *testing.T) {
	userPrompt := "Description: Create a text file named success.txt containing Task Complete. Tech Stack: bash Constraints: None Success Criteria: success.txt exists."
	assistantSummary := "• Description: Create a text file named success.txt containing Task Complete.\n• Tech Stack: bash\n• Constraints: None\n• Success Criteria: success.txt exists.\n\nReady to start?"

	// Jaccard similarity should exceed 85% and trigger the deduplication collapse
	got := SanitizeForTUI(assistantSummary, 80, "/tmp", true, userPrompt)
	expected := "> Project specifications loaded. Awaiting confirmation.\n\nReady to start?"
	if got != expected {
		t.Errorf("expected collapsed specs, got %q", got)
	}
}

func TestJaccardDeduplication_PhaseGating(t *testing.T) {
	// Deduplication only runs if a non-empty userPrompt is passed in.
	content := "• Description: Create success.txt\n• Tech Stack: bash"
	got := SanitizeForTUI(content, 80, "/tmp", false) // no userPrompt passed
	if !strings.Contains(got, "Tech Stack") {
		t.Errorf("deduplication should not trigger without userPrompt, got %q", got)
	}
}

func TestQuestionOptionHighlighting_BoldCollision(t *testing.T) {
	content := "Ready to start **immediately**?\n1. Confirm **database** now"
	got := SanitizeForTUI(content, 80, "/tmp", true)
	expected := "❯ Ready to start **immediately**?\n1. ❯ Confirm **database** now"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestJaccardDeduplication_MangleCheck(t *testing.T) {
	userPrompt := "Create a database schema for user profiles. Fields: id, username, email. Tech Stack: postgres."
	assistantSummary := "• Create a database schema for user profiles.\n• Fields: id, username, email.\n• Tech Stack: postgres.\n\nReady to start?"
	got := SanitizeForTUI(assistantSummary, 80, "/tmp", true, userPrompt)
	expected := "> Project specifications loaded. Awaiting confirmation.\n\nReady to start?"
	if got != expected {
		t.Errorf("expected clean replacement, got %q", got)
	}
}

func TestSanitizeForTUI_NoMangle(t *testing.T) {
	content := "Based on the codebase analysis, here is my understanding:\n• Tech Stack: Go\n\nPlease let me know if you have any corrections.\nReady to start?"
	got := SanitizeForTUI(content, 80, "/tmp", false)
	if !strings.Contains(got, "corrections") || !strings.Contains(got, "understanding") {
		t.Errorf("content was mangled, got %q", got)
	}
}

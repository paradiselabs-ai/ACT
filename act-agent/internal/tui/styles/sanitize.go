package styles

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
)

// ── markdown-aware link pattern ──────────────────────────────────────────────
// Matches [label](file:///absolute/path) constructs. Captures:
//   group 1: the label text
//   group 2: the full file:/// URL
var markdownFileLinkRe = regexp.MustCompile(`\[([^\]]*)\]\((file:///[^)]+)\)`)

// ── inline backtick span pattern ─────────────────────────────────────────────
// Matches `...` (single backtick) inline code spans. We skip sanitisation
// inside these to avoid corrupting code references.
var inlineCodeRe = regexp.MustCompile("`[^`]+`")

// ── bare file:/// URL pattern ────────────────────────────────────────────────
// Matches file:/// URLs that are NOT inside a markdown link construct.
var bareFileURLRe = regexp.MustCompile(`file:///[^\s)\]]+`)

// ── bullet-point bold label pattern ──────────────────────────────────────────
// Matches: • **Label**: value
var bulletRe = regexp.MustCompile(`^(\s*[•\-\*]\s+)\*\*([^\*]+)\*\*:\s*(.+)$`)

// ── numbered option pattern ──────────────────────────────────────────────────
// Matches: 1. Option text
var optionRe = regexp.MustCompile(`^(\s*\d+\.\s+)(.+)$`)

// ── standalone question pattern ──────────────────────────────────────────────
// Matches: Ready to start? (ends with ?)
var questionRe = regexp.MustCompile(`^([^\s•\-\*\d].*\?)$`)

// SanitizeForTUI pre-processes markdown content before Glamour rendering.
// It performs three ordered passes:
//
//  1. Jaccard Similarity Deduplication: Gated to run only in intake/analysis
//     phases. Collapses redundant specification lists.
//
//  2. Markdown-aware link rewrite: [label](file:///abs/path) → relative path text.
//     This runs before truncation so that token parsing cannot corrupt link syntax.
//
//  3. Unified Line Scanning: A single, stateful line-by-line loop that tracks
//     fenced code blocks (```) and performs long-token truncation, bullet-point
//     label reformatting, and pre-render question/option highlights.
//
// Additionally, tabs are converted to 4 spaces for consistent lipgloss.Width()
// alignment across terminals.
//
// projectRoot is used to compute relative paths from file:/// URLs. If empty,
// config.WorkingDirectory() is used as fallback.
func SanitizeForTUI(content string, maxWidth int, projectRoot string, isAwaitingInput bool, userPrompt ...string) string {
	if maxWidth < 20 {
		maxWidth = 20
	}

	if projectRoot == "" {
		projectRoot = config.WorkingDirectory()
	}

	// Normalise path separators for matching.
	projectRoot = strings.ReplaceAll(projectRoot, `\`, "/")
	if !strings.HasSuffix(projectRoot, "/") {
		projectRoot += "/"
	}

	// ── Pass 1: Jaccard Similarity Deduplication ─────────────────────────
	if len(userPrompt) > 0 && userPrompt[0] != "" && strings.Contains(content, "•") {
		similarity := JaccardSimilarity(content, userPrompt[0])
		if similarity > 0.85 {
			readyText := "Ready to start?"
			if strings.Contains(content, readyText) {
				return "> Project specifications loaded. Awaiting confirmation.\n\nReady to start?"
			}
			return "> Project specifications loaded. Awaiting confirmation."
		}
	}

	// Tab → spaces (consistent width across terminals).
	content = strings.ReplaceAll(content, "\t", "    ")

	// ── Pass 2: markdown-aware link rewrite ──────────────────────────────
	content = rewriteFileLinks(content, projectRoot)
	content = rewriteBareFileURLs(content, projectRoot)

	// ── Pass 3: Unified stateful line scanning ───────────────────────────
	content = processContentLines(content, maxWidth, isAwaitingInput)

	return content
}

// rewriteFileLinks replaces markdown file links with relative path text.
func rewriteFileLinks(content string, projectRoot string) string {
	return markdownFileLinkRe.ReplaceAllStringFunc(content, func(match string) string {
		subs := markdownFileLinkRe.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		urlPath := subs[2]
		path := strings.TrimPrefix(urlPath, "file:///")
		path = strings.ReplaceAll(path, `\`, "/")

		normRoot := strings.ToLower(projectRoot)
		normPath := strings.ToLower(path)
		if strings.HasPrefix(normPath, normRoot) {
			rel := path[len(projectRoot):]
			if rel == "" {
				rel = "."
			}
			return rel
		}

		label := subs[1]
		if label != "" {
			return label
		}
		return truncateToken(path, 60)
	})
}

// rewriteBareFileURLs replaces bare file:/// URLs with relative paths.
func rewriteBareFileURLs(line string, projectRoot string) string {
	return bareFileURLRe.ReplaceAllStringFunc(line, func(match string) string {
		path := strings.TrimPrefix(match, "file:///")
		path = strings.ReplaceAll(path, `\`, "/")

		normRoot := strings.ToLower(projectRoot)
		normPath := strings.ToLower(path)
		if strings.HasPrefix(normPath, normRoot) {
			rel := path[len(projectRoot):]
			if rel == "" {
				rel = "."
			}
			return rel
		}
		return truncateToken(path, 60)
	})
}

// processContentLines processes lines statefully to prevent mangling inside code blocks.
func processContentLines(content string, maxWidth int, isAwaitingInput bool) string {
	lines := strings.Split(content, "\n")
	inFence := false
	tokenMax := maxWidth - 4
	if tokenMax < 20 {
		tokenMax = 20
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Toggle code block state
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}

		if inFence {
			continue
		}

		// ── Pass A: Bullet point label reformatting ──────────────────
		// • **Label**: value → • Label: **value**
		if m := bulletRe.FindStringSubmatch(line); len(m) == 4 {
			prefix := m[1]
			label := m[2]
			val := m[3]
			if strings.Contains(val, "**") {
				line = fmt.Sprintf("%s%s: %s", prefix, label, val)
			} else {
				line = fmt.Sprintf("%s%s: **%s**", prefix, label, val)
			}
		}

		// ── Pass B: Question/Option highlighting ─────────────────────
		if isAwaitingInput {
			// Check if numbered option first (e.g. 1. Option)
			if m := optionRe.FindStringSubmatch(line); len(m) == 3 {
				prefix := m[1]
				val := m[2]
				if strings.Contains(val, "**") {
					line = fmt.Sprintf("%s❯ %s", prefix, val)
				} else {
					line = fmt.Sprintf("%s❯ **%s**", prefix, val)
				}
			} else if questionRe.MatchString(line) {
				// Standalone question
				if strings.Contains(line, "**") {
					line = "❯ " + line
				} else {
					line = "❯ **" + line + "**"
				}
			}
		}

		// ── Pass C: Token Truncation ─────────────────────────────────
		lines[i] = truncateLineTokens(line, tokenMax)
	}

	return strings.Join(lines, "\n")
}

// truncateLineTokens truncates long tokens outside of inline code spans.
func truncateLineTokens(line string, maxTokenLen int) string {
	type protectedSpan struct {
		start, end int
	}
	var protected []protectedSpan

	for _, loc := range inlineCodeRe.FindAllStringIndex(line, -1) {
		protected = append(protected, protectedSpan{loc[0], loc[1]})
	}
	for _, loc := range markdownFileLinkRe.FindAllStringIndex(line, -1) {
		protected = append(protected, protectedSpan{loc[0], loc[1]})
	}

	isProtected := func(pos int) bool {
		for _, s := range protected {
			if pos >= s.start && pos < s.end {
				return true
			}
		}
		return false
	}

	var result strings.Builder
	result.Grow(len(line))
	wordStart := -1

	flush := func(end int) {
		if wordStart < 0 {
			return
		}
		word := line[wordStart:end]
		if len(word) > maxTokenLen && !isProtected(wordStart) {
			word = truncateToken(word, maxTokenLen)
		}
		result.WriteString(word)
		wordStart = -1
	}

	for i, ch := range line {
		if ch == ' ' || ch == '\t' {
			flush(i)
			result.WriteRune(ch)
		} else {
			if wordStart < 0 {
				wordStart = i
			}
		}
	}
	flush(len(line))

	return result.String()
}

// truncateToken shortens a token using head…tail format.
func truncateToken(token string, maxLen int) string {
	if len(token) <= maxLen {
		return token
	}
	if maxLen < 10 {
		return ansi.Truncate(token, maxLen, "…")
	}
	headLen := (maxLen * 3) / 5
	tailLen := maxLen - headLen - 1
	if tailLen < 3 {
		tailLen = 3
		headLen = maxLen - tailLen - 1
	}
	return token[:headLen] + "…" + token[len(token)-tailLen:]
}

// CapContentHeight truncates content to maxLines.
func CapContentHeight(content string, maxLines int) string {
	if maxLines <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	truncated := strings.Join(lines[:maxLines], "\n")
	remaining := len(lines) - maxLines
	return truncated + fmt.Sprintf("\n +%d more ▸", remaining)
}

// JaccardSimilarity computes set similarity between normalized texts.
func JaccardSimilarity(textA, textB string) float64 {
	wordsA := getWordSet(textA)
	wordsB := getWordSet(textB)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	intersection := 0
	for word := range wordsA {
		if wordsB[word] {
			intersection++
		}
	}

	union := len(wordsA)
	for word := range wordsB {
		if !wordsA[word] {
			union++
		}
	}

	return float64(intersection) / float64(union)
}

func getWordSet(text string) map[string]bool {
	text = strings.ToLower(text)
	replacer := strings.NewReplacer(
		"•", "", "*", "", "-", "", "#", "",
		".", "", ",", "", ":", "", ";", "",
		"?", "", "!", "", "(", "", ")", "",
		"[", "", "]", "", "`", "", "\"", "",
	)
	text = replacer.Replace(text)
	words := strings.Fields(text)
	set := make(map[string]bool)
	for _, w := range words {
		if len(w) > 2 {
			set[w] = true
		}
	}
	return set
}

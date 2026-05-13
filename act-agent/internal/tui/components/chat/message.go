package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/x/ansi"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/diff"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/agent"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/tools"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

type uiMessageType int

const (
	userMessageType uiMessageType = iota
	assistantMessageType
	toolMessageType

	maxResultHeight = 30
)

type uiMessage struct {
	ID          string
	messageType uiMessageType
	position    int
	height      int
	content     string
}

func toMarkdown(content string, focused bool, width int) string {
	r := styles.GetMarkdownRenderer(width)
	rendered, _ := r.Render(content)
	return rendered
}

// toPlainText renders content without Glamour. Used for streaming assistant
// messages: Glamour's CommonMark parse + ANSI syntax highlight costs seconds
// on multi-KB content, and re-runs every token update when the message cache
// is invalidated. During stream we show raw text; on FinishReasonEndTurn the
// caller re-renders through toMarkdown so the finished message looks identical
// to the pre-change behavior.
func toPlainText(content string, width int) string {
	return styles.BaseStyle().Width(width).Render(content)
}

func roleColor(role string) color.Color {
	t := theme.CurrentTheme()
	switch role {
	case "planner":
		return t.Primary()
	case "observer":
		return t.TextMuted()
	case "assurance":
		return compat.AdaptiveColor{Dark: lipgloss.Color("#8a7755"), Light: lipgloss.Color("#9a8a60")}
	case "qa", "qa_synthesizer":
		return compat.AdaptiveColor{Dark: lipgloss.Color("#5a8a60"), Light: lipgloss.Color("#4a7a50")}
	default:
		return t.Primary()
	}
}

func roleLabel(role string, width int) string {
	if role == "" {
		return ""
	}

	// Planner gets a bold uppercase banner; other roles get a muted dot prefix
	if role == "planner" {
		return styles.BaseStyle().
			Foreground(roleColor(role)).
			Bold(true).
			Width(width).
			Render("  PLANNER")
	}

	label := role
	if role == "qa_synthesizer" {
		label = "qa"
	}
	return styles.BaseStyle().
		Foreground(roleColor(role)).
		Width(width).
		Render("  \u00b7 " + label)
}

func renderMessage(msg string, isUser bool, isFocused bool, width int, role string, info ...string) string {
	return renderMessageBody(msg, isUser, isFocused, width, role, true, info...)
}

// renderMessageBody is renderMessage with an explicit useMarkdown switch. When
// useMarkdown is false, Glamour rendering is skipped entirely — the body is
// rendered as plain text inside the same border/padding treatment. Used for
// streaming assistant messages (see KI-01 in KNOWN_ISSUES.md) where Glamour's
// per-token re-render was blocking the Bubbletea Update loop for seconds.
func renderMessageBody(msg string, isUser bool, isFocused bool, width int, role string, useMarkdown bool, info ...string) string {
	t := theme.CurrentTheme()

	style := styles.BaseStyle().
		Width(width).
		BorderLeft(true).
		Foreground(t.TextMuted()).
		BorderForeground(t.Primary()).
		BorderStyle(lipgloss.ThickBorder())

	if isUser {
		style = style.BorderForeground(t.Secondary())
	} else if role != "" {
		style = style.BorderForeground(roleColor(role))
	}

	var body string
	if useMarkdown {
		body = styles.ForceReplaceBackgroundWithLipgloss(toMarkdown(msg, isFocused, width), t.Background())
	} else {
		body = toPlainText(msg, width)
	}

	parts := []string{strings.TrimSuffix(body, "\n")}
	if len(info) > 0 {
		parts = append(parts, info...)
	}

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		),
	)
}

func renderUserMessage(msg message.Message, isFocused bool, width int, position int) uiMessage {
	var styledAttachments []string
	t := theme.CurrentTheme()
	attachmentStyles := styles.BaseStyle().
		MarginLeft(1).
		Background(t.TextMuted()).
		Foreground(t.Text())
	for _, attachment := range msg.BinaryContent() {
		file := filepath.Base(attachment.Path)
		var filename string
		if len(file) > 10 {
			filename = fmt.Sprintf(" %s %s...", styles.DocumentIcon, file[0:7])
		} else {
			filename = fmt.Sprintf(" %s %s", styles.DocumentIcon, file)
		}
		styledAttachments = append(styledAttachments, attachmentStyles.Render(filename))
	}
	content := ""
	if len(styledAttachments) > 0 {
		attachmentContent := styles.BaseStyle().Width(width).Render(lipgloss.JoinHorizontal(lipgloss.Left, styledAttachments...))
		content = renderMessage(msg.Content().String(), true, isFocused, width, "", attachmentContent)
	} else {
		content = renderMessage(msg.Content().String(), true, isFocused, width, "")
	}
	userMsg := uiMessage{
		ID:          msg.ID,
		messageType: userMessageType,
		position:    position,
		height:      lipgloss.Height(content),
		content:     content,
	}
	return userMsg
}

// renderSystemMessage renders an orchestrator-injected coordination event
// (task created, agent completed, validation passed/failed, agent message,
// etc) as a single muted line with no avatar / no header. The TUI's chat
// list switch routes Role==System messages here so the user sees real-time
// swarm activity without having to tail a log file.
func renderSystemMessage(msg message.Message, width int, position int) uiMessage {
	t := theme.CurrentTheme()
	text := msg.Content().String()
	if text == "" {
		text = "(empty system message)"
	}
	const maxSystemLines = 5
	lines := strings.Split(text, "\n")
	if len(lines) > maxSystemLines {
		lines = lines[:maxSystemLines]
		lines[maxSystemLines-1] = ansi.Truncate(lines[maxSystemLines-1], width-6, "") + " …"
		text = strings.Join(lines, "\n")
	}
	rendered := styles.BaseStyle().
		Foreground(t.TextMuted()).
		Width(width).
		PaddingLeft(2).
		Render(text)
	return uiMessage{
		ID:          msg.ID,
		messageType: userMessageType, // reuse user type slot — list only checks for height/content
		position:    position,
		height:      lipgloss.Height(rendered),
		content:     rendered,
	}
}

// Returns multiple uiMessages because of the tool calls.
// useMarkdown decouples the Glamour render decision from msg.IsFinished() so
// callers can defer Glamour off the Update goroutine. When false, the body is
// rendered as plain text regardless of finish state — used by the sync
// renderView path while the async Glamour cmd is in flight. When true, full
// Glamour markdown is applied — used by the async goroutine.
func renderAssistantMessage(
	msg message.Message,
	role string,
	msgIndex int,
	allMessages []message.Message, // we need this to get tool results and the user message
	messagesService message.Service, // We need this to get the task tool messages
	focusedUIMessageId string,
	isSummary bool,
	width int,
	position int,
	useMarkdown bool,
) []uiMessage {
	messages := []uiMessage{}
	content := msg.Content().String()
	thinking := msg.IsThinking()
	thinkingContent := msg.ReasoningContent().Thinking
	finished := msg.IsFinished()
	finishData := msg.FinishPart()
	info := []string{}

	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	// Add finish info if available
	if finished {
		switch finishData.Reason {
		case message.FinishReasonEndTurn:
			took := formatTimestampDiff(msg.CreatedAt, finishData.Time)
			info = append(info, baseStyle.
				Width(width-1).
				Foreground(t.TextMuted()).
				Render(fmt.Sprintf(" %s (%s)", string(msg.Model), took)),
			)
		case message.FinishReasonCanceled:
			info = append(info, baseStyle.
				Width(width-1).
				Foreground(t.TextMuted()).
				Render(fmt.Sprintf(" %s (%s)", string(msg.Model), "canceled")),
			)
		case message.FinishReasonError:
			info = append(info, baseStyle.
				Width(width-1).
				Foreground(t.TextMuted()).
				Render(fmt.Sprintf(" %s (%s)", string(msg.Model), "error")),
			)
		case message.FinishReasonPermissionDenied:
			info = append(info, baseStyle.
				Width(width-1).
				Foreground(t.TextMuted()).
				Render(fmt.Sprintf(" %s (%s)", string(msg.Model), "permission denied")),
			)
		}
	}
	if content != "" || (finished && finishData.Reason == message.FinishReasonEndTurn) {
		if content == "" {
			content = "*Finished without output*"
		}
		if isSummary {
			info = append(info, baseStyle.Width(width-1).Foreground(t.TextMuted()).Render(" (summary)"))
		}

		// Skip Glamour markdown while the message is still streaming — each
		// token update invalidates the render cache, and Glamour on multi-KB
		// content blocks the Bubbletea Update loop for seconds. Plain text is
		// ~instant. On FinishReasonEndTurn, list.go kicks an async goroutine
		// that re-enters this function with useMarkdown=true and posts the
		// rendered result back as a markdownRenderedMsg — keeping Glamour off
		// the Update goroutine entirely.
		body := renderMessageBody(content, false, true, width, role, useMarkdown, info...)
		if label := roleLabel(role, width-1); label != "" {
			content = lipgloss.JoinVertical(lipgloss.Left, label, body)
		} else {
			content = body
		}
		messages = append(messages, uiMessage{
			ID:          msg.ID,
			messageType: assistantMessageType,
			position:    position,
			height:      lipgloss.Height(content),
			content:     content,
		})
		position += messages[0].height
		position++ // for the space
	} else if thinking && thinkingContent != "" {
		// Thinking/reasoning tokens stream in real time — same Glamour-cost
		// issue as regular streaming content above. Plain text until it
		// settles.
		content = renderMessageBody(thinkingContent, false, msg.ID == focusedUIMessageId, width, role, false)
	}

	for i, toolCall := range msg.ToolCalls() {
		toolCallContent := renderToolMessage(
			toolCall,
			allMessages,
			messagesService,
			focusedUIMessageId,
			false,
			width,
			i+1,
		)
		messages = append(messages, toolCallContent)
		position += toolCallContent.height
		position++ // for the space
	}
	return messages
}

func findToolResponse(toolCallID string, futureMessages []message.Message) *message.ToolResult {
	for _, msg := range futureMessages {
		for _, result := range msg.ToolResults() {
			if result.ToolCallID == toolCallID {
				return &result
			}
		}
	}
	return nil
}

func toolName(name string) string {
	switch name {
	case agent.AgentToolName:
		return "Task"
	case tools.BashToolName:
		return "Bash"
	case tools.EditToolName:
		return "Edit"
	case tools.FetchToolName:
		return "Fetch"
	case tools.GlobToolName:
		return "Glob"
	case tools.GrepToolName:
		return "Grep"
	case tools.LSToolName:
		return "List"
	case tools.SourcegraphToolName:
		return "Sourcegraph"
	case tools.ViewToolName:
		return "View"
	case tools.WriteToolName:
		return "Write"
	case tools.PatchToolName:
		return "Patch"
	}
	return name
}

func getToolAction(name string) string {
	switch name {
	case agent.AgentToolName:
		return "Preparing prompt..."
	case tools.BashToolName:
		return "Building command..."
	case tools.EditToolName:
		return "Preparing edit..."
	case tools.FetchToolName:
		return "Writing fetch..."
	case tools.GlobToolName:
		return "Finding files..."
	case tools.GrepToolName:
		return "Searching content..."
	case tools.LSToolName:
		return "Listing directory..."
	case tools.SourcegraphToolName:
		return "Searching code..."
	case tools.ViewToolName:
		return "Reading file..."
	case tools.WriteToolName:
		return "Preparing write..."
	case tools.PatchToolName:
		return "Preparing patch..."
	}
	return "Working..."
}

// renders params, params[0] (params[1]=params[2] ....)
func renderParams(paramsWidth int, params ...string) string {
	if len(params) == 0 {
		return ""
	}
	mainParam := params[0]
	if len(mainParam) > paramsWidth {
		mainParam = mainParam[:paramsWidth-3] + "..."
	}

	if len(params) == 1 {
		return mainParam
	}
	otherParams := params[1:]
	// create pairs of key/value
	// if odd number of params, the last one is a key without value
	if len(otherParams)%2 != 0 {
		otherParams = append(otherParams, "")
	}
	parts := make([]string, 0, len(otherParams)/2)
	for i := 0; i < len(otherParams); i += 2 {
		key := otherParams[i]
		value := otherParams[i+1]
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	partsRendered := strings.Join(parts, ", ")
	remainingWidth := paramsWidth - lipgloss.Width(partsRendered) - 5 // for the space
	if remainingWidth < 30 {
		// No space for the params, just show the main
		return mainParam
	}

	if len(parts) > 0 {
		mainParam = fmt.Sprintf("%s (%s)", mainParam, strings.Join(parts, ", "))
	}

	return ansi.Truncate(mainParam, paramsWidth, "...")
}

func removeWorkingDirPrefix(path string) string {
	wd := config.WorkingDirectory()
	if strings.HasPrefix(path, wd) {
		path = strings.TrimPrefix(path, wd)
	}
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	if strings.HasPrefix(path, "./") {
		path = strings.TrimPrefix(path, "./")
	}
	if strings.HasPrefix(path, "../") {
		path = strings.TrimPrefix(path, "../")
	}
	return path
}

func renderToolParams(paramWidth int, toolCall message.ToolCall) string {
	params := ""
	switch toolCall.Name {
	case agent.AgentToolName:
		var params agent.AgentParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		prompt := strings.ReplaceAll(params.Prompt, "\n", " ")
		return renderParams(paramWidth, prompt)
	case tools.BashToolName:
		var params tools.BashParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		command := strings.ReplaceAll(params.Command, "\n", " ")
		return renderParams(paramWidth, command)
	case tools.EditToolName:
		var params tools.EditParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		filePath := removeWorkingDirPrefix(params.FilePath)
		return renderParams(paramWidth, filePath)
	case tools.FetchToolName:
		var params tools.FetchParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		url := params.URL
		toolParams := []string{
			url,
		}
		if params.Format != "" {
			toolParams = append(toolParams, "format", params.Format)
		}
		if params.Timeout != 0 {
			toolParams = append(toolParams, "timeout", (time.Duration(params.Timeout) * time.Second).String())
		}
		return renderParams(paramWidth, toolParams...)
	case tools.GlobToolName:
		var params tools.GlobParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		pattern := params.Pattern
		toolParams := []string{
			pattern,
		}
		if params.Path != "" {
			toolParams = append(toolParams, "path", params.Path)
		}
		return renderParams(paramWidth, toolParams...)
	case tools.GrepToolName:
		var params tools.GrepParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		pattern := params.Pattern
		toolParams := []string{
			pattern,
		}
		if params.Path != "" {
			toolParams = append(toolParams, "path", params.Path)
		}
		if params.Include != "" {
			toolParams = append(toolParams, "include", params.Include)
		}
		if params.LiteralText {
			toolParams = append(toolParams, "literal", "true")
		}
		return renderParams(paramWidth, toolParams...)
	case tools.LSToolName:
		var params tools.LSParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		path := params.Path
		if path == "" {
			path = "."
		}
		return renderParams(paramWidth, path)
	case tools.SourcegraphToolName:
		var params tools.SourcegraphParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		return renderParams(paramWidth, params.Query)
	case tools.ViewToolName:
		var params tools.ViewParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		filePath := removeWorkingDirPrefix(params.FilePath)
		toolParams := []string{
			filePath,
		}
		if params.Limit != 0 {
			toolParams = append(toolParams, "limit", fmt.Sprintf("%d", params.Limit))
		}
		if params.Offset != 0 {
			toolParams = append(toolParams, "offset", fmt.Sprintf("%d", params.Offset))
		}
		return renderParams(paramWidth, toolParams...)
	case tools.WriteToolName:
		var params tools.WriteParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		filePath := removeWorkingDirPrefix(params.FilePath)
		return renderParams(paramWidth, filePath)
	default:
		input := strings.ReplaceAll(toolCall.Input, "\n", " ")
		params = renderParams(paramWidth, input)
	}
	return params
}

func truncateHeight(content string, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		return strings.Join(lines[:height], "\n")
	}
	return content
}

func renderToolResponse(toolCall message.ToolCall, response message.ToolResult, width int) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	if response.IsError {
		errContent := fmt.Sprintf("Error: %s", strings.ReplaceAll(response.Content, "\n", " "))
		errContent = ansi.Truncate(errContent, width-1, "...")
		return baseStyle.
			Width(width).
			Foreground(t.Error()).
			Render(errContent)
	}

	resultContent := truncateHeight(response.Content, maxResultHeight)

	// Tool results use plain-text rendering. Glamour syntax highlighting on
	// every tool result (bash output, file views, fetch bodies) was a
	// cumulative freeze source — each Chroma lexer load is 100-500ms the
	// first time (bash, text, go, json, etc. each have their own lexer),
	// and a single assistant message with N tool calls re-renders all N
	// through Glamour when cache invalidates. 12 tool calls in a Planner
	// turn measured at 32 seconds for one render pass. Plain text with the
	// existing border/width treatment keeps results readable without the
	// Chroma cost. See KI-01.
	toolTextStyle := baseStyle.Width(width).Foreground(t.TextMuted())

	switch toolCall.Name {
	case agent.AgentToolName:
		return toolTextStyle.Render(resultContent)
	case tools.BashToolName:
		return toolTextStyle.Render(resultContent)
	case tools.EditToolName:
		metadata := tools.EditResponseMetadata{}
		json.Unmarshal([]byte(response.Metadata), &metadata)
		truncDiff := truncateHeight(metadata.Diff, maxResultHeight)
		formattedDiff, _ := diff.FormatDiff(truncDiff, diff.WithTotalWidth(width))
		return formattedDiff
	case tools.FetchToolName:
		return toolTextStyle.Render(resultContent)
	case tools.GlobToolName:
		return toolTextStyle.Render(resultContent)
	case tools.GrepToolName:
		return toolTextStyle.Render(resultContent)
	case tools.LSToolName:
		return toolTextStyle.Render(resultContent)
	case tools.SourcegraphToolName:
		return toolTextStyle.Render(resultContent)
	case tools.ViewToolName:
		metadata := tools.ViewResponseMetadata{}
		json.Unmarshal([]byte(response.Metadata), &metadata)
		return toolTextStyle.Render(truncateHeight(metadata.Content, maxResultHeight))
	case tools.WriteToolName:
		params := tools.WriteParams{}
		json.Unmarshal([]byte(toolCall.Input), &params)
		return toolTextStyle.Render(truncateHeight(params.Content, maxResultHeight))
	default:
		return toolTextStyle.Render(resultContent)
	}
}

func renderToolMessage(
	toolCall message.ToolCall,
	allMessages []message.Message,
	messagesService message.Service,
	focusedUIMessageId string,
	nested bool,
	width int,
	position int,
) uiMessage {
	if nested {
		width = width - 3
	}

	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	style := baseStyle.
		Width(width).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1).
		BorderForeground(t.TextMuted())

	response := findToolResponse(toolCall.ID, allMessages)
	toolNameText := baseStyle.Foreground(t.TextMuted()).
		Render(fmt.Sprintf("%s: ", toolName(toolCall.Name)))

	if !toolCall.Finished {
		// Get a brief description of what the tool is doing
		toolAction := getToolAction(toolCall.Name)

		progressText := baseStyle.
			Width(width - 2 - lipgloss.Width(toolNameText)).
			Foreground(t.TextMuted()).
			Render(fmt.Sprintf("%s", toolAction))

		content := style.Render(lipgloss.JoinHorizontal(lipgloss.Left, toolNameText, progressText))
		toolMsg := uiMessage{
			messageType: toolMessageType,
			position:    position,
			height:      lipgloss.Height(content),
			content:     content,
		}
		return toolMsg
	}

	params := renderToolParams(width-2-lipgloss.Width(toolNameText), toolCall)
	responseContent := ""
	if response != nil {
		responseContent = renderToolResponse(toolCall, *response, width-2)
		responseContent = strings.TrimSuffix(responseContent, "\n")
	} else {
		responseContent = baseStyle.
			Italic(true).
			Width(width - 2).
			Foreground(t.TextMuted()).
			Render("Waiting for response...")
	}

	parts := []string{}
	if !nested {
		formattedParams := baseStyle.
			Width(width - 2 - lipgloss.Width(toolNameText)).
			Foreground(t.TextMuted()).
			Render(params)

		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Left, toolNameText, formattedParams))
	} else {
		prefix := baseStyle.
			Foreground(t.TextMuted()).
			Render(" └ ")
		formattedParams := baseStyle.
			Width(width - 2 - lipgloss.Width(toolNameText)).
			Foreground(t.TextMuted()).
			Render(params)
		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Left, prefix, toolNameText, formattedParams))
	}

	if toolCall.Name == agent.AgentToolName {
		taskMessages, _ := messagesService.List(context.Background(), toolCall.ID)
		toolCalls := []message.ToolCall{}
		for _, v := range taskMessages {
			toolCalls = append(toolCalls, v.ToolCalls()...)
		}
		for _, call := range toolCalls {
			rendered := renderToolMessage(call, []message.Message{}, messagesService, focusedUIMessageId, true, width, 0)
			parts = append(parts, rendered.content)
		}
	}
	if responseContent != "" && !nested {
		parts = append(parts, responseContent)
	}

	content := style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		),
	)
	if nested {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		)
	}
	toolMsg := uiMessage{
		messageType: toolMessageType,
		position:    position,
		height:      lipgloss.Height(content),
		content:     content,
	}
	return toolMsg
}

// Helper function to format the time difference between two Unix timestamps
func formatTimestampDiff(start, end int64) string {
	diffSeconds := float64(end-start) / 1000.0 // Convert to seconds
	if diffSeconds < 1 {
		return fmt.Sprintf("%dms", int(diffSeconds*1000))
	}
	if diffSeconds < 60 {
		return fmt.Sprintf("%.1fs", diffSeconds)
	}
	return fmt.Sprintf("%.1fm", diffSeconds/60)
}

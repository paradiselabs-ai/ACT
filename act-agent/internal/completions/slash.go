package completions

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/dialog"
)

type SlashCommandItem struct {
	Command     string
	Description string
}

var DefaultSlashCommands = []SlashCommandItem{
	{"/plan", "Plan a task with Tier 1 Planner"},
	{"/run", "Execute a task directly"},
	{"/model", "Select active model config / role"},
	{"/swarm", "Run task with Swarm (Tier 2 agents)"},
	{"/status", "System & agent status overview"},
	{"/help", "Show keyboard shortcuts & help"},
	{"/log", "View recent ChronLog entries"},
	{"/tasks", "Show unverified task graph"},
	{"/validation", "Check pending validation queue"},
	{"/conflicts", "View file-lock conflicts"},
	{"/compact", "Summarize session & start compact one"},
	{"/init", "Create or update project ACT.md"},
	{"/clear", "Clear chat session"},
	{"/backend", "List/Set Tier 1 role backends"},
	{"/role", "Select active model config / role"},
	{"@planner", "Direct request to Planner agent"},
}

type slashCommandsContextGroup struct{}

func (s *slashCommandsContextGroup) GetId() string {
	return "slash"
}

func (s *slashCommandsContextGroup) GetEntry() dialog.CompletionItemI {
	return dialog.NewCompletionItem(dialog.CompletionItem{
		Title: "Slash Commands",
		Value: "slash",
	})
}

func (s *slashCommandsContextGroup) GetChildEntries(query string) ([]dialog.CompletionItemI, error) {
	if strings.Contains(query, " ") {
		return nil, nil
	}

	var candidates []string
	cmdMap := make(map[string]SlashCommandItem)

	for _, item := range DefaultSlashCommands {
		candidates = append(candidates, item.Command)
		cmdMap[item.Command] = item
	}

	var matches []string
	if query == "" {
		matches = candidates
	} else {
		searchQuery := query
		if searchQuery != "" && searchQuery[0] != '/' && searchQuery[0] != '@' {
			searchQuery = "/" + searchQuery
		}
		matches = fuzzy.Find(searchQuery, candidates)
	}

	items := make([]dialog.CompletionItemI, 0, len(matches))
	for _, match := range matches {
		item := cmdMap[match]
		ci := dialog.NewCompletionItem(dialog.CompletionItem{
			Title:       item.Command,
			Value:       item.Command + " ",
			Description: item.Description,
		})
		items = append(items, ci)
	}

	return items, nil
}

func NewSlashCommandsContextGroup() dialog.CompletionProvider {
	return &slashCommandsContextGroup{}
}

// Package act provides a native HTTP client for the ACT coordination server.
// Replaces the previous shell-out implementation (which ran npx tsx cli/act-cli.ts).
package act

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

// Client communicates with the ACT coordination server via HTTP.
type Client struct {
	ServerURL  string
	AgentID    string
	Project    string
	httpClient *http.Client
}

// NewClient creates a new ACT client with native HTTP.
func NewClient(agentID, project string) *Client {
	serverURL := os.Getenv("ACT_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	serverURL = strings.TrimRight(serverURL, "/")

	return &Client{
		ServerURL: serverURL,
		AgentID:   agentID,
		Project:   project,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Register registers this agent with the ACT coordination server.
func (c *Client) Register() error {
	if c.Project == "" {
		return fmt.Errorf("act register: Client.Project is required (per-project isolation)")
	}
	body := map[string]any{
		"agentId":      c.AgentID,
		"name":         c.AgentID,
		"projectName":  c.Project,
		"capabilities": []string{"code", "coordination"},
	}
	resp, err := c.post("/api/agents/register", body)
	if err != nil {
		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "already registered") {
			logging.Info("Agent already registered", "agent_id", c.AgentID)
			return nil
		}
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetContext fetches the full agent context (brief, task, parallel agents, messages).
func (c *Client) GetContext() (string, error) {
	var sections []string

	// Fetch brief, assigned task, all tasks, agents, locks, messages in parallel
	type result struct {
		key string
		val string
		err error
	}
	ch := make(chan result, 6)

	go func() {
		brief, err := c.getBrief()
		ch <- result{"brief", brief, err}
	}()
	go func() {
		task, err := c.getAssignedTask()
		ch <- result{"task", task, err}
	}()
	go func() {
		tasks, err := c.ListTasks()
		ch <- result{"tasks", tasks, err}
	}()
	go func() {
		agents, err := c.ListAgents()
		ch <- result{"agents", agents, err}
	}()
	go func() {
		locks, err := c.GetFileLocks()
		ch <- result{"locks", locks, err}
	}()
	go func() {
		msgs, err := c.getMessages(5)
		ch <- result{"messages", msgs, err}
	}()

	results := make(map[string]string)
	for i := 0; i < 6; i++ {
		r := <-ch
		if r.err == nil && r.val != "" {
			results[r.key] = r.val
		}
	}

	if v := results["brief"]; v != "" {
		sections = append(sections, "## Agent Brief\n"+v)
	}
	if v := results["task"]; v != "" {
		sections = append(sections, "## Current Task\n"+v)
	}
	if v := results["agents"]; v != "" {
		sections = append(sections, "## Agents\n"+v)
	}
	if v := results["tasks"]; v != "" {
		sections = append(sections, "## Tasks\n"+v)
	}
	if v := results["locks"]; v != "" {
		sections = append(sections, "## File Locks\n"+v)
	}
	if v := results["messages"]; v != "" {
		sections = append(sections, "## Messages\n"+v)
	}

	return strings.Join(sections, "\n\n"), nil
}

// ReportProgress updates task progress percentage.
func (c *Client) ReportProgress(taskID string, percent int) error {
	body := map[string]any{
		"agentId":  c.AgentID,
		"progress": percent,
		"status":   "in_progress",
	}
	resp, err := c.post(fmt.Sprintf("/api/tasks/%s/progress", url.PathEscape(taskID)), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ReportComplete marks a task as complete with a result summary.
func (c *Client) ReportComplete(taskID string, result string) error {
	body := map[string]any{
		"agentId": c.AgentID,
		"success": true,
		"result":  result,
	}
	resp, err := c.post(fmt.Sprintf("/api/tasks/%s/complete", url.PathEscape(taskID)), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ClaimFiles claims exclusive editing rights on files.
func (c *Client) ClaimFiles(taskID string, files []string) error {
	body := map[string]any{
		"agent_id":     c.AgentID,
		"task_id":      taskID,
		"project_name": c.Project,
		"file_paths":   files,
	}
	resp, err := c.post("/api/files/claim", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ReleaseFiles releases file locks.
func (c *Client) ReleaseFiles(files []string) error {
	body := map[string]any{
		"agent_id":     c.AgentID,
		"project_name": c.Project,
		"file_paths":   files,
	}
	resp, err := c.post("/api/files/release", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SendMessage sends a coordination message to other agents.
func (c *Client) SendMessage(text string) error {
	body := map[string]any{
		"sender":      c.AgentID,
		"projectName": c.Project,
		"message":     text,
	}
	resp, err := c.post("/api/messages", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SubmitForValidation submits a completed task for Assurance review.
func (c *Client) SubmitForValidation(taskID string) error {
	body := map[string]any{
		"agentId": c.AgentID,
	}
	resp, err := c.post(fmt.Sprintf("/api/tasks/%s/submit-for-validation", url.PathEscape(taskID)), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// Status fetches the ACT system status as a formatted string.
func (c *Client) Status() (string, error) {
	return c.getString("/api/status")
}

// PVMSearch searches coordination memory for relevant patterns.
func (c *Client) PVMSearch(query string) (string, error) {
	path := fmt.Sprintf("/api/pvm/search?query=%s&limit=10", url.QueryEscape(query))
	return c.getString(path)
}

// IsAvailable returns true if the ACT server is reachable.
func (c *Client) IsAvailable() bool {
	resp, err := c.httpClient.Get(c.ServerURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// --- internal helpers ---

func (c *Client) getBrief() (string, error) {
	path := fmt.Sprintf("/api/projects/%s/briefs/%s",
		url.PathEscape(c.Project), url.PathEscape(c.AgentID))
	data, err := c.getJSON(path)
	if err != nil {
		return "", err
	}
	if content, ok := data["content"].(string); ok {
		return content, nil
	}
	return "", nil
}

func (c *Client) getAssignedTask() (string, error) {
	path := fmt.Sprintf("/api/tasks/assigned?agent_id=%s", url.QueryEscape(c.AgentID))
	if c.Project != "" {
		path += "&project=" + url.QueryEscape(c.Project)
	}
	return c.getString(path)
}

func (c *Client) getMessages(limit int) (string, error) {
	path := fmt.Sprintf("/api/agents/%s/messages?limit=%d",
		url.PathEscape(c.AgentID), limit)
	if c.Project != "" {
		path += "&project=" + url.QueryEscape(c.Project)
	}
	return c.getString(path)
}

func (c *Client) post(path string, body map[string]any) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.ServerURL+path,
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		logging.Warn("ACT HTTP request failed", "path", path, "error", err)
		return nil, fmt.Errorf("act POST %s: %w", path, err)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		logging.Warn("ACT HTTP error", "path", path, "status", resp.StatusCode, "body", string(bodyBytes))
		return nil, fmt.Errorf("act POST %s: HTTP %d: %s", path, resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

func (c *Client) getJSON(path string) (map[string]any, error) {
	resp, err := c.httpClient.Get(c.ServerURL + path)
	if err != nil {
		return nil, fmt.Errorf("act GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("act GET %s: HTTP %d", path, resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("act GET %s: decode: %w", path, err)
	}
	return data, nil
}

func (c *Client) getString(path string) (string, error) {
	resp, err := c.httpClient.Get(c.ServerURL + path)
	if err != nil {
		return "", fmt.Errorf("act GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("act GET %s: HTTP %d", path, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("act GET %s: read: %w", path, err)
	}
	return strings.TrimSpace(string(bodyBytes)), nil
}

// CreateTask creates a new task on the ACT server.
// Used by the Planner agent to dispatch work to swarm agents.
func (c *Client) CreateTask(title, description string, requiredCapabilities []string, priority string, dependencies []string, metadata map[string]any) (string, error) {
	body := map[string]any{
		"title":                title,
		"description":          description,
		"requiredCapabilities": requiredCapabilities,
		"priority":             priority,
		"dependencies":         dependencies,
		"metadata":             metadata,
	}
	resp, err := c.post("/api/tasks", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("decode create task response: %w", err)
	}
	return data.Task.ID, nil
}

// SetTaskDependencies replaces a task's dependency list. Used by the
// orchestrator's two-pass CREATE_TASK dispatch — pass 1 creates every task
// without dependencies and collects server-assigned IDs; pass 2 resolves
// each Planner-emitted title-string dependency to its corresponding ID and
// PATCHes the task to use IDs. Without this, dependencies referenced by
// title (the only thing the Planner can emit, since IDs don't exist before
// creation) never match anything in the server's task map and the dependent
// tasks sit in `pending` forever.
func (c *Client) SetTaskDependencies(taskID string, dependencyIDs []string) error {
	body := map[string]any{
		"dependencies": dependencyIDs,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal dependencies: %w", err)
	}
	req, err := http.NewRequest("PATCH", c.ServerURL+"/api/tasks/"+url.PathEscape(taskID)+"/dependencies", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("act PATCH dependencies: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("act PATCH dependencies: HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

// RetryTask resets a failed task to pending and increments its retryCount.
// Returns an error if the task is not failed, doesn't exist, or has exceeded
// MAX_TASK_RETRIES (server returns 409 permanentlyFailed=true).
func (c *Client) RetryTask(taskID string) error {
	req, err := http.NewRequest("POST", c.ServerURL+"/api/tasks/"+url.PathEscape(taskID)+"/retry", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("act POST retry: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("act POST retry: HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

// AbandonTask marks a task permanently failed with metadata.abandoned=true.
// Distinct from RetryTask — abandoned tasks are not re-dispatched. Used by
// the Planner when a task is unrecoverable. Reason is required for audit.
func (c *Client) AbandonTask(taskID, reason string) error {
	body := map[string]any{"reason": reason}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal abandon body: %w", err)
	}
	req, err := http.NewRequest("POST", c.ServerURL+"/api/tasks/"+url.PathEscape(taskID)+"/abandon", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("act POST abandon: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("act POST abandon: HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

// ListTasks fetches tasks from the ACT server. Scoped to the client's
// current project when set — without scoping, Observer's status snapshot
// sees cross-project tasks and flags them as stuck/failed anomalies in
// the current session.
//
// Returns the raw JSON string for the orchestrator to parse.
// (The orchestrator will use orchestrator_types.go structs to decode this.)
func (c *Client) ListTasks() (string, error) {
	path := "/api/tasks"
	if c.Project != "" {
		path += "?project=" + url.QueryEscape(c.Project)
	}
	return c.getString(path)
}

// GetPendingValidation fetches tasks awaiting Assurance validation. Scoped
// to the client's current project so Assurance only sees work from the
// project the TUI is attached to — without scoping, tasks from prior
// sessions in other directories leak into this one's validation queue.
func (c *Client) GetPendingValidation() (string, error) {
	path := "/api/tasks/pending-validation"
	if c.Project != "" {
		path += "?project=" + url.QueryEscape(c.Project)
	}
	return c.getString(path)
}

// GetValidatedTasks fetches tasks that have passed Assurance and are awaiting
// QA synthesis. Scoped to the client's current project for the same reason
// as GetPendingValidation — prevents cross-project QA runs.
func (c *Client) GetValidatedTasks() (string, error) {
	path := "/api/tasks/validated"
	if c.Project != "" {
		path += "?project=" + url.QueryEscape(c.Project)
	}
	return c.getString(path)
}

// ListAgents fetches registered agents from the ACT server. Scoped to the
// client's current project when set — cross-project agents are invisible.
func (c *Client) ListAgents() (string, error) {
	path := "/api/agents"
	if c.Project != "" {
		path += "?project=" + url.QueryEscape(c.Project)
	}
	return c.getString(path)
}

// GetFileLocks fetches current file locks. Scoped to the client's current
// project when set — same absolute path in another project is a separate lock.
func (c *Client) GetFileLocks() (string, error) {
	path := "/api/files/locks"
	if c.Project != "" {
		path += "?project=" + url.QueryEscape(c.Project)
	}
	return c.getString(path)
}

// GetLog fetches recent coordination log entries. Scoped to the client's
// current project when set — without scoping, the orchestrator's
// coordinationEventLoop surfaces cross-project task events into the current
// session's chat.
//
// limit is the maximum number of entries to return.
func (c *Client) GetLog(limit int) (string, error) {
	path := fmt.Sprintf("/api/log?limit=%d", limit)
	if c.Project != "" {
		path += "&project=" + url.QueryEscape(c.Project)
	}
	return c.getString(path)
}

// GetProject fetches a single project by name. Returns (data, true) on 200,
// (nil, false) on 404, error on other failures. Used by the orchestrator at
// startup to detect whether intake mode should fire.
func (c *Client) GetProject(name string) (map[string]any, bool, error) {
	path := fmt.Sprintf("/api/projects/%s", url.PathEscape(name))
	resp, err := c.httpClient.Get(c.ServerURL + path)
	if err != nil {
		return nil, false, fmt.Errorf("act GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, false, nil
	}
	if resp.StatusCode >= 400 {
		return nil, false, fmt.Errorf("act GET %s: HTTP %d", path, resp.StatusCode)
	}
	// Server wraps the project in {success, project: {...}}. Callers expect
	// the inner shape (description, techStack, etc. at top level).
	var wrapper struct {
		Success bool           `json:"success"`
		Project map[string]any `json:"project"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, false, fmt.Errorf("decode project: %w", err)
	}
	if !wrapper.Success || wrapper.Project == nil {
		return nil, false, nil
	}
	return wrapper.Project, true, nil
}

// CreateProject POSTs a new project to the ACT server. Called by the
// orchestrator after the Planner emits PROJECT_BRIEF during intake mode.
func (c *Client) CreateProject(name, workspace, description, techStack, constraints, successCriteria string, agents []string) error {
	body := map[string]any{
		"name":            name,
		"workspace":       workspace,
		"description":     description,
		"techStack":       techStack,
		"constraints":     constraints,
		"successCriteria": successCriteria,
		"agents":          agents,
	}
	resp, err := c.post("/api/projects", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SubmitVerdict submits an Assurance validation verdict for a task.
// Used by the orchestrator after the Assurance agent produces a verdict.
func (c *Client) SubmitVerdict(taskID, agentID string, passed bool, score int, criteriaResults []map[string]any, gaps, feedback string) error {
	body := map[string]any{
		"agentId":         agentID,
		"passed":          passed,
		"score":           score,
		"criteriaResults": criteriaResults,
		"gaps":            gaps,
		"feedback":        feedback,
	}
	resp, err := c.post(fmt.Sprintf("/api/tasks/%s/validation-verdict", url.PathEscape(taskID)), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SubmitSynthesis records a QA/Synthesizer outcome for a validated task.
// kind is "complete" or "need_clarification"; targetAgent + question only
// apply to the clarification path. Called from the orchestrator's QA poller
// once parseSynthesisResponse resolves the agent's reply.
func (c *Client) SubmitSynthesis(taskID, agentID, kind, summary, targetAgent, question string) error {
	body := map[string]any{
		"agentId":     agentID,
		"kind":        kind,
		"summary":     summary,
		"targetAgent": targetAgent,
		"question":    question,
	}
	resp, err := c.post(fmt.Sprintf("/api/tasks/%s/synthesis", url.PathEscape(taskID)), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

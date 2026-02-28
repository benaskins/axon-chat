package chat

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TaskRunner submits tasks to the task runner service.
type TaskRunner interface {
	SubmitTask(ctx context.Context, req *TaskSubmitRequest) (*TaskSubmission, error)
	GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error)
	ListTasks(ctx context.Context, agentSlug string, limit, offset int) ([]TaskStatus, error)
	IssueAgentCert(ctx context.Context, slug, username string) error
}

// TaskSubmitRequest is the unified request body for POST /api/tasks.
type TaskSubmitRequest struct {
	Type   string `json:"type"`   // "claude_session" | "image_generation"
	Params any    `json:"params"` // type-specific payload
}

// NewClaudeSessionRequest creates a TaskSubmitRequest for a claude_session task.
func NewClaudeSessionRequest(description, requestedBy, username string) *TaskSubmitRequest {
	return &TaskSubmitRequest{
		Type: "claude_session",
		Params: map[string]string{
			"description":  description,
			"requested_by": requestedBy,
			"username":     username,
		},
	}
}

// NewImageTaskRequest creates a TaskSubmitRequest for an image_generation task.
func NewImageTaskRequest(params *ImageTaskSubmission) *TaskSubmitRequest {
	return &TaskSubmitRequest{
		Type:   "image_generation",
		Params: params,
	}
}

// TaskSubmission is the response from submitting a new task.
type TaskSubmission struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// ImageTaskSubmission holds the parameters for an image generation task.
type ImageTaskSubmission struct {
	Prompt         string `json:"prompt"`
	ReferenceImage string `json:"reference_image,omitempty"` // base64-encoded PNG
	AgentSlug      string `json:"agent_slug"`
	UserID         string `json:"user_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	ImageID        string `json:"image_id"`
	Private        bool   `json:"private,omitempty"`
}

// TaskStatus is the response from checking a task's progress.
type TaskStatus struct {
	TaskID      string  `json:"task_id"`
	Type        string  `json:"type,omitempty"`
	Status      string  `json:"status"`
	Description string  `json:"description,omitempty"`
	Summary     string  `json:"result_summary,omitempty"`
	Error       string  `json:"error,omitempty"`
	ArtifactID  string  `json:"artifact_id,omitempty"`
	RequestedBy string  `json:"requested_by,omitempty"`
	CreatedAt   *string `json:"created_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
}

// TaskRunnerClient is an HTTP client for the task runner service.
type TaskRunnerClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewTaskRunnerClient creates a client pointing at the task runner service.
// Pass a non-nil tlsConfig to enable mTLS; nil uses default (plain HTTP for tests).
func NewTaskRunnerClient(baseURL string, tlsConfig *tls.Config) *TaskRunnerClient {
	client := &http.Client{Timeout: 10 * time.Second}
	if tlsConfig != nil {
		client.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	}
	return &TaskRunnerClient{
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (c *TaskRunnerClient) SubmitTask(ctx context.Context, submitReq *TaskSubmitRequest) (*TaskSubmission, error) {
	body, err := json.Marshal(submitReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/tasks", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("task runner request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("task runner returned %d: %s", resp.StatusCode, errResp["error"])
	}

	var sub TaskSubmission
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &sub, nil
}

func (c *TaskRunnerClient) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tasks/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("task runner request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task runner returned %d", resp.StatusCode)
	}

	var status TaskStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &status, nil
}

func (c *TaskRunnerClient) ListTasks(ctx context.Context, agentSlug string, limit, offset int) ([]TaskStatus, error) {
	reqURL := fmt.Sprintf("%s/api/tasks?agent=%s&limit=%d&offset=%d", c.baseURL, agentSlug, limit, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("task runner request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task runner returned %d", resp.StatusCode)
	}

	var tasks []TaskStatus
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return tasks, nil
}

func (c *TaskRunnerClient) IssueAgentCert(ctx context.Context, slug, username string) error {
	body, err := json.Marshal(map[string]string{
		"slug":     slug,
		"username": username,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/agent-certs", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("task runner request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("cert issuance returned %d: %s", resp.StatusCode, errResp["error"])
	}

	return nil
}

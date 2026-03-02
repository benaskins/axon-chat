package chat

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/benaskins/axon"
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

// TaskSubmission is the response from submitting a new task.
type TaskSubmission struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
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
	client *axon.InternalClient
}

// NewTaskRunnerClient creates a client pointing at the task runner service.
// Pass a non-nil tlsConfig to enable mTLS; nil uses default (plain HTTP for tests).
func NewTaskRunnerClient(baseURL string, tlsConfig *tls.Config) *TaskRunnerClient {
	c := axon.NewInternalClient(baseURL)
	if tlsConfig != nil {
		c.HTTPClient = &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
		}
	}
	return &TaskRunnerClient{client: c}
}

func (c *TaskRunnerClient) SubmitTask(ctx context.Context, submitReq *TaskSubmitRequest) (*TaskSubmission, error) {
	var sub TaskSubmission
	if err := c.client.Post(ctx, "/api/tasks", submitReq, &sub); err != nil {
		return nil, fmt.Errorf("submit task: %w", err)
	}
	return &sub, nil
}

func (c *TaskRunnerClient) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	var status TaskStatus
	if err := c.client.Get(ctx, "/api/tasks/"+taskID, &status); err != nil {
		return nil, fmt.Errorf("get task status: %w", err)
	}
	return &status, nil
}

func (c *TaskRunnerClient) ListTasks(ctx context.Context, agentSlug string, limit, offset int) ([]TaskStatus, error) {
	path := fmt.Sprintf("/api/tasks?agent=%s&limit=%d&offset=%d", agentSlug, limit, offset)
	var tasks []TaskStatus
	if err := c.client.Get(ctx, path, &tasks); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

func (c *TaskRunnerClient) IssueAgentCert(ctx context.Context, slug, username string) error {
	body := map[string]string{"slug": slug, "username": username}
	if err := c.client.Post(ctx, "/api/agent-certs", body, nil); err != nil {
		return fmt.Errorf("issue agent cert: %w", err)
	}
	return nil
}

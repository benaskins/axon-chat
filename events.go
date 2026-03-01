package chat

// Event wraps different async event types delivered via SSE.
type Event struct {
	Type string     `json:"type"`
	Task *TaskEvent `json:"task,omitempty"`
}

// TaskEvent is sent when an async code change task completes or fails.
type TaskEvent struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
	Error   string `json:"error,omitempty"`
}

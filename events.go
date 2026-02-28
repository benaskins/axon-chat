package chat

// Event wraps different async event types delivered via SSE.
type Event struct {
	Type  string      `json:"type"`
	Image *ImageEvent `json:"image,omitempty"`
	Task  *TaskEvent  `json:"task,omitempty"`
}

// ImageEvent is sent when an async image generation completes or fails.
type ImageEvent struct {
	ImageID      string `json:"image_id"`
	ImageURL     string `json:"image_url,omitempty"`
	NSFWDetected bool   `json:"nsfw_detected,omitempty"`
	ImageError   string `json:"image_error,omitempty"`
}

// TaskEvent is sent when an async code change task completes or fails.
type TaskEvent struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
	Error   string `json:"error,omitempty"`
}

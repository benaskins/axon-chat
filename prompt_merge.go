package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	ollamaapi "github.com/ollama/ollama/api"
)

// ImageGenConfig holds configuration for image generation baseline rules.
type ImageGenConfig struct {
	BaselinePrompt        string `json:"baseline_prompt"`
	PrivateBaselinePrompt string `json:"private_baseline_prompt"`
	MergeModel            string `json:"merge_model"`
	MergeInstruction      string `json:"merge_instruction"`
	PrivateMergeInstr     string `json:"private_merge_instruction"`
}

// LoadImageGenConfig loads the image generation config from a JSON file.
func LoadImageGenConfig(path string) (*ImageGenConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config ImageGenConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &config, nil
}

// PromptMerger intelligently merges baseline rules with agent context and scene prompts.
type PromptMerger struct {
	client ChatClient
	config *ImageGenConfig
}

// NewPromptMerger creates a new prompt merger.
func NewPromptMerger(client ChatClient, config *ImageGenConfig) *PromptMerger {
	return &PromptMerger{
		client: client,
		config: config,
	}
}

// MergePrompt intelligently merges baseline, agent system prompt, recent conversation,
// and scene into a single image generation prompt.
func (m *PromptMerger) MergePrompt(systemPrompt string, recentMessages []Message, scene string) (string, error) {
	return m.mergePromptWith(m.config.MergeInstruction, m.config.BaselinePrompt, systemPrompt, recentMessages, scene)
}

// MergePromptPrivate merges using the private instruction (no clothing cues).
func (m *PromptMerger) MergePromptPrivate(systemPrompt string, recentMessages []Message, scene string) (string, error) {
	instruction := m.config.PrivateMergeInstr
	if instruction == "" {
		instruction = m.config.MergeInstruction
	}
	baseline := m.config.PrivateBaselinePrompt
	if baseline == "" {
		baseline = m.config.BaselinePrompt
	}
	return m.mergePromptWith(instruction, baseline, systemPrompt, recentMessages, scene)
}

func (m *PromptMerger) mergePromptWith(mergeInstruction, baseline, systemPrompt string, recentMessages []Message, scene string) (string, error) {
	if m.config == nil {
		return scene, fmt.Errorf("no config loaded")
	}

	start := time.Now()

	// Build conversation context from recent messages
	var convParts []string
	for _, msg := range recentMessages {
		if msg.Content != "" {
			convParts = append(convParts, msg.Role+": "+msg.Content)
		}
	}
	conversation := strings.Join(convParts, "\n")

	// Build merge instruction by replacing placeholders
	instruction := strings.ReplaceAll(mergeInstruction, "{baseline}", baseline)
	instruction = strings.ReplaceAll(instruction, "{system_prompt}", systemPrompt)
	instruction = strings.ReplaceAll(instruction, "{conversation}", conversation)
	instruction = strings.ReplaceAll(instruction, "{scene}", scene)

	// Call merge model
	messages := []ollamaapi.Message{
		{Role: "user", Content: instruction},
	}

	stream := false
	req := &ollamaapi.ChatRequest{
		Model:    m.config.MergeModel,
		Messages: messages,
		Stream:   &stream,
	}

	var merged strings.Builder
	err := m.client.Chat(context.Background(), req, func(resp ollamaapi.ChatResponse) error {
		merged.WriteString(resp.Message.Content)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("merge LLM call failed: %w", err)
	}

	result := strings.TrimSpace(merged.String())
	latency := time.Since(start).Milliseconds()

	slog.Info("merged image prompt",
		"prompt", result,
		"system_prompt_len", len(systemPrompt),
		"conversation_len", len(conversation),
		"scene", scene,
		"merge_latency_ms", latency)

	return result, nil
}

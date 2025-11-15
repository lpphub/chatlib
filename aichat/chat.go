package aichat

import (
	"context"
)

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Created int64  `json:"created"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason,omitempty"`
	Error        error  `json:"-"`
}

// ModelProvider 模型提供者接口
type ModelProvider interface {
	// StreamChat 流式对话
	StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)
	// IsAvailable 检查模型是否可用
	IsAvailable() bool
}

// ModelFactory 模型工厂
type ModelFactory interface {
	GetProvider(modelName string) (ModelProvider, error)
	ListAvailableModels() []string
}

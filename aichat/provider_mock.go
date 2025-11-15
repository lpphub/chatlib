package aichat

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type MockProvider struct {
	name string
}

func NewMockProvider(name string) *MockProvider {
	return &MockProvider{name: name}
}

func (p *MockProvider) IsAvailable() bool {
	return true
}

func (p *MockProvider) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	chunks := make(chan StreamChunk, 100)

	// 模拟流式响应
	go func() {
		defer close(chunks)

		// 获取最后一条用户消息
		userMessage := ""
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				userMessage = req.Messages[i].Content
				break
			}
		}

		// 模拟响应内容
		response := fmt.Sprintf("这是来自 %s 模型的回复。你说的是：%s\n\n让我为你详细解答：\n",
			p.name, userMessage)

		// 添加一些模拟的详细内容
		details := []string{
			"首先，这是第一点说明。",
			"其次，这里有一些补充信息。",
			"另外，还需要考虑以下因素。",
			"最后，总结一下要点。",
		}

		// 逐字符流式输出
		words := strings.Split(response, "")
		for _, word := range words {
			select {
			case <-ctx.Done():
				return
			case chunks <- StreamChunk{Content: word}:
				time.Sleep(20 * time.Millisecond) // 模拟网络延迟
			}
		}

		// 输出详细内容
		for _, detail := range details {
			words := strings.Split(detail+"\n", "")
			for _, word := range words {
				select {
				case <-ctx.Done():
					return
				case chunks <- StreamChunk{Content: word}:
					time.Sleep(20 * time.Millisecond)
				}
			}
		}

		// 发送结束信号
		chunks <- StreamChunk{FinishReason: "stop"}
	}()

	return chunks, nil
}

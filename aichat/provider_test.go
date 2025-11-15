package aichat

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestBaseProvider_StreamChat(t *testing.T) {
	t.Run("test llm chat", func(t *testing.T) {
		deepseek := BaseProvider{
			options: ProviderOptions{
				APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
				BaseURL: "https://api.deepseek.com/chat/completions",
			},
			Client: &http.Client{
				Timeout: 120 * time.Minute,
			},
		}

		req := &ChatRequest{
			Model: "deepseek-chat",
			Messages: []ChatMessage{
				{
					Role:    "system",
					Content: "你是一个编辑助手",
				},
				{
					Role:    "user",
					Content: "帮我查一下郑州明天的天气怎么样",
				},
			},
			Stream: true,
		}

		chunks, err := deepseek.StreamChat(context.Background(), req)
		if err != nil {
			t.Error(err)
		}

		for chunk := range chunks {
			// 1. 处理错误
			if chunk.Error != nil {
				fmt.Printf("\n[错误] 流式传输中断: %v\n", chunk.Error)
				break
			}

			// 2. 处理结束标志
			if chunk.FinishReason != "" {
				fmt.Println() // 结束前补一个换行，分隔内容与结束提示
				fmt.Printf("[结束] 流式传输完成 (原因: %s)\n", chunk.FinishReason)
				break
			}
			fmt.Print(chunk.Content)

			// 4. 智能换行判断（根据内容特征）
			if chunk.Content != "" {

				runes := []rune(chunk.Content)
				lastRune := runes[len(runes)-1] // 获取最后一个字符（rune类型）
				// 判断是否是句子结束标点（中英文都支持）
				if lastRune == '。' || lastRune == '？' || lastRune == '！' ||
					lastRune == '.' || lastRune == '?' || lastRune == '!' {
					fmt.Println() // 句子结束后换行
				}
			}
		}
	})
}

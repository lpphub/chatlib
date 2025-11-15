package aichat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ProviderOptions struct {
	BaseURL string // API URL
	APIKey  string // API 密钥
}

type BaseProvider struct {
	options ProviderOptions
	Client  *http.Client
}

func (p BaseProvider) StreamChat(ctx context.Context, request *ChatRequest) (<-chan StreamChunk, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.options.BaseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.options.APIKey)

	// 启动goroutine处理流式响应
	chunks := make(chan StreamChunk, 100)
	go func() {
		defer close(chunks)

		var resp *http.Response
		resp, err = p.Client.Do(req)
		if err != nil {
			chunks <- StreamChunk{Error: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			chunks <- StreamChunk{Error: fmt.Errorf("API error: %s", string(body))}
			return
		}

		scanner := bufio.NewScanner(resp.Body)

		for {
			select {
			case <-ctx.Done():
				// 超时：终止读取并返回错误
				chunks <- StreamChunk{Error: fmt.Errorf("streamer read timeout: %v", ctx.Err())}
				return
			default:
				if !scanner.Scan() {
					if err = scanner.Err(); err != nil {
						chunks <- StreamChunk{Error: err}
					}
					return
				}

				line := scanner.Text()
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if data == "[DONE]" {
						return
					}

					// todo 可扩展自定义解析
					var response ChatResponse
					if err = json.Unmarshal([]byte(data), &response); err != nil {
						continue
					}

					if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
						chunks <- StreamChunk{
							Content:      response.Choices[0].Delta.Content,
							FinishReason: response.Choices[0].FinishReason,
						}
					}
				}
			}
		}
	}()
	return chunks, nil
}

func (p BaseProvider) IsAvailable() bool {
	return false
}

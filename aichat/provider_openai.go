package aichat

import (
	"net/http"
	"time"
)

type OpenAIProvider struct {
	BaseProvider
}

func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		BaseProvider{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Client: &http.Client{
				Timeout: 5 * time.Minute,
			},
		},
	}
}

func (p *OpenAIProvider) IsAvailable() bool {
	return p.APIKey != ""
}

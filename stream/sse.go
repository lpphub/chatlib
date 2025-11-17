package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type DataProcessor func(data []byte) ([]byte, error)

type SSEStream struct {
	Processor DataProcessor
}

func (s *SSEStream) Stream(ctx context.Context, w http.ResponseWriter, dataChan <-chan []byte) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, open := <-dataChan:
			if !open {
				return nil
			}

			if s.Processor != nil {
				var err error
				data, err = s.Processor(data)
				if err != nil {
					return err
				}
			}

			_, err := w.Write(data)
			if err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

func SSEDataProcessor(data []byte) ([]byte, error) {
	// Convert data to SSE format: data: <content>\n\n
	sseData := fmt.Sprintf("data: %s\n\n", string(data))
	return []byte(sseData), nil
}

type SSEMessage struct {
	ID    string
	Event string
	Data  string
	Retry *int
}

func SSEMessageProcessor(data []byte) ([]byte, error) {
	var msg SSEMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	var sseData strings.Builder

	// Add ID field if provided
	if msg.ID != "" {
		sseData.WriteString(fmt.Sprintf("id: %s\n", msg.ID))
	}

	// Add Event field if provided
	if msg.Event != "" {
		sseData.WriteString(fmt.Sprintf("event: %s\n", msg.Event))
	}

	// Add Data field (always required)
	dataLines := strings.Split(msg.Data, "\n")
	for _, line := range dataLines {
		sseData.WriteString(fmt.Sprintf("data: %s\n", line))
	}

	// Add Retry field if provided
	if msg.Retry != nil {
		sseData.WriteString(fmt.Sprintf("retry: %d\n", *msg.Retry))
	}

	// End message with double newline
	sseData.WriteString("\n")

	return []byte(sseData.String()), nil
}

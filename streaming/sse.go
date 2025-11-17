package streaming

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// SSEConverter Server-Sent Events 转换器
type SSEConverter struct {
	options *StreamOptions
}

func (s *SSEConverter) Convert(ctx context.Context, w http.ResponseWriter, dataChan <-chan StreamData) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return fmt.Errorf("streaming not supported")
	}

	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-streamer")
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

			if data.Err != nil {
				if s.options.OnError != nil {
					s.options.OnError(data.Err)
				}
				return data.Err
			}

			chunk := data.Payload
			if s.options.Processor != nil {
				bytes, err := s.options.Processor(chunk)
				if err != nil {
					return err
				}
				chunk = bytes
			}

			// 写入SSE格式
			_, err := w.Write(chunk)
			if err != nil {
				log.Printf("Error writing chunk to response: %v", err)
				return err
			}
			flusher.Flush()
		}
	}
}

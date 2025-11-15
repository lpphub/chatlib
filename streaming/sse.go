package streaming

import (
	"context"
	"fmt"
	"io"
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

			// 写入SSE格式
			if err := s.write(w, data.Payload); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

// writeSSE 写入SSE格式数据
func (s *SSEConverter) write(w io.Writer, data []byte) error {
	if s.options.Processor != nil {
		bytes, err := s.options.Processor(data)
		if err != nil {
			return err
		}
		data = bytes
	}

	_, err := fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

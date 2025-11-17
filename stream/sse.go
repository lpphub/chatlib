package stream

import (
	"context"
	"fmt"
	"net/http"
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

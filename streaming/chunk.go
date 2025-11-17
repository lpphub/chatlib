package streaming

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type ChunkConverter struct {
	options *StreamOptions
}

func (c *ChunkConverter) Convert(ctx context.Context, w http.ResponseWriter, dataChan <-chan StreamData) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return fmt.Errorf("streaming not supported")
	}

	// 设置必要的 HTTP 头
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case data, open := <-dataChan:
			if !open {
				return nil
			}

			if data.Err != nil {
				if c.options.OnError != nil {
					c.options.OnError(data.Err)
				}
				return data.Err
			}

			// 处理数据
			chunk := data.Payload
			if c.options.Processor != nil {
				processed, err := c.options.Processor(chunk)
				if err != nil {
					return err
				}
				chunk = processed
			}

			_, err := w.Write(chunk)
			if err != nil {
				log.Printf("Error writing chunk to response: %v", err)
				return err
			}

			flusher.Flush()
		}
	}
}

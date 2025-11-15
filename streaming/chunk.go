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

func (c *ChunkConverter) ChunkedStream(ctx context.Context, w http.ResponseWriter, contentType string, dataChan <-chan []byte) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return fmt.Errorf("streaming not supported")
	}

	// 2. 设置必要的 HTTP 头
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-ctx.Done():
			log.Println("Client disconnected.")
			return ctx.Err()

		case chunk, open := <-dataChan:
			if !open {
				log.Println("Chunked stream finished.")
				return nil
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

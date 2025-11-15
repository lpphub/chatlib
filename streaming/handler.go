package streaming

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

type StreamHandler struct {
	timeout   time.Duration
	producer  StreamProducer
	converter StreamConverter
}

// ServeHTTP 实现http.Handler接口
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	dataChan, err := h.producer.Produce(ctx, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 转换并输出
	if err = h.converter.Convert(ctx, w, dataChan); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			log.Fatalf("streaming convert err: %v", err)
		}
	}
}

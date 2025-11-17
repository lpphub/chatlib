package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSEStream_Stream(t *testing.T) {
	tests := []struct {
		name           string
		sse            *SSEStream
		ctx            context.Context
		dataChan       chan []byte
		expectedError  string
		expectedResult string
		checkHeaders   bool
	}{
		{
			name: "SuccessWithoutProcessor",
			sse:  &SSEStream{},
			ctx:  context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 1)
				ch <- []byte("test message")
				close(ch)
				return ch
			}(),
			expectedResult: "test message",
			checkHeaders:   true,
		},
		{
			name: "SuccessWithProcessor",
			sse: &SSEStream{
				Processor: func(data []byte) ([]byte, error) {
					return append([]byte("processed: "), data...), nil
				},
			},
			ctx: context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 1)
				ch <- []byte("test message")
				close(ch)
				return ch
			}(),
			expectedResult: "processed: test message",
		},
		{
			name: "ProcessorError",
			sse: &SSEStream{
				Processor: func(data []byte) ([]byte, error) {
					return nil, errors.New("processor error")
				},
			},
			ctx: context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 1)
				ch <- []byte("test message")
				close(ch)
				return ch
			}(),
			expectedError: "processor error",
		},
		{
			name: "ContextCancellation",
			sse:  &SSEStream{},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			dataChan:      make(chan []byte),
			expectedError: "context canceled",
		},
		{
			name: "ContextTimeout",
			sse:  &SSEStream{},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()
				return ctx
			}(),
			dataChan:      make(chan []byte),
			expectedError: "context deadline exceeded",
		},
		{
			name: "MultipleMessages",
			sse:  &SSEStream{},
			ctx:  context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 3)
				ch <- []byte("message 1")
				ch <- []byte("message 2")
				ch <- []byte("message 3")
				close(ch)
				return ch
			}(),
			expectedResult: "message 1message 2message 3",
		},
		{
			name: "WithoutFlusher",
			sse:  &SSEStream{},
			ctx:  context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 1)
				ch <- []byte("test message")
				close(ch)
				return ch
			}(),
			expectedError: "streaming not supported",
		},
		{
			name: "WriteError",
			sse:  &SSEStream{},
			ctx:  context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 1)
				ch <- []byte("test message")
				close(ch)
				return ch
			}(),
			expectedError: "write error",
		},
		{
			name:      "SSEDataProcessor_SimpleMessages",
			sse:       &SSEStream{Processor: SSEDataProcessor},
			ctx:       context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 2)
				ch <- []byte("Hello World")
				ch <- []byte("Second message")
				close(ch)
				return ch
			}(),
			expectedResult: "data: Hello World\n\ndata: Second message\n\n",
			checkHeaders:   true,
		},
		{
			name:      "SSEDataProcessor_SpecialCharacters",
			sse:       &SSEStream{Processor: SSEDataProcessor},
			ctx:       context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 2)
				ch <- []byte("Test with 中文 & symbols!@#")
				ch <- []byte("Line 1\nLine 2")
				close(ch)
				return ch
			}(),
			expectedResult: "data: Test with 中文 & symbols!@#\n\ndata: Line 1\nLine 2\n\n",
		},
		{
			name: "SSEMessageProcessor_CompleteMessages",
			sse:  &SSEStream{Processor: SSEMessageProcessor},
			ctx:  context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 2)
				msg1 := SSEMessage{ID: "123", Event: "greeting", Data: "Hello World"}
				msg2 := SSEMessage{ID: "456", Event: "update", Data: "Updated data"}
				data1, _ := json.Marshal(msg1)
				data2, _ := json.Marshal(msg2)
				ch <- data1
				ch <- data2
				close(ch)
				return ch
			}(),
			expectedResult: "id: 123\nevent: greeting\ndata: Hello World\n\nid: 456\nevent: update\ndata: Updated data\n\n",
		},
		{
			name: "SSEMessageProcessor_WithRetry",
			sse:  &SSEStream{Processor: SSEMessageProcessor},
			ctx:  context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 1)
				retry := 5000
				msg := SSEMessage{ID: "retry-test", Event: "error", Data: "Connection lost", Retry: &retry}
				data, _ := json.Marshal(msg)
				ch <- data
				close(ch)
				return ch
			}(),
			expectedResult: "id: retry-test\nevent: error\ndata: Connection lost\nretry: 5000\n\n",
		},
		{
			name: "SSEMessageProcessor_MultilineData",
			sse:  &SSEStream{Processor: SSEMessageProcessor},
			ctx:  context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 1)
				msg := SSEMessage{ID: "multiline", Event: "chat", Data: "Line 1\nLine 2\nLine 3"}
				data, _ := json.Marshal(msg)
				ch <- data
				close(ch)
				return ch
			}(),
			expectedResult: "id: multiline\nevent: chat\ndata: Line 1\ndata: Line 2\ndata: Line 3\n\n",
		},
				{
			name: "SSEMessageProcessor_ErrorHandling",
			sse:  &SSEStream{Processor: SSEMessageProcessor},
			ctx:  context.Background(),
			dataChan: func() chan []byte {
				ch := make(chan []byte, 1)
				ch <- []byte("invalid json")
				close(ch)
				return ch
			}(),
			expectedError: "invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w http.ResponseWriter
			if tt.name == "WithoutFlusher" {
				w = &nonFlusherResponseWriter{}
			} else if tt.name == "WriteError" {
				w = &errorResponseWriter{}
			} else {
				w = httptest.NewRecorder()
			}

			err := tt.sse.Stream(tt.ctx, w, tt.dataChan)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error %q, got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("Expected error %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}

			if tt.expectedResult != "" {
				recorder, ok := w.(*httptest.ResponseRecorder)
				if ok {
					result := recorder.Body.String()
					if result != tt.expectedResult {
						t.Errorf("Expected result %q, got %q", tt.expectedResult, result)
					}
				}
			}

			if tt.checkHeaders {
				recorder, ok := w.(*httptest.ResponseRecorder)
				if ok {
					if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
						t.Errorf("Expected Content-Type text/event-stream, got %q", contentType)
					}
					if cacheControl := recorder.Header().Get("Cache-Control"); cacheControl != "no-cache" {
						t.Errorf("Expected Cache-Control no-cache, got %q", cacheControl)
					}
					if connection := recorder.Header().Get("Connection"); connection != "keep-alive" {
						t.Errorf("Expected Connection keep-alive, got %q", connection)
					}
					if accelBuffering := recorder.Header().Get("X-Accel-Buffering"); accelBuffering != "no" {
						t.Errorf("Expected X-Accel-Buffering no, got %q", accelBuffering)
					}
				}
			}
		})
	}
}

// Helper types for testing

type nonFlusherResponseWriter struct {
	buf bytes.Buffer
}

func (w *nonFlusherResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (w *nonFlusherResponseWriter) Write(data []byte) (int, error) {
	return w.buf.Write(data)
}

func (w *nonFlusherResponseWriter) WriteHeader(statusCode int) {}

type errorResponseWriter struct {
	buf bytes.Buffer
}

func (w *errorResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (w *errorResponseWriter) Write(data []byte) (int, error) {
	return 0, errors.New("write error")
}

func (w *errorResponseWriter) WriteHeader(statusCode int) {}

func (w *errorResponseWriter) Flush() {}

// Benchmark test
func BenchmarkSSEStream_Stream(b *testing.B) {
	sse := &SSEStream{}
	ctx := context.Background()

	processor := func(data []byte) ([]byte, error) {
		return data, nil
	}
	sse.Processor = processor

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataChan := make(chan []byte, 100)
		for j := 0; j < 100; j++ {
			dataChan <- []byte("benchmark message")
		}
		close(dataChan)

		w := httptest.NewRecorder()

		err := sse.Stream(ctx, w, dataChan)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

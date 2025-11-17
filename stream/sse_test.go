package stream

import (
	"bytes"
	"context"
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

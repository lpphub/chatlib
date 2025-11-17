package streaming

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockProducer struct {
	data []StreamData
	err  error
}

func (m *mockProducer) Produce(ctx context.Context, req *http.Request) (<-chan StreamData, error) {
	if m.err != nil {
		return nil, m.err
	}

	dataChan := make(chan StreamData, len(m.data))
	for _, d := range m.data {
		dataChan <- d
	}
	close(dataChan)
	return dataChan, nil
}

type mockConverter struct {
	err error
}

func (m *mockConverter) Convert(ctx context.Context, w http.ResponseWriter, dataChan <-chan StreamData) error {
	if m.err != nil {
		return m.err
	}

	for data := range dataChan {
		if data.Err != nil {
			return data.Err
		}
		w.Write(data.Payload)
	}
	return nil
}

func TestStreamHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name       string
		timeout    time.Duration
		producer   StreamProducer
		converter  StreamConverter
		wantStatus int
		wantBody   string
	}{
		{
			name:    "successful streaming",
			timeout: 5 * time.Second,
			producer: &mockProducer{
				data: []StreamData{
					{Payload: []byte("hello ")},
					{Payload: []byte("world")},
				},
			},
			converter:  &mockConverter{},
			wantStatus: http.StatusOK,
			wantBody:   "hello world",
		},
		{
			name:    "producer error",
			timeout: 5 * time.Second,
			producer: &mockProducer{
				err: errors.New("producer error"),
			},
			converter:  &mockConverter{},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "producer error\n",
		},
		{
			name:    "converter error",
			timeout: 5 * time.Second,
			producer: &mockProducer{
				data: []StreamData{
					{Payload: []byte("test")},
				},
			},
			converter: &mockConverter{
				err: errors.New("converter error"),
			},
			wantStatus: http.StatusOK, // converter errors are logged, not returned as HTTP errors
			wantBody:   "",
		},
		{
			name:    "context cancelled",
			timeout: time.Nanosecond, // Very short timeout to trigger cancellation
			producer: &mockProducer{
				data: []StreamData{
					{Payload: []byte("test")},
				},
			},
			converter:  &mockConverter{},
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &StreamHandler{
				timeout:   tt.timeout,
				producer:  tt.producer,
				converter: tt.converter,
			}

			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if status := w.Code; status != tt.wantStatus {
				t.Errorf("ServeHTTP() status = %v, want %v", status, tt.wantStatus)
			}

			if body := w.Body.String(); body != tt.wantBody {
				t.Errorf("ServeHTTP() body = %v, want %v", body, tt.wantBody)
			}
		})
	}
}

func TestStreamHandler_ContextCancellation(t *testing.T) {
	h := &StreamHandler{
		timeout: 5 * time.Second,
		producer: &mockProducer{
			data: []StreamData{
				{Payload: []byte("test")},
			},
		},
		converter: &mockConverter{
			err: context.Canceled,
		},
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	// Should not panic or log fatal for context cancellation
	if status := w.Code; status != http.StatusOK {
		t.Errorf("ServeHTTP() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestStreamHandler_DeadlineExceeded(t *testing.T) {
	h := &StreamHandler{
		timeout: 5 * time.Second,
		producer: &mockProducer{
			data: []StreamData{
				{Payload: []byte("test")},
			},
		},
		converter: &mockConverter{
			err: context.DeadlineExceeded,
		},
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	// Should not panic or log fatal for deadline exceeded
	if status := w.Code; status != http.StatusOK {
		t.Errorf("ServeHTTP() status = %v, want %v", w.Code, http.StatusOK)
	}
}

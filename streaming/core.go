package streaming

import (
	"context"
	"net/http"
)

// StreamData 流数据包
type StreamData struct {
	Payload []byte // 数据内容
	Err     error  // 错误信息
}

// StreamConverter 流转换器
type StreamConverter interface {
	Convert(ctx context.Context, w http.ResponseWriter, dataChan <-chan StreamData) error
}

// StreamProducer 流生产者
type StreamProducer interface {
	Produce(ctx context.Context, req *http.Request) (<-chan StreamData, error)
}
type StreamProducerFunc func(ctx context.Context, req *http.Request) (<-chan StreamData, error)

func (f StreamProducerFunc) Produce(ctx context.Context, req *http.Request) (<-chan StreamData, error) {
	return f(ctx, req)
}

// StreamDataProcessor 数据处理器
type StreamDataProcessor func(data []byte) ([]byte, error)

// StreamOptions 流选项
type StreamOptions struct {
	// 数据处理器
	Processor StreamDataProcessor
	// 错误处理
	OnError func(error)
	// 缓冲区大小
	BufferSize int
	// 是否自动关闭
	AutoClose bool
}

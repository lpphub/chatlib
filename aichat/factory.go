package aichat

import (
	"fmt"
	"sync"
)

type DefaultModelFactory struct {
	providers map[string]ModelProvider
	mu        sync.RWMutex
}

func NewDefaultModelFactory() *DefaultModelFactory {
	return &DefaultModelFactory{
		providers: make(map[string]ModelProvider),
	}
}

func (f *DefaultModelFactory) RegisterProvider(name string, provider ModelProvider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[name] = provider
}

func (f *DefaultModelFactory) GetProvider(modelName string) (ModelProvider, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	provider, ok := f.providers[modelName]
	if !ok {
		return nil, fmt.Errorf("model %s not found", modelName)
	}

	if !provider.IsAvailable() {
		return nil, fmt.Errorf("model %s is not available", modelName)
	}

	return provider, nil
}

func (f *DefaultModelFactory) ListAvailableModels() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var models []string
	for modelName, provider := range f.providers {
		if provider.IsAvailable() {
			models = append(models, modelName)
		}
	}
	return models
}

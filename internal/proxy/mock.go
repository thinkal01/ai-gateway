package proxy

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MockProvider 模拟提供商（开发/测试环境）
type MockProvider struct {
	baseURL string
	client  *http.Client
}

// NewMockProvider 创建 MockProvider 实例
func NewMockProvider(baseURL string) *MockProvider {
	return &MockProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

func (p *MockProvider) ChatCompletion(body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/mock/chat/completions", body)
	if err != nil {
		return nil, fmt.Errorf("create chat completion request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat completion request failed: %w", err)
	}
	return resp, nil
}

func (p *MockProvider) ListModels() (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, p.baseURL+"/mock/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create list models request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list models request failed: %w", err)
	}
	return resp, nil
}

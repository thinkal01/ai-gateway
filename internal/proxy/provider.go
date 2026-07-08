package proxy

import (
	"io"
	"net/http"
)

// Provider AI 模型提供商接口
type Provider interface {
	// ChatCompletion 转发聊天补全请求
	ChatCompletion(body io.Reader) (*http.Response, error)

	// ListModels 获取模型列表
	ListModels() (*http.Response, error)
}

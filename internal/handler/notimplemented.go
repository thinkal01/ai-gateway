package handler

import "net/http"

// NotImplementedHandler 返回 501 状态码
type NotImplementedHandler struct{}

// NewNotImplementedHandler 创建 NotImplementedHandler
func NewNotImplementedHandler() *NotImplementedHandler {
	return &NotImplementedHandler{}
}

// ServeHTTP 实现 http.Handler 接口
func (h *NotImplementedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "endpoint not implemented yet")
}

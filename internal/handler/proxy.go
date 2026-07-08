package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/vrviu/ai-gateway/internal/auth"
	"github.com/vrviu/ai-gateway/internal/middleware"
	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/proxy"
	"github.com/vrviu/ai-gateway/internal/usage"
)

// ProxyHandler 代理请求 handler
type ProxyHandler struct {
	provider proxy.Provider
	recorder *usage.Recorder
}

// NewProxyHandler 创建 ProxyHandler
func NewProxyHandler(provider proxy.Provider, recorder *usage.Recorder) *ProxyHandler {
	return &ProxyHandler{
		provider: provider,
		recorder: recorder,
	}
}

// ChatCompletionRequest 代理请求体
type ChatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
}

// ChatCompletion 处理 /v1/chat/completions
func (h *ProxyHandler) ChatCompletion(w http.ResponseWriter, r *http.Request) {
	// 从 context 获取认证信息
	tenantID := middleware.GetTenantID(r.Context())
	keyID := middleware.GetKeyID(r.Context())
	scopes := middleware.GetScopes(r.Context())

	// 解析请求体
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages is required")
		return
	}

	// Scope 校验
	if !auth.MatchScope(scopes, req.Model) {
		writeError(w, http.StatusForbidden, "insufficient permissions for model "+req.Model)
		return
	}

	// 序列化请求体以转发
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to serialize request")
		return
	}

	// 转发到下游
	resp, err := h.provider.ChatCompletion(strings.NewReader(string(bodyBytes)))
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream service unavailable: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// 读取下游响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to read upstream response")
		return
	}

	// 下游返回非 200 时，直接透传
	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	// 估算 token 数（简化：按字符数估算）
	promptTokens := estimateTokens(req.Messages)
	completionTokens := estimateTokens([]map[string]string{{"content": string(respBody)}})
	totalTokens := promptTokens + completionTokens

	// 异步记录用量
	h.recorder.Record(&model.UsageRecord{
		TenantID:         tenantID,
		KeyID:            keyID,
		Model:            req.Model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		RequestID:        chimw.GetReqID(r.Context()),
	})

	// 返回下游响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBody)
}

// ListModels 处理 /v1/models
func (h *ProxyHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	resp, err := h.provider.ListModels()
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream service unavailable: "+err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to read upstream response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// estimateTokens 估算 token 数（简化版：每 4 字符 ≈ 1 token）
func estimateTokens(messages []map[string]string) int {
	total := 0
	for _, msg := range messages {
		for _, v := range msg {
			total += len(v) / 4
		}
	}
	if total < 1 {
		total = 1
	}
	return total
}

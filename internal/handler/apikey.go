package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vrviu/ai-gateway/internal/auth"
	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/store"
)

// ApiKeyHandler API Key 管理 handler
type ApiKeyHandler struct {
	store store.ApiKeyStore
}

// NewApiKeyHandler 创建 ApiKeyHandler
func NewApiKeyHandler(store store.ApiKeyStore) *ApiKeyHandler {
	return &ApiKeyHandler{store: store}
}

// Create 创建 API Key
func (h *ApiKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateApiKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 手动校验必填字段
	if req.TenantID == "" || req.Name == "" || req.Scopes == "" {
		writeError(w, http.StatusBadRequest, "tenant_id, name and scopes are required")
		return
	}

	// 生成 Key
	fullKey, prefix, keyHash, err := auth.GenerateKey(8, 16)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate key")
		return
	}

	apiKey := &model.ApiKey{
		TenantID:  req.TenantID,
		KeyHash:   keyHash,
		KeyPrefix: prefix,
		Name:      req.Name,
		Scopes:    req.Scopes,
		IsActive:  true,
		ExpiresAt: req.ExpiresAt,
	}

	if err := h.store.Create(apiKey); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 响应中返回完整 Key（仅创建时可见）
	resp := toApiKeyResponse(apiKey)
	resp.FullKey = fullKey
	writeJSON(w, http.StatusCreated, resp)
}

// GetByID 获取 API Key
func (h *ApiKeyHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	apiKey, err := h.store.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if apiKey == nil {
		writeError(w, http.StatusNotFound, "api key not found")
		return
	}

	writeJSON(w, http.StatusOK, toApiKeyResponse(apiKey))
}

// List 列出所有 API Key
func (h *ApiKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	apiKeys, err := h.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]*model.ApiKeyResponse, len(apiKeys))
	for i, k := range apiKeys {
		responses[i] = toApiKeyResponse(k)
	}

	writeJSON(w, http.StatusOK, responses)
}

// Update 更新 API Key
func (h *ApiKeyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateApiKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.store.Update(id, &req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiKey, _ := h.store.GetByID(id)
	writeJSON(w, http.StatusOK, toApiKeyResponse(apiKey))
}

// Delete 删除 API Key
func (h *ApiKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toApiKeyResponse(k *model.ApiKey) *model.ApiKeyResponse {
	if k == nil {
		return nil
	}
	var expiresAt *time.Time
	if !k.ExpiresAt.IsZero() {
		expiresAt = &k.ExpiresAt
	}
	return &model.ApiKeyResponse{
		ID:        k.ID,
		TenantID:  k.TenantID,
		KeyPrefix: k.KeyPrefix,
		Name:      k.Name,
		Scopes:    k.Scopes,
		IsActive:  k.IsActive,
		ExpiresAt: expiresAt,
		CreatedAt: k.CreatedAt,
		UpdatedAt: k.UpdatedAt,
	}
}

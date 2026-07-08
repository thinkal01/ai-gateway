package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/store"
)

// TenantHandler 租户管理 handler
type TenantHandler struct {
	store store.TenantStore
}

// NewTenantHandler 创建 TenantHandler
func NewTenantHandler(store store.TenantStore) *TenantHandler {
	return &TenantHandler{store: store}
}

// Create 创建租户
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenant := &model.Tenant{
		Name:     req.Name,
		IsActive: true,
	}

	if err := h.store.Create(tenant); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toTenantResponse(tenant))
}

// GetByID 获取租户
func (h *TenantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenant, err := h.store.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tenant == nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	writeJSON(w, http.StatusOK, toTenantResponse(tenant))
}

// List 列出所有租户
func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]*model.TenantResponse, len(tenants))
	for i, t := range tenants {
		responses[i] = toTenantResponse(t)
	}

	writeJSON(w, http.StatusOK, responses)
}

// Update 更新租户
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.store.Update(id, &req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tenant, _ := h.store.GetByID(id)
	writeJSON(w, http.StatusOK, toTenantResponse(tenant))
}

// Delete 删除租户
func (h *TenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toTenantResponse(t *model.Tenant) *model.TenantResponse {
	if t == nil {
		return nil
	}
	return &model.TenantResponse{
		ID:        t.ID,
		Name:      t.Name,
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

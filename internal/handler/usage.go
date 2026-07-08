package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/store"
)

// UsageHandler 用量查询 handler
type UsageHandler struct {
	store store.UsageStore
}

// NewUsageHandler 创建 UsageHandler
func NewUsageHandler(store store.UsageStore) *UsageHandler {
	return &UsageHandler{store: store}
}

// Query 查询用量记录
func (h *UsageHandler) Query(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := &model.UsageQuery{
		TenantID: q.Get("tenant_id"),
		KeyID:    q.Get("key_id"),
		Model:    q.Get("model"),
	}

	if startStr := q.Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			query.StartAt = t
		}
	}
	if endStr := q.Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			query.EndAt = t
		}
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			query.Limit = l
		}
	}
	if offsetStr := q.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			query.Offset = o
		}
	}

	if query.Limit <= 0 {
		query.Limit = 20
	}

	records, err := h.store.Query(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []*model.UsageRecord{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  records,
		"total": len(records),
	})
}

// Summary 查询用量汇总
func (h *UsageHandler) Summary(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	startAt := time.Time{}
	endAt := time.Time{}

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startAt = t
		}
	}
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endAt = t
		}
	}

	summary, err := h.store.GetSummary(tenantID, startAt, endAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

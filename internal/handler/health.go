package handler

import (
	"encoding/json"
	"net/http"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// HealthCheck 健康检查 handler
func HealthCheck(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:  "ok",
			Version: version,
		})
	}
}

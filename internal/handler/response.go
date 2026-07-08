package handler

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	resp := ErrorResponse{}
	resp.Error.Message = message
	resp.Error.Code = status
	switch {
	case status >= 500:
		resp.Error.Type = "server_error"
	case status >= 400:
		resp.Error.Type = "request_error"
	default:
		resp.Error.Type = "unknown"
	}
	writeJSON(w, status, resp)
}

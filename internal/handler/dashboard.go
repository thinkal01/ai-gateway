package handler

import (
	_ "embed"
	"net/http"
)

//go:embed dashboard.html
var dashboardHTMLContent string

// Dashboard 返回管理面板页面
func Dashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(dashboardHTMLContent))
	}
}

package router

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/vrviu/ai-gateway/internal/auth"
	"github.com/vrviu/ai-gateway/internal/config"
	"github.com/vrviu/ai-gateway/internal/handler"
	"github.com/vrviu/ai-gateway/internal/middleware"
	"github.com/vrviu/ai-gateway/internal/proxy"
	"github.com/vrviu/ai-gateway/internal/store"
	"github.com/vrviu/ai-gateway/internal/usage"
)

// New 创建路由
func New(cfg *config.Config, db *sql.DB) (*chi.Mux, error) {
	r := chi.NewRouter()

	// 全局中间件
	r.Use(chimw.RequestID)
	r.Use(middleware.Logging)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS("*"))

	// 初始化 store
	tenantStore := store.NewTenantStore(db)
	apiKeyStore := store.NewApiKeyStore(db)
	usageStore := store.NewUsageStore(db)

	// 初始化 auth service
	authService := auth.NewService(apiKeyStore, 8) // 前缀长度 8 位，与 GenerateKey(8, ...) 保持一致

	// 初始化 usage recorder
	usageRecorder := usage.NewRecorder(usageStore, cfg.UsageFlushInterval, cfg.UsageFlushBatchSize)

	// 初始化 provider
	var provider proxy.Provider
	provider = proxy.NewMockProvider(cfg.MockProviderBaseURL)

	// 初始化 handler
	healthHandler := handler.HealthCheck(cfg.Version)
	tenantHandler := handler.NewTenantHandler(tenantStore)
	apiKeyHandler := handler.NewApiKeyHandler(apiKeyStore)
	usageHandler := handler.NewUsageHandler(usageStore)
	proxyHandler := handler.NewProxyHandler(provider, usageRecorder)

	// 健康检查（无需认证）
	r.Get("/health", healthHandler)

	// 管理面板
	r.Get("/dashboard", handler.Dashboard())
	r.Get("/dashboard/", handler.Dashboard())

	// API 路由（需要认证）
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(authService))

		// 代理端点
		r.Post("/v1/chat/completions", proxyHandler.ChatCompletion)
		r.Get("/v1/models", proxyHandler.ListModels)
	})

	// 管理 API（内部使用）
	r.Route("/api", func(r chi.Router) {
		r.Route("/tenants", func(r chi.Router) {
			r.Post("/", tenantHandler.Create)
			r.Get("/", tenantHandler.List)
			r.Get("/{id}", tenantHandler.GetByID)
			r.Put("/{id}", tenantHandler.Update)
			r.Delete("/{id}", tenantHandler.Delete)
		})

		r.Route("/keys", func(r chi.Router) {
			r.Post("/", apiKeyHandler.Create)
			r.Get("/", apiKeyHandler.List)
			r.Get("/{id}", apiKeyHandler.GetByID)
			r.Put("/{id}", apiKeyHandler.Update)
			r.Delete("/{id}", apiKeyHandler.Delete)
		})

		// 用量查询
		r.Route("/usage", func(r chi.Router) {
			r.Get("/", usageHandler.Query)
			r.Get("/summary", usageHandler.Summary)
		})
	})

	return r, nil
}

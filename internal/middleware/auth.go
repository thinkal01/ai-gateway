package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vrviu/ai-gateway/internal/auth"
)

type contextKey string

const (
	// ContextKeyTenantID 上下文中存租户ID的 key
	ContextKeyTenantID contextKey = "tenant_id"
	// ContextKeyKeyID 上下文中存 KeyID 的 key
	ContextKeyKeyID contextKey = "key_id"
	// ContextKeyScopes 上下文中存 Scopes 的 key
	ContextKeyScopes contextKey = "scopes"
)

// Authenticate API Key 认证中间件
func Authenticate(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从 Authorization header 提取 Key
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":{"message":"missing authorization header","type":"auth_error","code":401}}`, http.StatusUnauthorized)
				return
			}

			keyStr := strings.TrimPrefix(authHeader, "Bearer ")
			if keyStr == authHeader {
				http.Error(w, `{"error":{"message":"invalid authorization format","type":"auth_error","code":401}}`, http.StatusUnauthorized)
				return
			}

			// 从请求路径或 body 提取 model 名称
			modelName := extractModelFromRequest(r)

			// 验证 Key
			result, err := authService.ValidateKey(keyStr, modelName)
			if err != nil {
				log.Printf("[Auth] validation failed: %v", err)
				http.Error(w, `{"error":{"message":"authentication failed","type":"auth_error","code":401}}`, http.StatusUnauthorized)
				return
			}

			// 把认证信息存入 context
			ctx := context.WithValue(r.Context(), ContextKeyTenantID, result.TenantID)
			ctx = context.WithValue(ctx, ContextKeyKeyID, result.KeyID)
			ctx = context.WithValue(ctx, ContextKeyScopes, result.Scopes)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractModelFromRequest 从请求中提取模型名称
// 这里简化处理：从 URL 路径中匹配 chat/completions ，模型名由后续 handler 从 body 中解析
func extractModelFromRequest(r *http.Request) string {
	// 对于 /v1/chat/completions 或 /chat/completions 路径，默认允许（模型名在 body 中）
	if strings.Contains(r.URL.Path, "chat/completions") {
		return "*"
	}
	// 模型列表不依赖特定模型
	if strings.Contains(r.URL.Path, "/v1/models") {
		return "*"
	}
	return ""
}

// GetTenantID 从 context 获取租户 ID
func GetTenantID(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyTenantID).(string); ok {
		return v
	}
	return ""
}

// GetKeyID 从 context 获取 Key ID
func GetKeyID(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyKeyID).(string); ok {
		return v
	}
	return ""
}

// GetScopes 从 context 获取 Scopes
func GetScopes(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyScopes).(string); ok {
		return v
	}
	return ""
}

// Logging 请求日志中间件
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[HTTP] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// CORS CORS 中间件
func CORS(allowedOrigins string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

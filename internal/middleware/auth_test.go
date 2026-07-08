package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrviu/ai-gateway/internal/auth"
	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/store"
)

func setupAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}
	return db
}

func createTestKey(t *testing.T, db *sql.DB) string {
	t.Helper()

	// 创建租户
	tenantStore := store.NewTenantStore(db)
	tenant := &model.Tenant{Name: "test-tenant", IsActive: true}
	err := tenantStore.Create(tenant)
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	// 生成有效的 API Key
	fullKey, prefix, keyHash, err := auth.GenerateKey(8, 16)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	apiKeyStore := store.NewApiKeyStore(db)
	key := &model.ApiKey{
		TenantID:  tenant.ID,
		Name:      "test-key",
		KeyPrefix: prefix,
		KeyHash:   keyHash,
		Scopes:    "*",
		IsActive:  true,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	err = apiKeyStore.Create(key)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	return fullKey
}

func createAuthService(t *testing.T, db *sql.DB) *auth.Service {
	t.Helper()
	apiKeyStore := store.NewApiKeyStore(db)
	return auth.NewService(apiKeyStore, 8)
}

func nextHandlerOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 context 中的值
		tenantID := GetTenantID(r.Context())
		keyID := GetKeyID(r.Context())
		scopes := GetScopes(r.Context())

		if tenantID == "" || keyID == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"scopes":"` + scopes + `"}`))
	})
}

func TestAuthenticate_ValidKey(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	fullKey := createTestKey(t, db)
	authService := createAuthService(t, db)
	middleware := Authenticate(authService)

	req := httptest.NewRequest("POST", "/v1/chat/completions",
		strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Authorization", "Bearer "+fullKey)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	middleware(nextHandlerOK()).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestAuthenticate_MissingHeader(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	authService := createAuthService(t, db)
	middleware := Authenticate(authService)

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	w := httptest.NewRecorder()
	middleware(nextHandlerOK()).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "missing authorization header") {
		t.Errorf("expected 'missing authorization header' in response, got: %s", body)
	}
}

func TestAuthenticate_InvalidFormat(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	authService := createAuthService(t, db)
	middleware := Authenticate(authService)

	// 使用 "Token" 而不是 "Bearer"
	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Token invalid-format")
	w := httptest.NewRecorder()
	middleware(nextHandlerOK()).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "invalid authorization format") {
		t.Errorf("expected 'invalid authorization format' in response, got: %s", body)
	}
}

func TestAuthenticate_InvalidKey(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	authService := createAuthService(t, db)
	middleware := Authenticate(authService)

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-invalid-key")
	w := httptest.NewRecorder()
	middleware(nextHandlerOK()).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCORS(t *testing.T) {
	middleware := CORS("*")

	req := httptest.NewRequest("OPTIONS", "/health", nil)
	w := httptest.NewRecorder()
	middleware(nextHandlerOK()).ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}

	// 验证 CORS headers
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}
}

func TestCORS_HeaderOnNormalRequest(t *testing.T) {
	middleware := CORS("https://example.com")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for normal request, got %d", w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'https://example.com', got '%s'", origin)
	}
}

func TestGetTenantID_EmptyContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if id := GetTenantID(req.Context()); id != "" {
		t.Errorf("expected empty string, got '%s'", id)
	}
}

func TestGetKeyID_EmptyContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if id := GetKeyID(req.Context()); id != "" {
		t.Errorf("expected empty string, got '%s'", id)
	}
}

func TestGetScopes_EmptyContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if scopes := GetScopes(req.Context()); scopes != "" {
		t.Errorf("expected empty string, got '%s'", scopes)
	}
}

func TestLogging(t *testing.T) {
	middleware := Logging
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestExtractModelFromRequest_ChatCompletions(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	model := extractModelFromRequest(req)
	if model != "*" {
		t.Errorf("expected '*', got '%s'", model)
	}
}

func TestExtractModelFromRequest_ListModels(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/models", nil)
	model := extractModelFromRequest(req)
	if model != "*" {
		t.Errorf("expected '*', got '%s'", model)
	}
}

func TestExtractModelFromRequest_OtherPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	model := extractModelFromRequest(req)
	if model != "" {
		t.Errorf("expected empty string, got '%s'", model)
	}
}

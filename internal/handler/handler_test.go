package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vrviu/ai-gateway/internal/middleware"
	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/proxy"
	"github.com/vrviu/ai-gateway/internal/store"
	"github.com/vrviu/ai-gateway/internal/usage"
)

func setupHandlerDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}
	return db
}

func withChiURLParam(ctx context.Context, key, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}

func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := HealthCheck("1.0.0")
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", resp.Version)
	}
}

func TestHealthCheck_ContentType(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := HealthCheck("1.0.0")
	handler.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
	}
}

func TestNotImplemented(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	w := httptest.NewRecorder()

	handler := NewNotImplementedHandler()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", w.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error.Message != "endpoint not implemented yet" {
		t.Errorf("unexpected error message: %s", errResp.Error.Message)
	}
}

func TestTenantHandler_Create(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)

	body := `{"name":"test-tenant"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var resp model.TenantResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "test-tenant" {
		t.Errorf("expected name 'test-tenant', got '%s'", resp.Name)
	}
	if resp.ID == "" {
		t.Error("expected ID to be set")
	}
}

func TestTenantHandler_Create_InvalidBody(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)

	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(`invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTenantHandler_List(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)

	// 创建 2 个租户
	for _, name := range []string{"tenant-a", "tenant-b"} {
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Create(w, req)
	}

	req := httptest.NewRequest("GET", "/api/tenants", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []model.TenantResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 tenants, got %d", len(resp))
	}
}

func TestTenantHandler_Delete(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)

	// 先创建
	body := `{"name":"to-delete"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var created model.TenantResponse
	json.NewDecoder(w.Body).Decode(&created)

	// 删除 - 需要 chi URL 参数上下文
	req2 := httptest.NewRequest("DELETE", "/api/tenants/"+created.ID, nil)
	req2 = req2.WithContext(withChiURLParam(req2.Context(), "id", created.ID))
	w2 := httptest.NewRecorder()
	handler.Delete(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w2.Code)
	}

	// 验证已删除
	got, _ := tenantStore.GetByID(created.ID)
	if got != nil {
		t.Error("expected tenant to be deleted")
	}
}

func TestApiKeyHandler_Create(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)

	// 先创建租户
	tenantBody := `{"name":"test-tenant"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(tenantBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tenantHandler := NewTenantHandler(tenantStore)
	tenantHandler.Create(w, req)

	var tenant model.TenantResponse
	json.NewDecoder(w.Body).Decode(&tenant)

	// 创建 Key
	keyBody := fmt.Sprintf(`{"tenant_id":"%s","name":"test-key","scopes":"*","expires_at":"2027-01-01T00:00:00Z"}`, tenant.ID)
	req2 := httptest.NewRequest("POST", "/api/keys", strings.NewReader(keyBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.Create(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w2.Code)
	}

	var resp model.ApiKeyResponse
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Name != "test-key" {
		t.Errorf("expected name 'test-key', got '%s'", resp.Name)
	}
	if resp.FullKey == "" {
		t.Error("expected full_key to be returned on creation")
	}
}

func TestApiKeyHandler_Create_InvalidBody(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)

	req := httptest.NewRequest("POST", "/api/keys", strings.NewReader(`invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApiKeyHandler_List(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	apiKeyStore := store.NewApiKeyStore(db)
	tenantHandler := NewTenantHandler(tenantStore)
	apiKeyHandler := NewApiKeyHandler(apiKeyStore)

	// 创建租户
	tenantBody := `{"name":"test-tenant"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(tenantBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tenantHandler.Create(w, req)

	var tenant model.TenantResponse
	json.NewDecoder(w.Body).Decode(&tenant)

	// 创建 2 个 Key
	keyNames := []string{"key-a", "key-b"}
	for _, name := range keyNames {
		keyBody := fmt.Sprintf(`{"tenant_id":"%s","name":"%s","scopes":"*","expires_at":"2027-01-01T00:00:00Z"}`, tenant.ID, name)
		req2 := httptest.NewRequest("POST", "/api/keys", strings.NewReader(keyBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		apiKeyHandler.Create(w2, req2)

		if w2.Code != http.StatusCreated {
			t.Fatalf("failed to create key %s: %d, body: %s", name, w2.Code, w2.Body.String())
		}
	}

	req3 := httptest.NewRequest("GET", "/api/keys", nil)
	w3 := httptest.NewRecorder()
	apiKeyHandler.List(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w3.Code)
	}

	var resp []model.ApiKeyResponse
	json.NewDecoder(w3.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 keys, got %d", len(resp))
	}
}

func TestApiKeyHandler_Delete(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	apiKeyStore := store.NewApiKeyStore(db)
	tenantHandler := NewTenantHandler(tenantStore)
	apiKeyHandler := NewApiKeyHandler(apiKeyStore)

	// 创建租户
	tenantBody := `{"name":"test-tenant"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(tenantBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tenantHandler.Create(w, req)

	var tenant model.TenantResponse
	json.NewDecoder(w.Body).Decode(&tenant)

	// 创建 Key
	keyBody := fmt.Sprintf(`{"tenant_id":"%s","name":"to-delete","scopes":"*","expires_at":"2027-01-01T00:00:00Z"}`, tenant.ID)
	req2 := httptest.NewRequest("POST", "/api/keys", strings.NewReader(keyBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	apiKeyHandler.Create(w2, req2)

	var created model.ApiKeyResponse
	json.NewDecoder(w2.Body).Decode(&created)

	// 删除 - 需要 chi URL 参数上下文
	req3 := httptest.NewRequest("DELETE", "/api/keys/"+created.ID, nil)
	req3 = req3.WithContext(withChiURLParam(req3.Context(), "id", created.ID))
	w3 := httptest.NewRecorder()
	apiKeyHandler.Delete(w3, req3)

	if w3.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w3.Code)
	}

	// 验证已删除
	got, _ := apiKeyStore.GetByID(created.ID)
	if got != nil {
		t.Error("expected key to be deleted")
	}
}

// --- Mock Provider for ProxyHandler tests ---

type mockProvider struct {
	chatCompletionFn func(body io.Reader) (*http.Response, error)
	listModelsFn     func() (*http.Response, error)
}

func (m *mockProvider) ChatCompletion(body io.Reader) (*http.Response, error) {
	if m.chatCompletionFn != nil {
		return m.chatCompletionFn(body)
	}
	return nil, fmt.Errorf("chatCompletionFn not set")
}

func (m *mockProvider) ListModels() (*http.Response, error) {
	if m.listModelsFn != nil {
		return m.listModelsFn()
	}
	return nil, fmt.Errorf("listModelsFn not set")
}

// --- TenantHandler tests ---

func TestTenantHandler_GetByID(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)

	// 先创建
	body := `{"name":"test-tenant"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var created model.TenantResponse
	json.NewDecoder(w.Body).Decode(&created)

	// 查询
	req2 := httptest.NewRequest("GET", "/api/tenants/"+created.ID, nil)
	req2 = req2.WithContext(withChiURLParam(req2.Context(), "id", created.ID))
	w2 := httptest.NewRecorder()
	handler.GetByID(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}

	var resp model.TenantResponse
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Name != "test-tenant" {
		t.Errorf("expected name 'test-tenant', got '%s'", resp.Name)
	}
	if resp.ID != created.ID {
		t.Errorf("expected ID '%s', got '%s'", created.ID, resp.ID)
	}
}

func TestTenantHandler_GetByID_NotFound(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)

	req := httptest.NewRequest("GET", "/api/tenants/nonexistent", nil)
	req = req.WithContext(withChiURLParam(req.Context(), "id", "nonexistent"))
	w := httptest.NewRecorder()
	handler.GetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestTenantHandler_Update(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)

	// 先创建
	body := `{"name":"original"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var created model.TenantResponse
	json.NewDecoder(w.Body).Decode(&created)

	// 更新
	updateBody := `{"name":"updated"}`
	req2 := httptest.NewRequest("PUT", "/api/tenants/"+created.ID, strings.NewReader(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	req2 = req2.WithContext(withChiURLParam(req2.Context(), "id", created.ID))
	w2 := httptest.NewRecorder()
	handler.Update(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}

	var resp model.TenantResponse
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Name != "updated" {
		t.Errorf("expected name 'updated', got '%s'", resp.Name)
	}
}

func TestTenantHandler_Update_InvalidBody(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)

	req := httptest.NewRequest("PUT", "/api/tenants/some-id", strings.NewReader(`invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withChiURLParam(req.Context(), "id", "some-id"))
	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- ApiKeyHandler tests ---

func TestApiKeyHandler_GetByID(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	apiKeyStore := store.NewApiKeyStore(db)
	tenantHandler := NewTenantHandler(tenantStore)
	apiKeyHandler := NewApiKeyHandler(apiKeyStore)

	// 创建租户
	tenantBody := `{"name":"test-tenant"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(tenantBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tenantHandler.Create(w, req)

	var tenant model.TenantResponse
	json.NewDecoder(w.Body).Decode(&tenant)

	// 创建 Key
	keyBody := fmt.Sprintf(`{"tenant_id":"%s","name":"test-key","scopes":"*","expires_at":"2027-01-01T00:00:00Z"}`, tenant.ID)
	req2 := httptest.NewRequest("POST", "/api/keys", strings.NewReader(keyBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	apiKeyHandler.Create(w2, req2)

	var created model.ApiKeyResponse
	json.NewDecoder(w2.Body).Decode(&created)

	// 查询
	req3 := httptest.NewRequest("GET", "/api/keys/"+created.ID, nil)
	req3 = req3.WithContext(withChiURLParam(req3.Context(), "id", created.ID))
	w3 := httptest.NewRecorder()
	apiKeyHandler.GetByID(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w3.Code)
	}

	var resp model.ApiKeyResponse
	json.NewDecoder(w3.Body).Decode(&resp)
	if resp.Name != "test-key" {
		t.Errorf("expected name 'test-key', got '%s'", resp.Name)
	}
	if resp.ID != created.ID {
		t.Errorf("expected ID '%s', got '%s'", created.ID, resp.ID)
	}
	// GetByID 不应返回 full_key
	if resp.FullKey != "" {
		t.Errorf("expected empty full_key, got '%s'", resp.FullKey)
	}
}

func TestApiKeyHandler_GetByID_NotFound(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)

	req := httptest.NewRequest("GET", "/api/keys/nonexistent", nil)
	req = req.WithContext(withChiURLParam(req.Context(), "id", "nonexistent"))
	w := httptest.NewRecorder()
	handler.GetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestApiKeyHandler_Update(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	tenantStore := store.NewTenantStore(db)
	apiKeyStore := store.NewApiKeyStore(db)
	tenantHandler := NewTenantHandler(tenantStore)
	apiKeyHandler := NewApiKeyHandler(apiKeyStore)

	// 创建租户
	tenantBody := `{"name":"test-tenant"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(tenantBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tenantHandler.Create(w, req)

	var tenant model.TenantResponse
	json.NewDecoder(w.Body).Decode(&tenant)

	// 创建 Key
	keyBody := fmt.Sprintf(`{"tenant_id":"%s","name":"original","scopes":"*","expires_at":"2027-01-01T00:00:00Z"}`, tenant.ID)
	req2 := httptest.NewRequest("POST", "/api/keys", strings.NewReader(keyBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	apiKeyHandler.Create(w2, req2)

	var created model.ApiKeyResponse
	json.NewDecoder(w2.Body).Decode(&created)

	// 更新
	updateBody := `{"name":"updated","is_active":false}`
	req3 := httptest.NewRequest("PUT", "/api/keys/"+created.ID, strings.NewReader(updateBody))
	req3.Header.Set("Content-Type", "application/json")
	req3 = req3.WithContext(withChiURLParam(req3.Context(), "id", created.ID))
	w3 := httptest.NewRecorder()
	apiKeyHandler.Update(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w3.Code)
	}

	var resp model.ApiKeyResponse
	json.NewDecoder(w3.Body).Decode(&resp)
	if resp.Name != "updated" {
		t.Errorf("expected name 'updated', got '%s'", resp.Name)
	}
	if resp.IsActive != false {
		t.Errorf("expected is_active false, got %v", resp.IsActive)
	}
}

func TestApiKeyHandler_Update_InvalidBody(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)

	req := httptest.NewRequest("PUT", "/api/keys/some-id", strings.NewReader(`invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withChiURLParam(req.Context(), "id", "some-id"))
	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApiKeyHandler_Create_MissingFields(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)

	// 缺少必填字段
	body := `{"name":"test-key"}`
	req := httptest.NewRequest("POST", "/api/keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- ProxyHandler tests ---

func TestProxyHandler_ChatCompletion(t *testing.T) {
	// 创建 mock 下游服务
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"choices": []map[string]interface{}{{"message": map[string]string{"content": "Hello!"}}},
		})
	}))
	defer mockServer.Close()

	// 创建基础设施
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	provider := proxy.NewMockProvider(mockServer.URL)
	handler := NewProxyHandler(provider, recorder)

	// 构造带认证上下文的请求
	body := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyTenantID, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyKeyID, "key-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyScopes, "*"))

	w := httptest.NewRecorder()
	handler.ChatCompletion(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["id"] != "chatcmpl-123" {
		t.Errorf("expected id 'chatcmpl-123', got '%v'", resp["id"])
	}

	// 验证用量记录已写入
	time.Sleep(200 * time.Millisecond) // 等待异步刷新
	records, err := usageStore.Query(&model.UsageQuery{TenantID: "tenant-1", Limit: 10})
	if err != nil {
		t.Fatalf("failed to query usage: %v", err)
	}
	if len(records) == 0 {
		t.Error("expected usage records to be created")
	}
	if len(records) > 0 && records[0].Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got '%s'", records[0].Model)
	}
}

func TestProxyHandler_ChatCompletion_InvalidBody(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	// Provider 不会被调用，传 nil 即可
	handler := NewProxyHandler(nil, recorder)

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyTenantID, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyKeyID, "key-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyScopes, "*"))

	w := httptest.NewRecorder()
	handler.ChatCompletion(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProxyHandler_ChatCompletion_MissingModel(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	handler := NewProxyHandler(nil, recorder)

	body := `{"messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyTenantID, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyKeyID, "key-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyScopes, "*"))

	w := httptest.NewRecorder()
	handler.ChatCompletion(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProxyHandler_ChatCompletion_MissingMessages(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	handler := NewProxyHandler(nil, recorder)

	body := `{"model":"gpt-4"}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyTenantID, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyKeyID, "key-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyScopes, "*"))

	w := httptest.NewRecorder()
	handler.ChatCompletion(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProxyHandler_ChatCompletion_ScopeDenied(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	handler := NewProxyHandler(nil, recorder)

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyTenantID, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyKeyID, "key-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyScopes, "claude-3")) // 只允许 claude-3

	w := httptest.NewRecorder()
	handler.ChatCompletion(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestProxyHandler_ChatCompletion_UpstreamError(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	// 使用会失败的 mock provider
	provider := &mockProvider{
		chatCompletionFn: func(body io.Reader) (*http.Response, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	handler := NewProxyHandler(provider, recorder)

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyTenantID, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyKeyID, "key-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyScopes, "*"))

	w := httptest.NewRecorder()
	handler.ChatCompletion(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestProxyHandler_ChatCompletion_Non200Upstream(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	// 创建返回 429 的 mock 下游
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer mockServer.Close()

	provider := proxy.NewMockProvider(mockServer.URL)
	handler := NewProxyHandler(provider, recorder)

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyTenantID, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyKeyID, "key-1"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyScopes, "*"))

	w := httptest.NewRecorder()
	handler.ChatCompletion(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Body.String() != `{"error":"rate limited"}` {
		t.Errorf("expected body '{\"error\":\"rate limited\"}', got '%s'", w.Body.String())
	}
}

func TestProxyHandler_ListModels(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string][]string{"models": {"gpt-4", "gpt-3.5"}})
	}))
	defer mockServer.Close()

	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	provider := proxy.NewMockProvider(mockServer.URL)
	handler := NewProxyHandler(provider, recorder)

	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	handler.ListModels(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProxyHandler_ListModels_UpstreamError(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	recorder := usage.NewRecorder(usageStore, 100*time.Millisecond, 10)
	defer recorder.Stop()

	provider := &mockProvider{
		listModelsFn: func() (*http.Response, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	handler := NewProxyHandler(provider, recorder)

	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	handler.ListModels(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

// --- UsageHandler tests ---

func TestUsageHandler_Query(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)

	// 先创建一些用量记录
	records := []*model.UsageRecord{
		{TenantID: "tenant-1", KeyID: "key-1", Model: "gpt-4", PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30, RequestID: "req-1"},
		{TenantID: "tenant-1", KeyID: "key-2", Model: "claude-3", PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15, RequestID: "req-2"},
	}
	if err := usageStore.BatchCreate(records); err != nil {
		t.Fatalf("failed to create usage records: %v", err)
	}

	// 查询全部
	req := httptest.NewRequest("GET", "/api/usage?limit=10", nil)
	w := httptest.NewRecorder()
	handler.Query(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 2 {
		t.Errorf("expected 2 records, got %v", resp)
	}
	total, ok := resp["total"].(float64)
	if !ok || int(total) != 2 {
		t.Errorf("expected total 2, got %v", total)
	}
}

func TestUsageHandler_Query_WithFilters(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)

	records := []*model.UsageRecord{
		{TenantID: "tenant-1", KeyID: "key-1", Model: "gpt-4", PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30, RequestID: "req-1"},
		{TenantID: "tenant-1", KeyID: "key-2", Model: "claude-3", PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15, RequestID: "req-2"},
		{TenantID: "tenant-2", KeyID: "key-3", Model: "gpt-4", PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3, RequestID: "req-3"},
	}
	if err := usageStore.BatchCreate(records); err != nil {
		t.Fatalf("failed to create usage records: %v", err)
	}

	// 按 tenant_id 过滤
	req := httptest.NewRequest("GET", "/api/usage?tenant_id=tenant-1&limit=10", nil)
	w := httptest.NewRecorder()
	handler.Query(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 2 {
		t.Errorf("expected 2 records for tenant-1, got %v", resp)
	}

	// 按 model 过滤
	req2 := httptest.NewRequest("GET", "/api/usage?model=claude-3&limit=10", nil)
	w2 := httptest.NewRecorder()
	handler.Query(w2, req2)

	json.NewDecoder(w2.Body).Decode(&resp)
	data2, ok := resp["data"].([]interface{})
	if !ok || len(data2) != 1 {
		t.Errorf("expected 1 record for claude-3, got %v", resp)
	}
}

func TestUsageHandler_Query_EmptyResult(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)

	req := httptest.NewRequest("GET", "/api/usage?limit=10", nil)
	w := httptest.NewRecorder()
	handler.Query(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 0 {
		t.Errorf("expected empty data, got %v", resp)
	}
}

func TestUsageHandler_Summary(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)

	// 创建一些记录
	now := time.Now().UTC()
	records := []*model.UsageRecord{
		{TenantID: "tenant-1", KeyID: "key-1", Model: "gpt-4", PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30, RequestID: "req-1"},
		{TenantID: "tenant-1", KeyID: "key-1", Model: "gpt-4", PromptTokens: 5, CompletionTokens: 5, TotalTokens: 10, RequestID: "req-2"},
	}
	if err := usageStore.BatchCreate(records); err != nil {
		t.Fatalf("failed to create usage records: %v", err)
	}

	start := now.Add(-1 * time.Hour).Format(time.RFC3339)
	end := now.Add(1 * time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest("GET", "/api/usage/summary?tenant_id=tenant-1&start="+start+"&end="+end, nil)
	w := httptest.NewRecorder()
	handler.Summary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var summary model.UsageSummary
	json.NewDecoder(w.Body).Decode(&summary)
	if summary.TotalPromptTokens != 15 {
		t.Errorf("expected TotalPromptTokens 15, got %d", summary.TotalPromptTokens)
	}
	if summary.TotalCompletionTokens != 25 {
		t.Errorf("expected TotalCompletionTokens 25, got %d", summary.TotalCompletionTokens)
	}
	if summary.TotalTokens != 40 {
		t.Errorf("expected TotalTokens 40, got %d", summary.TotalTokens)
	}
	if summary.RequestCount != 2 {
		t.Errorf("expected RequestCount 2, got %d", summary.RequestCount)
	}
}

func TestUsageHandler_Summary_MissingTenantID(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)

	req := httptest.NewRequest("GET", "/api/usage/summary", nil)
	w := httptest.NewRecorder()
	handler.Summary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Edge case and coverage improvement tests ---

func TestToTenantResponse_Nil(t *testing.T) {
	resp := toTenantResponse(nil)
	if resp != nil {
		t.Errorf("expected nil, got %v", resp)
	}
}

func TestToApiKeyResponse_Nil(t *testing.T) {
	resp := toApiKeyResponse(nil)
	if resp != nil {
		t.Errorf("expected nil, got %v", resp)
	}
}

func TestWriteError_DefaultCase(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusMultipleChoices, "redirect")
	if w.Code != http.StatusMultipleChoices {
		t.Errorf("expected 300, got %d", w.Code)
	}
	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error.Type != "unknown" {
		t.Errorf("expected type 'unknown', got '%s'", errResp.Error.Type)
	}
}

func TestDashboard(t *testing.T) {
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()
	Dashboard().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/html; charset=utf-8', got '%s'", ct)
	}

	// 验证 HTML 关键结构存在
	body := w.Body.String()
	checks := []struct {
		name string
		fn   func() bool
	}{
		{"包含页面标题", func() bool { return strings.Contains(body, "AI Gateway 管理面板") }},
		{"包含侧边栏导航", func() bool { return strings.Contains(body, "sidebar") }},
		{"包含概览页面", func() bool { return strings.Contains(body, "page-overview") }},
		{"包含租户管理页面", func() bool { return strings.Contains(body, "page-tenants") }},
		{"包含 Key 管理页面", func() bool { return strings.Contains(body, "page-keys") }},
		{"包含用量监控页面", func() bool { return strings.Contains(body, "page-usage") }},
		{"包含租户模态框", func() bool { return strings.Contains(body, "tenant-modal") }},
		{"包含 Key 模态框", func() bool { return strings.Contains(body, "key-modal") }},
		{"包含 Toast 通知", func() bool { return strings.Contains(body, "toast") }},
		{"包含 JS API 客户端", func() bool { return strings.Contains(body, "async function api") }},
		{"包含 loadOverview 函数", func() bool { return strings.Contains(body, "function loadOverview") }},
		{"包含 loadTenants 函数", func() bool { return strings.Contains(body, "function loadTenants") }},
		{"包含 loadKeys 函数", func() bool { return strings.Contains(body, "function loadKeys") }},
		{"包含 loadUsageSummary 函数", func() bool { return strings.Contains(body, "function loadUsageSummary") }},
		{"包含 escapeHtml 函数", func() bool { return strings.Contains(body, "function escapeHtml") }},
		{"包含 showPage 导航函数", func() bool { return strings.Contains(body, "function showPage") }},
		{"包含 API_BASE 地址配置", func() bool { return strings.Contains(body, "API_BASE = ''") }},
		{"包含复制 Key 功能", func() bool { return strings.Contains(body, "function copyFullKey") }},
		{"包含日期范围查询控件", func() bool { return strings.Contains(body, "usage-start") }},
		{"包含 stats 卡片区域", func() bool { return strings.Contains(body, "overview-cards") }},
	}
	for _, c := range checks {
		if !c.fn() {
			t.Errorf("Dashboard HTML 缺失: %s", c.name)
		}
	}
}

func TestEstimateTokens_EmptyContent(t *testing.T) {
	messages := []map[string]string{{"role": "user", "content": ""}}
	tokens := estimateTokens(messages)
	if tokens != 1 {
		t.Errorf("expected 1 token for empty content, got %d", tokens)
	}
}

// --- Store error tests (via closed database) ---

func TestTenantHandler_Create_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)
	db.Close() // force store error

	body := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/api/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestTenantHandler_GetByID_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)
	db.Close()

	req := httptest.NewRequest("GET", "/api/tenants/some-id", nil)
	req = req.WithContext(withChiURLParam(req.Context(), "id", "some-id"))
	w := httptest.NewRecorder()
	handler.GetByID(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestTenantHandler_List_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)
	db.Close()

	req := httptest.NewRequest("GET", "/api/tenants", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestTenantHandler_Update_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)
	db.Close()

	body := `{"name":"updated"}`
	req := httptest.NewRequest("PUT", "/api/tenants/some-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withChiURLParam(req.Context(), "id", "some-id"))
	w := httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestTenantHandler_Delete_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	tenantStore := store.NewTenantStore(db)
	handler := NewTenantHandler(tenantStore)
	db.Close()

	req := httptest.NewRequest("DELETE", "/api/tenants/some-id", nil)
	req = req.WithContext(withChiURLParam(req.Context(), "id", "some-id"))
	w := httptest.NewRecorder()
	handler.Delete(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestApiKeyHandler_Create_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)
	db.Close()

	body := `{"tenant_id":"t1","name":"key","scopes":"*"}`
	req := httptest.NewRequest("POST", "/api/keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestApiKeyHandler_GetByID_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)
	db.Close()

	req := httptest.NewRequest("GET", "/api/keys/some-id", nil)
	req = req.WithContext(withChiURLParam(req.Context(), "id", "some-id"))
	w := httptest.NewRecorder()
	handler.GetByID(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestApiKeyHandler_List_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)
	db.Close()

	req := httptest.NewRequest("GET", "/api/keys", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestApiKeyHandler_Update_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)
	db.Close()

	body := `{"name":"updated"}`
	req := httptest.NewRequest("PUT", "/api/keys/some-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withChiURLParam(req.Context(), "id", "some-id"))
	w := httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestApiKeyHandler_Delete_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	apiKeyStore := store.NewApiKeyStore(db)
	handler := NewApiKeyHandler(apiKeyStore)
	db.Close()

	req := httptest.NewRequest("DELETE", "/api/keys/some-id", nil)
	req = req.WithContext(withChiURLParam(req.Context(), "id", "some-id"))
	w := httptest.NewRecorder()
	handler.Delete(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestUsageHandler_Query_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)
	db.Close()

	req := httptest.NewRequest("GET", "/api/usage?limit=10", nil)
	w := httptest.NewRecorder()
	handler.Query(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestUsageHandler_Summary_StoreError(t *testing.T) {
	db := setupHandlerDB(t)
	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)
	db.Close()

	req := httptest.NewRequest("GET", "/api/usage/summary?tenant_id=t1", nil)
	w := httptest.NewRecorder()
	handler.Summary(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UsageHandler query parameter edge cases ---

func TestUsageHandler_Query_InvalidTimeFormat(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)

	record := &model.UsageRecord{
		TenantID: "t1", Model: "gpt-4", PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30, RequestID: "req-1",
	}
	if err := usageStore.Create(record); err != nil {
		t.Fatalf("failed to create usage record: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/usage?start=invalid-time&end=also-invalid&limit=10", nil)
	w := httptest.NewRecorder()
	handler.Query(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUsageHandler_Query_InvalidPagination(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)

	record := &model.UsageRecord{
		TenantID: "t1", Model: "gpt-4", PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30, RequestID: "req-1",
	}
	if err := usageStore.Create(record); err != nil {
		t.Fatalf("failed to create usage record: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/usage?limit=abc&offset=xyz", nil)
	w := httptest.NewRecorder()
	handler.Query(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/usage?limit=0", nil)
	w2 := httptest.NewRecorder()
	handler.Query(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}

	req3 := httptest.NewRequest("GET", "/api/usage?offset=-1", nil)
	w3 := httptest.NewRecorder()
	handler.Query(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w3.Code)
	}
}

func TestUsageHandler_Summary_InvalidTime(t *testing.T) {
	db := setupHandlerDB(t)
	defer db.Close()

	usageStore := store.NewUsageStore(db)
	handler := NewUsageHandler(usageStore)

	record := &model.UsageRecord{
		TenantID: "t1", Model: "gpt-4", PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30, RequestID: "req-1",
	}
	if err := usageStore.Create(record); err != nil {
		t.Fatalf("failed to create usage record: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/usage/summary?tenant_id=t1&start=invalid&end=invalid", nil)
	w := httptest.NewRecorder()
	handler.Summary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

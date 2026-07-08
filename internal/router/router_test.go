package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrviu/ai-gateway/internal/config"
	"github.com/vrviu/ai-gateway/internal/store"
)

func setupRouter(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := config.Load()
	r, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	return httptest.NewServer(r)
}

func TestRouter_HealthEndpoint(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
}

func TestRouter_DashboardEndpoint(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	tests := []struct {
		name string
		path string
	}{
		{"without slash", "/dashboard"},
		{"with slash", "/dashboard/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tt.path)
			if err != nil {
				t.Fatalf("unexpected request error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}
			if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
				t.Errorf("expected text/html, got %s", ct)
			}
		})
	}
}

func TestRouter_V1Endpoints_RequireAuth(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"chat completions", "POST", "/v1/chat/completions"},
		{"list models", "GET", "/v1/models"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unexpected request error: %v", err)
			}
			defer resp.Body.Close()

			// Should fail auth (401) because no API key is provided
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", resp.StatusCode)
			}
		})
	}
}

func TestRouter_ApiEndpoints(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	// Tenant CRUD
	t.Run("POST /api/tenants", func(t *testing.T) {
		resp, err := http.Post(srv.URL+"/api/tenants", "application/json", nil)
		if err != nil {
			t.Fatalf("unexpected request error: %v", err)
		}
		defer resp.Body.Close()

		// Should fail with 400 (invalid JSON) rather than 404
		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("expected non-404 status, got 404 — route not registered")
		}
	})

	t.Run("GET /api/tenants", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/tenants")
		if err != nil {
			t.Fatalf("unexpected request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("expected non-404 status, got 404 — route not registered")
		}
	})

	// API keys
	t.Run("POST /api/keys", func(t *testing.T) {
		resp, err := http.Post(srv.URL+"/api/keys", "application/json", nil)
		if err != nil {
			t.Fatalf("unexpected request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("expected non-404 status, got 404 — route not registered")
		}
	})

	// Usage
	t.Run("GET /api/usage", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/usage")
		if err != nil {
			t.Fatalf("unexpected request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("expected non-404 status, got 404 — route not registered")
		}
	})
}

func TestRouter_NotFound(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

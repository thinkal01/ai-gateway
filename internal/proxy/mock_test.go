package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMockProvider_ChatCompletion(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/mock/chat/completions" {
			t.Errorf("expected /mock/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer mockServer.Close()

	provider := NewMockProvider(mockServer.URL)
	body := strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`)

	resp, err := provider.ChatCompletion(body)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMockProvider_ChatCompletion_ServerError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer mockServer.Close()

	provider := NewMockProvider(mockServer.URL)
	body := strings.NewReader(`{"model":"gpt-4"}`)

	resp, err := provider.ChatCompletion(body)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestMockProvider_ListModels(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/mock/models" {
			t.Errorf("expected /mock/models, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string][]string{"models": {"gpt-4", "gpt-3.5"}})
	}))
	defer mockServer.Close()

	provider := NewMockProvider(mockServer.URL)
	resp, err := provider.ListModels()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMockProvider_BaseURLTrim(t *testing.T) {
	// 测试带尾斜杠的 URL 被正确处理
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mock/chat/completions" {
			t.Errorf("expected /mock/chat/completions, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	provider := NewMockProvider(mockServer.URL + "/") // 带尾斜杠
	body := strings.NewReader(`{}`)
	resp, err := provider.ChatCompletion(body)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer resp.Body.Close()
}

func TestMockProvider_ConnectionRefused(t *testing.T) {
	provider := NewMockProvider("http://127.0.0.1:1") // 极不可能有服务的端口
	body := strings.NewReader(`{}`)
	_, err := provider.ChatCompletion(body)
	if err == nil {
		t.Error("expected error for connection refused, got nil")
	}
}

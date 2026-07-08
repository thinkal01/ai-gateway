package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/store"
)

// mockApiKeyStore 模拟 ApiKeyStore
type mockApiKeyStore struct {
	store.ApiKeyStore
	keys map[string]*model.ApiKey // key: prefix -> apiKey
}

func newMockApiKeyStore() *mockApiKeyStore {
	return &mockApiKeyStore{
		keys: make(map[string]*model.ApiKey),
	}
}

func (m *mockApiKeyStore) FindByPrefix(prefix string) (*model.ApiKey, error) {
	k, ok := m.keys[prefix]
	if !ok {
		return nil, nil
	}
	return k, nil
}

func (m *mockApiKeyStore) UpdateLastUsed(id string, t time.Time) error {
	return nil
}

func setupTestKey(t *testing.T, store *mockApiKeyStore, prefix, hash, scopes string, isActive bool, expiresAt time.Time) {
	t.Helper()
	store.keys[prefix] = &model.ApiKey{
		ID:        "test-key-id",
		TenantID:  "test-tenant-id",
		KeyPrefix: prefix,
		KeyHash:   hash,
		Scopes:    scopes,
		IsActive:  isActive,
		ExpiresAt: expiresAt,
	}
}

func TestValidateKey_Valid(t *testing.T) {
	mockStore := newMockApiKeyStore()
	svc := NewService(mockStore, 8)

	// 生成一个 key
	fullKey, prefix, hash, err := GenerateKey(8, 16)
	if err != nil {
		t.Fatal(err)
	}

	setupTestKey(t, mockStore, prefix, hash, "*", true, time.Now().UTC().Add(24*time.Hour))

	result, err := svc.ValidateKey(fullKey, "gpt-4")
	if err != nil {
		t.Fatalf("expected valid key, got error: %v", err)
	}
	if result.TenantID != "test-tenant-id" {
		t.Errorf("expected tenant id 'test-tenant-id', got '%s'", result.TenantID)
	}
	if result.KeyID != "test-key-id" {
		t.Errorf("expected key id 'test-key-id', got '%s'", result.KeyID)
	}
}

func TestValidateKey_InvalidPrefix(t *testing.T) {
	mockStore := newMockApiKeyStore()
	svc := NewService(mockStore, 8)

	// 使用一个不存在的 key
	_, err := svc.ValidateKey("sk-nonexistentkey123", "gpt-4")
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
}

func TestValidateKey_HashMismatch(t *testing.T) {
	mockStore := newMockApiKeyStore()
	svc := NewService(mockStore, 8)

	// 生成一个 key，但用不同哈希存入
	fullKey, prefix, _, err := GenerateKey(8, 16)
	if err != nil {
		t.Fatal(err)
	}

	// 手动构造一个不同的哈希（确保不同）
	h := sha256.Sum256([]byte("some-different-key-value"))
	differentHash := hex.EncodeToString(h[:])

	setupTestKey(t, mockStore, prefix, differentHash, "*", true, time.Now().UTC().Add(24*time.Hour))

	_, err = svc.ValidateKey(fullKey, "gpt-4")
	if err == nil {
		t.Fatal("expected error for hash mismatch, got nil")
	}
}

func TestValidateKey_DisabledKey(t *testing.T) {
	mockStore := newMockApiKeyStore()
	svc := NewService(mockStore, 8)

	fullKey, prefix, hash, err := GenerateKey(8, 16)
	if err != nil {
		t.Fatal(err)
	}

	setupTestKey(t, mockStore, prefix, hash, "*", false, time.Now().UTC().Add(24*time.Hour))

	_, err = svc.ValidateKey(fullKey, "gpt-4")
	if err == nil {
		t.Fatal("expected error for disabled key, got nil")
	}
}

func TestValidateKey_ExpiredKey(t *testing.T) {
	mockStore := newMockApiKeyStore()
	svc := NewService(mockStore, 8)

	fullKey, prefix, hash, err := GenerateKey(8, 16)
	if err != nil {
		t.Fatal(err)
	}

	setupTestKey(t, mockStore, prefix, hash, "*", true, time.Now().UTC().Add(-1*time.Hour))

	_, err = svc.ValidateKey(fullKey, "gpt-4")
	if err == nil {
		t.Fatal("expected error for expired key, got nil")
	}
}

func TestValidateKey_InsufficientScope(t *testing.T) {
	mockStore := newMockApiKeyStore()
	svc := NewService(mockStore, 8)

	fullKey, prefix, hash, err := GenerateKey(8, 16)
	if err != nil {
		t.Fatal(err)
	}

	// 只允许 gpt-3.5，但请求 gpt-4
	setupTestKey(t, mockStore, prefix, hash, "gpt-3.5", true, time.Now().UTC().Add(24*time.Hour))

	_, err = svc.ValidateKey(fullKey, "gpt-4")
	if err == nil {
		t.Fatal("expected error for insufficient scope, got nil")
	}
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		length int
		want   string
	}{
		{"normal key", "sk-abcdefghijklmnop", 8, "abcdefgh"},
		{"short key", "abc", 8, ""},
		{"empty key", "", 8, ""},
		{"with whitespace", "  sk-test1234abcd  ", 8, "test1234"},
		{"without sk prefix", "rawkeyhere", 5, "rawke"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPrefix(tt.input, tt.length)
			if got != tt.want {
				t.Errorf("extractPrefix(%q, %d) = %q, want %q", tt.input, tt.length, got, tt.want)
			}
		})
	}
}

func TestHashKey(t *testing.T) {
	// 相同输入产生相同哈希
	h1 := hashKey("sk-test-key-123")
	h2 := hashKey("sk-test-key-123")
	if h1 != h2 {
		t.Error("expected same hash for same input")
	}

	// 不同输入产生不同哈希
	h3 := hashKey("sk-other-key-456")
	if h1 == h3 {
		t.Error("expected different hash for different input")
	}
}

func TestGenerateKey(t *testing.T) {
	fullKey, prefix, hash, err := GenerateKey(8, 16)
	if err != nil {
		t.Fatal(err)
	}

	// 验证格式：sk-{prefix}{secret}
	if len(fullKey) < 5 {
		t.Fatal("key too short")
	}
	if fullKey[:3] != "sk-" {
		t.Errorf("expected key to start with 'sk-', got %q", fullKey[:3])
	}

	// 验证前缀是 key 中 "sk-" 之后的8位
	expectedPrefix := fullKey[3 : 3+8]
	if prefix != expectedPrefix {
		t.Errorf("expected prefix %q, got %q", expectedPrefix, prefix)
	}

	// 验证哈希正确
	expectedHash := hashKey(fullKey)
	if hash != expectedHash {
		t.Errorf("expected hash %q, got %q", expectedHash, hash)
	}
}

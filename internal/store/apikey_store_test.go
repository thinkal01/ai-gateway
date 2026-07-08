package store

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/vrviu/ai-gateway/internal/model"
)

func setupTestTenant(t *testing.T, db *sql.DB) *model.Tenant {
	t.Helper()
	s := NewTenantStore(db)
	tenant := &model.Tenant{Name: "test-tenant-for-key", IsActive: true}
	if err := s.Create(tenant); err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}
	return tenant
}

func TestApiKeyStore_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewApiKeyStore(db)

	key := &model.ApiKey{
		TenantID:  tenant.ID,
		Name:      "test-key",
		KeyPrefix: "prefix123",
		KeyHash:   "hash123",
		Scopes:    "gpt-4,gpt-3.5",
		IsActive:  true,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	err := s.Create(key)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if key.ID == "" {
		t.Error("expected key ID to be set")
	}
}

func TestApiKeyStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewApiKeyStore(db)

	key := &model.ApiKey{
		TenantID:  tenant.ID,
		Name:      "get-key",
		KeyPrefix: "prefix456",
		KeyHash:   "hash456",
		Scopes:    "*",
		IsActive:  true,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	s.Create(key)

	got, err := s.GetByID(key.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got == nil {
		t.Fatal("expected key, got nil")
	}
	if got.Name != "get-key" {
		t.Errorf("expected name 'get-key', got '%s'", got.Name)
	}
}

func TestApiKeyStore_FindByPrefix(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewApiKeyStore(db)

	key := &model.ApiKey{
		TenantID:  tenant.ID,
		Name:      "prefix-search",
		KeyPrefix: "findme123",
		KeyHash:   "hash789",
		Scopes:    "*",
		IsActive:  true,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	s.Create(key)

	got, err := s.FindByPrefix("findme123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got == nil {
		t.Fatal("expected key, got nil")
	}
	if got.KeyPrefix != "findme123" {
		t.Errorf("expected prefix 'findme123', got '%s'", got.KeyPrefix)
	}
}

func TestApiKeyStore_FindByPrefix_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewApiKeyStore(db)
	got, err := s.FindByPrefix("nonexistent")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for not found, got key")
	}
}

func TestApiKeyStore_ListByTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewApiKeyStore(db)

	// 创建2个 key，使用不同的前缀和哈希
	for i := 0; i < 2; i++ {
		prefix := fmt.Sprintf("listpref%d", i)
		hash := fmt.Sprintf("hash%d", i)
		err := s.Create(&model.ApiKey{
			TenantID:  tenant.ID,
			Name:      "key",
			KeyPrefix: prefix,
			KeyHash:   hash,
			Scopes:    "*",
			IsActive:  true,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("failed to create test key %d: %v", i, err)
		}
	}

	keys, err := s.ListByTenant(tenant.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestApiKeyStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewApiKeyStore(db)

	key := &model.ApiKey{
		TenantID:  tenant.ID,
		Name:      "before-update",
		KeyPrefix: "updprefix",
		KeyHash:   "uphash",
		Scopes:    "*",
		IsActive:  true,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	s.Create(key)

	newName := "after-update"
	newScopes := "gpt-4"
	falseVal := false
	err := s.Update(key.ID, &model.UpdateApiKeyRequest{
		Name:     &newName,
		Scopes:   &newScopes,
		IsActive: &falseVal,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := s.GetByID(key.ID)
	if got.Name != "after-update" {
		t.Errorf("expected name 'after-update', got '%s'", got.Name)
	}
	if got.Scopes != "gpt-4" {
		t.Errorf("expected scopes 'gpt-4', got '%s'", got.Scopes)
	}
	if got.IsActive {
		t.Error("expected is_active to be false")
	}
}

func TestApiKeyStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewApiKeyStore(db)

	key := &model.ApiKey{
		TenantID:  tenant.ID,
		Name:      "to-delete",
		KeyPrefix: "delprefix",
		KeyHash:   "delhash",
		Scopes:    "*",
		IsActive:  true,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	s.Create(key)

	err := s.Delete(key.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := s.GetByID(key.ID)
	if got != nil {
		t.Error("expected key to be deleted")
	}
}

func TestApiKeyStore_UpdateLastUsed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewApiKeyStore(db)

	key := &model.ApiKey{
		TenantID:  tenant.ID,
		Name:      "last-used",
		KeyPrefix: "lastused",
		KeyHash:   "luhash",
		Scopes:    "*",
		IsActive:  true,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	s.Create(key)

	now := time.Now().UTC()
	err := s.UpdateLastUsed(key.ID, now)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := s.GetByID(key.ID)
	if got.LastUsedAt == nil {
		t.Fatal("expected last_used_at to be set")
	}
}

package store

import (
	"database/sql"
	"testing"

	"github.com/vrviu/ai-gateway/internal/model"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}
	return db
}

func TestTenantStore_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewTenantStore(db)
	tenant := &model.Tenant{Name: "test-tenant", IsActive: true}

	err := s.Create(tenant)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if tenant.ID == "" {
		t.Error("expected tenant ID to be set")
	}
}

func TestTenantStore_CreateDuplicateName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewTenantStore(db)

	s.Create(&model.Tenant{Name: "duplicate", IsActive: true})
	err := s.Create(&model.Tenant{Name: "duplicate", IsActive: true})
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestTenantStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewTenantStore(db)
	tenant := &model.Tenant{Name: "get-test", IsActive: true}
	s.Create(tenant)

	got, err := s.GetByID(tenant.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got == nil {
		t.Fatal("expected tenant, got nil")
	}
	if got.Name != "get-test" {
		t.Errorf("expected name 'get-test', got '%s'", got.Name)
	}
}

func TestTenantStore_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewTenantStore(db)
	got, err := s.GetByID("nonexistent-id")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for not found, got tenant")
	}
}

func TestTenantStore_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewTenantStore(db)

	s.Create(&model.Tenant{Name: "tenant-a", IsActive: true})
	s.Create(&model.Tenant{Name: "tenant-b", IsActive: true})

	tenants, err := s.List()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(tenants) != 2 {
		t.Errorf("expected 2 tenants, got %d", len(tenants))
	}
}

func TestTenantStore_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewTenantStore(db)
	tenants, err := s.List()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(tenants) != 0 {
		t.Errorf("expected 0 tenants, got %d", len(tenants))
	}
}

func TestTenantStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewTenantStore(db)
	tenant := &model.Tenant{Name: "before-update", IsActive: true}
	s.Create(tenant)

	newName := "after-update"
	falseVal := false
	err := s.Update(tenant.ID, &model.UpdateTenantRequest{Name: &newName, IsActive: &falseVal})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := s.GetByID(tenant.ID)
	if got.Name != "after-update" {
		t.Errorf("expected name 'after-update', got '%s'", got.Name)
	}
	if got.IsActive {
		t.Error("expected is_active to be false")
	}
}

func TestTenantStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewTenantStore(db)
	tenant := &model.Tenant{Name: "to-delete", IsActive: true}
	s.Create(tenant)

	err := s.Delete(tenant.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := s.GetByID(tenant.ID)
	if got != nil {
		t.Error("expected tenant to be deleted")
	}
}

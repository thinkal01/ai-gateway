package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrviu/ai-gateway/internal/model"
)

// TenantStore 租户存储接口
type TenantStore interface {
	Create(tenant *model.Tenant) error
	GetByID(id string) (*model.Tenant, error)
	List() ([]*model.Tenant, error)
	Update(id string, req *model.UpdateTenantRequest) error
	Delete(id string) error
}

type tenantStore struct {
	db *sql.DB
}

// NewTenantStore 创建 TenantStore 实例
func NewTenantStore(db *sql.DB) TenantStore {
	return &tenantStore{db: db}
}

func (s *tenantStore) Create(tenant *model.Tenant) error {
	tenant.ID = uuid.New().String()
	tenant.CreatedAt = time.Now().UTC()
	tenant.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(
		`INSERT INTO tenants (id, name, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		tenant.ID, tenant.Name, tenant.IsActive, tenant.CreatedAt.Format(time.RFC3339), tenant.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

func (s *tenantStore) GetByID(id string) (*model.Tenant, error) {
	row := s.db.QueryRow(
		`SELECT id, name, is_active, created_at, updated_at FROM tenants WHERE id = ?`, id,
	)

	var t model.Tenant
	var createdAt, updatedAt string
	err := row.Scan(&t.ID, &t.Name, &t.IsActive, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &t, nil
}

func (s *tenantStore) List() ([]*model.Tenant, error) {
	rows, err := s.db.Query(
		`SELECT id, name, is_active, created_at, updated_at FROM tenants ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*model.Tenant
	for rows.Next() {
		var t model.Tenant
		var createdAt, updatedAt string
		if err := rows.Scan(&t.ID, &t.Name, &t.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		tenants = append(tenants, &t)
	}
	return tenants, rows.Err()
}

func (s *tenantStore) Update(id string, req *model.UpdateTenantRequest) error {
	if req.Name != nil {
		_, err := s.db.Exec(`UPDATE tenants SET name = ?, updated_at = ? WHERE id = ?`, *req.Name, time.Now().UTC().Format(time.RFC3339), id)
		if err != nil {
			return fmt.Errorf("update tenant name: %w", err)
		}
	}
	if req.IsActive != nil {
		_, err := s.db.Exec(`UPDATE tenants SET is_active = ?, updated_at = ? WHERE id = ?`, *req.IsActive, time.Now().UTC().Format(time.RFC3339), id)
		if err != nil {
			return fmt.Errorf("update tenant is_active: %w", err)
		}
	}
	return nil
}

func (s *tenantStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM tenants WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	return nil
}

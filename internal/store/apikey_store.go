package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrviu/ai-gateway/internal/model"
)

// ApiKeyStore API Key 存储接口
type ApiKeyStore interface {
	Create(key *model.ApiKey) error
	GetByID(id string) (*model.ApiKey, error)
	FindByPrefix(prefix string) (*model.ApiKey, error)
	List() ([]*model.ApiKey, error)
	ListByTenant(tenantID string) ([]*model.ApiKey, error)
	Update(id string, req *model.UpdateApiKeyRequest) error
	Delete(id string) error
	UpdateLastUsed(id string, t time.Time) error
}

type apiKeyStore struct {
	db *sql.DB
}

// NewApiKeyStore 创建 ApiKeyStore 实例
func NewApiKeyStore(db *sql.DB) ApiKeyStore {
	return &apiKeyStore{db: db}
}

func (s *apiKeyStore) Create(key *model.ApiKey) error {
	key.ID = uuid.New().String()
	key.CreatedAt = time.Now().UTC()
	key.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(
		`INSERT INTO api_keys (id, tenant_id, name, key_prefix, key_hash, scopes, is_active, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		key.ID, key.TenantID, key.Name, key.KeyPrefix, key.KeyHash, key.Scopes, key.IsActive, key.ExpiresAt.Format(time.RFC3339), key.CreatedAt.Format(time.RFC3339), key.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

func (s *apiKeyStore) GetByID(id string) (*model.ApiKey, error) {
	row := s.db.QueryRow(
		`SELECT id, tenant_id, name, key_prefix, key_hash, scopes, is_active, expires_at, last_used_at, created_at, updated_at FROM api_keys WHERE id = ?`, id,
	)

	var k model.ApiKey
	var expiresAt, createdAt, updatedAt string
	var lastUsedAt *string
	err := row.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyPrefix, &k.KeyHash, &k.Scopes, &k.IsActive, &expiresAt, &lastUsedAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}

	k.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	k.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	k.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if lastUsedAt != nil {
		t, _ := time.Parse(time.RFC3339, *lastUsedAt)
		k.LastUsedAt = &t
	}
	return &k, nil
}

func (s *apiKeyStore) FindByPrefix(prefix string) (*model.ApiKey, error) {
	row := s.db.QueryRow(
		`SELECT id, tenant_id, name, key_prefix, key_hash, scopes, is_active, expires_at, last_used_at, created_at, updated_at FROM api_keys WHERE key_prefix = ?`, prefix,
	)

	var k model.ApiKey
	var expiresAt, createdAt, updatedAt string
	var lastUsedAt *string
	err := row.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyPrefix, &k.KeyHash, &k.Scopes, &k.IsActive, &expiresAt, &lastUsedAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find api key by prefix: %w", err)
	}

	k.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	k.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	k.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if lastUsedAt != nil {
		t, _ := time.Parse(time.RFC3339, *lastUsedAt)
		k.LastUsedAt = &t
	}
	return &k, nil
}

func (s *apiKeyStore) List() ([]*model.ApiKey, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, name, key_prefix, scopes, is_active, expires_at, last_used_at, created_at, updated_at FROM api_keys ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []*model.ApiKey
	for rows.Next() {
		var k model.ApiKey
		var expiresAt, createdAt, updatedAt string
		var lastUsedAt *string
		if err := rows.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.IsActive, &expiresAt, &lastUsedAt, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		k.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		k.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		k.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if lastUsedAt != nil {
			t, _ := time.Parse(time.RFC3339, *lastUsedAt)
			k.LastUsedAt = &t
		}
		keys = append(keys, &k)
	}
	return keys, rows.Err()
}

func (s *apiKeyStore) ListByTenant(tenantID string) ([]*model.ApiKey, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, name, key_prefix, scopes, is_active, expires_at, last_used_at, created_at, updated_at FROM api_keys WHERE tenant_id = ? ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []*model.ApiKey
	for rows.Next() {
		var k model.ApiKey
		var expiresAt, createdAt, updatedAt string
		var lastUsedAt *string
		if err := rows.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.IsActive, &expiresAt, &lastUsedAt, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		k.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		k.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		k.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if lastUsedAt != nil {
			t, _ := time.Parse(time.RFC3339, *lastUsedAt)
			k.LastUsedAt = &t
		}
		keys = append(keys, &k)
	}
	return keys, rows.Err()
}

func (s *apiKeyStore) Update(id string, req *model.UpdateApiKeyRequest) error {
	if req.Name != nil {
		_, err := s.db.Exec(`UPDATE api_keys SET name = ?, updated_at = ? WHERE id = ?`, *req.Name, time.Now().UTC().Format(time.RFC3339), id)
		if err != nil {
			return fmt.Errorf("update api key name: %w", err)
		}
	}
	if req.Scopes != nil {
		_, err := s.db.Exec(`UPDATE api_keys SET scopes = ?, updated_at = ? WHERE id = ?`, *req.Scopes, time.Now().UTC().Format(time.RFC3339), id)
		if err != nil {
			return fmt.Errorf("update api key scopes: %w", err)
		}
	}
	if req.IsActive != nil {
		_, err := s.db.Exec(`UPDATE api_keys SET is_active = ?, updated_at = ? WHERE id = ?`, *req.IsActive, time.Now().UTC().Format(time.RFC3339), id)
		if err != nil {
			return fmt.Errorf("update api key is_active: %w", err)
		}
	}
	if req.ExpiresAt != nil {
		_, err := s.db.Exec(`UPDATE api_keys SET expires_at = ?, updated_at = ? WHERE id = ?`, req.ExpiresAt.Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339), id)
		if err != nil {
			return fmt.Errorf("update api key expires_at: %w", err)
		}
	}
	return nil
}

func (s *apiKeyStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	return nil
}

func (s *apiKeyStore) UpdateLastUsed(id string, t time.Time) error {
	_, err := s.db.Exec(`UPDATE api_keys SET last_used_at = ?, updated_at = ? WHERE id = ?`, t.Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}
	return nil
}

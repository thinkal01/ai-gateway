package model

import "time"

// ApiKey API Key
type ApiKey struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"` // 前 N 位明文，用于查找
	KeyHash    string     `json:"key_hash"`   // 完整 Key 的哈希
	Scopes     string     `json:"scopes"`     // 逗号分隔的模型权限
	IsActive   bool       `json:"is_active"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// CreateApiKeyRequest 创建 Key 请求
type CreateApiKeyRequest struct {
	TenantID  string    `json:"tenant_id" validate:"required"`
	Name      string    `json:"name" validate:"required,min=1,max=64"`
	Scopes    string    `json:"scopes" validate:"required"`
	ExpiresAt time.Time `json:"expires_at"`
}

// UpdateApiKeyRequest 更新 Key 请求
type UpdateApiKeyRequest struct {
	Name      *string    `json:"name,omitempty"`
	Scopes    *string    `json:"scopes,omitempty"`
	IsActive  *bool      `json:"is_active,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ApiKeyResponse Key 响应（创建时返回完整 Key，后续不再返回）
type ApiKeyResponse struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	FullKey    string     `json:"full_key,omitempty"` // 仅创建时返回
	Scopes     string     `json:"scopes"`
	IsActive   bool       `json:"is_active"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// AuthResult 认证结果
type AuthResult struct {
	TenantID string
	KeyID    string
	Scopes   string
}

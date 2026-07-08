package model

import "time"

// Tenant 租户
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name string `json:"name" validate:"required,min=1,max=64"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	Name     *string `json:"name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// TenantResponse 租户响应
type TenantResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

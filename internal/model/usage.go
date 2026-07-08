package model

import "time"

// UsageRecord 用量记录
type UsageRecord struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	TenantName       string    `json:"tenant_name"`
	KeyID            string    `json:"key_id"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	RequestID        string    `json:"request_id"`
	CreatedAt        time.Time `json:"created_at"`
}

// UsageRecordRequest 创建用量记录请求
type UsageRecordRequest struct {
	TenantID         string `json:"tenant_id"`
	KeyID            string `json:"key_id"`
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	RequestID        string `json:"request_id"`
}

// UsageQuery 用量查询
type UsageQuery struct {
	TenantID string    `json:"tenant_id,omitempty"`
	KeyID    string    `json:"key_id,omitempty"`
	Model    string    `json:"model,omitempty"`
	StartAt  time.Time `json:"start_at,omitempty"`
	EndAt    time.Time `json:"end_at,omitempty"`
	Limit    int       `json:"limit,omitempty"`
	Offset   int       `json:"offset,omitempty"`
}

// UsageSummary 用量汇总
type UsageSummary struct {
	TotalPromptTokens     int `json:"total_prompt_tokens"`
	TotalCompletionTokens int `json:"total_completion_tokens"`
	TotalTokens           int `json:"total_tokens"`
	RequestCount          int `json:"request_count"`
}

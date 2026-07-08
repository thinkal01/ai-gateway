package store

import (
	"testing"
	"time"

	"github.com/vrviu/ai-gateway/internal/model"
)

func TestUsageStore_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewUsageStore(db)

	record := &model.UsageRecord{
		TenantID:         tenant.ID,
		KeyID:            "test-key-id",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		RequestID:        "req-001",
	}

	err := s.Create(record)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if record.ID == "" {
		t.Error("expected record ID to be set")
	}
}

func TestUsageStore_Query(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewUsageStore(db)

	// 插入 5 条记录
	for i := 0; i < 5; i++ {
		s.Create(&model.UsageRecord{
			TenantID:         tenant.ID,
			KeyID:            "test-key-id",
			Model:            "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			RequestID:        "req-001",
		})
	}

	records, err := s.Query(&model.UsageQuery{
		TenantID: tenant.ID,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(records) != 5 {
		t.Errorf("expected 5 records, got %d", len(records))
	}
}

func TestUsageStore_Query_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewUsageStore(db)

	for i := 0; i < 5; i++ {
		s.Create(&model.UsageRecord{
			TenantID:         tenant.ID,
			KeyID:            "test-key-id",
			Model:            "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			RequestID:        "req-001",
		})
	}

	records, err := s.Query(&model.UsageQuery{
		TenantID: tenant.ID,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestUsageStore_Query_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewUsageStore(db)
	records, err := s.Query(&model.UsageQuery{
		TenantID: "nonexistent",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestUsageStore_BatchCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewUsageStore(db)

	records := []*model.UsageRecord{
		{TenantID: tenant.ID, KeyID: "key1", Model: "gpt-4", PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, RequestID: "req-001"},
		{TenantID: tenant.ID, KeyID: "key2", Model: "gpt-3.5", PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300, RequestID: "req-002"},
		{TenantID: tenant.ID, KeyID: "key3", Model: "gpt-4", PromptTokens: 50, CompletionTokens: 25, TotalTokens: 75, RequestID: "req-003"},
	}

	err := s.BatchCreate(records)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 验证插入了 3 条
	all, _ := s.Query(&model.UsageQuery{
		TenantID: tenant.ID,
		Limit:    10,
	})
	if len(all) != 3 {
		t.Errorf("expected 3 records after batch insert, got %d", len(all))
	}
}

func TestUsageStore_GetSummary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	s := NewUsageStore(db)

	records := []*model.UsageRecord{
		{TenantID: tenant.ID, KeyID: "key1", Model: "gpt-4", PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, RequestID: "req-001"},
		{TenantID: tenant.ID, KeyID: "key2", Model: "gpt-3.5", PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300, RequestID: "req-002"},
		{TenantID: tenant.ID, KeyID: "key3", Model: "gpt-4", PromptTokens: 50, CompletionTokens: 25, TotalTokens: 75, RequestID: "req-003"},
	}
	s.BatchCreate(records)

	now := time.Now().UTC()
	summary, err := s.GetSummary(tenant.ID, now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if summary.RequestCount != 3 {
		t.Errorf("expected 3 requests, got %d", summary.RequestCount)
	}
	if summary.TotalTokens != 525 {
		t.Errorf("expected 525 total tokens, got %d", summary.TotalTokens)
	}
	if summary.TotalPromptTokens != 350 {
		t.Errorf("expected 350 prompt tokens, got %d", summary.TotalPromptTokens)
	}
	if summary.TotalCompletionTokens != 175 {
		t.Errorf("expected 175 completion tokens, got %d", summary.TotalCompletionTokens)
	}
}

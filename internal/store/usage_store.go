package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrviu/ai-gateway/internal/model"
)

// UsageStore 用量记录存储接口
type UsageStore interface {
	Create(record *model.UsageRecord) error
	BatchCreate(records []*model.UsageRecord) error
	Query(query *model.UsageQuery) ([]*model.UsageRecord, error)
	GetSummary(tenantID string, startAt, endAt time.Time) (*model.UsageSummary, error)
}

type usageStore struct {
	db *sql.DB
}

// NewUsageStore 创建 UsageStore 实例
func NewUsageStore(db *sql.DB) UsageStore {
	return &usageStore{db: db}
}

func (s *usageStore) Create(record *model.UsageRecord) error {
	record.ID = uuid.New().String()
	record.CreatedAt = time.Now().UTC()

	_, err := s.db.Exec(
		`INSERT INTO usage_records (id, tenant_id, key_id, model, prompt_tokens, completion_tokens, total_tokens, request_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.TenantID, record.KeyID, record.Model, record.PromptTokens, record.CompletionTokens, record.TotalTokens, record.RequestID, record.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create usage record: %w", err)
	}
	return nil
}

func (s *usageStore) BatchCreate(records []*model.UsageRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO usage_records (id, tenant_id, key_id, model, prompt_tokens, completion_tokens, total_tokens, request_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, record := range records {
		record.ID = uuid.New().String()
		record.CreatedAt = now
		_, err := stmt.Exec(record.ID, record.TenantID, record.KeyID, record.Model, record.PromptTokens, record.CompletionTokens, record.TotalTokens, record.RequestID, record.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("insert usage record: %w", err)
		}
	}

	return tx.Commit()
}

func (s *usageStore) Query(query *model.UsageQuery) ([]*model.UsageRecord, error) {
	q := `SELECT r.id, r.tenant_id, COALESCE(t.name, ''), r.key_id, r.model, r.prompt_tokens, r.completion_tokens, r.total_tokens, r.request_id, r.created_at FROM usage_records r LEFT JOIN tenants t ON r.tenant_id = t.id WHERE 1=1`
	args := []interface{}{}

	if query.TenantID != "" {
		q += " AND r.tenant_id = ?"
		args = append(args, query.TenantID)
	}
	if query.KeyID != "" {
		q += " AND r.key_id = ?"
		args = append(args, query.KeyID)
	}
	if query.Model != "" {
		q += " AND r.model = ?"
		args = append(args, query.Model)
	}
	if !query.StartAt.IsZero() {
		q += " AND r.created_at >= ?"
		args = append(args, query.StartAt.Format(time.RFC3339))
	}
	if !query.EndAt.IsZero() {
		q += " AND r.created_at <= ?"
		args = append(args, query.EndAt.Format(time.RFC3339))
	}

	q += " ORDER BY r.created_at DESC"

	if query.Limit > 0 {
		q += " LIMIT ?"
		args = append(args, query.Limit)
	}
	if query.Offset > 0 {
		q += " OFFSET ?"
		args = append(args, query.Offset)
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query usage records: %w", err)
	}
	defer rows.Close()

	var records []*model.UsageRecord
	for rows.Next() {
		var r model.UsageRecord
		var createdAt string
		if err := rows.Scan(&r.ID, &r.TenantID, &r.TenantName, &r.KeyID, &r.Model, &r.PromptTokens, &r.CompletionTokens, &r.TotalTokens, &r.RequestID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan usage record: %w", err)
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		records = append(records, &r)
	}
	return records, rows.Err()
}

func (s *usageStore) GetSummary(tenantID string, startAt, endAt time.Time) (*model.UsageSummary, error) {
	q := `SELECT COALESCE(SUM(prompt_tokens),0), COALESCE(SUM(completion_tokens),0), COALESCE(SUM(total_tokens),0), COUNT(*) FROM usage_records WHERE 1=1`
	args := []interface{}{}

	if tenantID != "" {
		q += " AND tenant_id = ?"
		args = append(args, tenantID)
	}
	if !startAt.IsZero() {
		q += " AND created_at >= ?"
		args = append(args, startAt.Format(time.RFC3339))
	}
	if !endAt.IsZero() {
		q += " AND created_at <= ?"
		args = append(args, endAt.Format(time.RFC3339))
	}

	var summary model.UsageSummary
	err := s.db.QueryRow(q, args...).Scan(&summary.TotalPromptTokens, &summary.TotalCompletionTokens, &summary.TotalTokens, &summary.RequestCount)
	if err != nil {
		return nil, fmt.Errorf("get usage summary: %w", err)
	}
	return &summary, nil
}

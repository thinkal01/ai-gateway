package usage

import (
	"sync"
	"testing"
	"time"

	"github.com/vrviu/ai-gateway/internal/model"
)

// mockUsageStore 模拟 UsageStore
type mockUsageStore struct {
	mu      sync.Mutex
	records []*model.UsageRecord
}

func (m *mockUsageStore) Create(record *model.UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, record)
	return nil
}

func (m *mockUsageStore) BatchCreate(records []*model.UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, records...)
	return nil
}

func (m *mockUsageStore) Query(query *model.UsageQuery) ([]*model.UsageRecord, error) {
	return nil, nil
}

func (m *mockUsageStore) GetSummary(tenantID string, startAt, endAt time.Time) (*model.UsageSummary, error) {
	return &model.UsageSummary{}, nil
}

func TestNewRecorder(t *testing.T) {
	mock := &mockUsageStore{}
	r := NewRecorder(mock, 100*time.Millisecond, 10)
	if r == nil {
		t.Fatal("expected recorder, got nil")
	}
	if r.flushInterval != 100*time.Millisecond {
		t.Errorf("expected flush interval 100ms, got %v", r.flushInterval)
	}
	if r.batchSize != 10 {
		t.Errorf("expected batch size 10, got %d", r.batchSize)
	}
	r.Stop()
}

func TestRecorder_Record_FlushByBatchSize(t *testing.T) {
	mock := &mockUsageStore{}
	r := NewRecorder(mock, 1*time.Hour, 3) // batch size = 3, long interval

	// 添加 3 条记录，触发 batch flush
	for i := 0; i < 3; i++ {
		r.Record(&model.UsageRecord{
			TenantID:         "tenant-1",
			KeyID:            "key-1",
			Model:            "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			RequestID:        "req-001",
		})
	}

	// 等待异步 flush 完成
	time.Sleep(50 * time.Millisecond)

	r.Stop()

	if len(mock.records) != 3 {
		t.Errorf("expected 3 records, got %d", len(mock.records))
	}
}

func TestRecorder_Stop_FlushRemaining(t *testing.T) {
	mock := &mockUsageStore{}
	r := NewRecorder(mock, 1*time.Hour, 100) // batch size > records, won't trigger batch flush

	// 添加 1 条记录
	r.Record(&model.UsageRecord{
		TenantID:         "tenant-1",
		KeyID:            "key-1",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		RequestID:        "req-001",
	})

	r.Stop()

	// Stop 会 flush 剩余数据
	if len(mock.records) != 1 {
		t.Errorf("expected 1 record after stop, got %d", len(mock.records))
	}
}

func TestRecorder_FlushByInterval(t *testing.T) {
	mock := &mockUsageStore{}
	r := NewRecorder(mock, 50*time.Millisecond, 100) // short interval

	r.Record(&model.UsageRecord{
		TenantID:         "tenant-1",
		KeyID:            "key-1",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		RequestID:        "req-001",
	})

	// 等待定时 flush
	time.Sleep(100 * time.Millisecond)

	r.Stop()

	if len(mock.records) != 1 {
		t.Errorf("expected 1 record after interval flush, got %d", len(mock.records))
	}
}

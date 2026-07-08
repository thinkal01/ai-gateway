package usage

import (
	"log"
	"sync"
	"time"

	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/store"
)

// Recorder 异步用量记录器
type Recorder struct {
	store         store.UsageStore
	buffer        []*model.UsageRecord
	mu            sync.Mutex
	flushInterval time.Duration
	batchSize     int
	stopCh        chan struct{}
}

// NewRecorder 创建异步用量记录器
func NewRecorder(s store.UsageStore, flushInterval time.Duration, batchSize int) *Recorder {
	r := &Recorder{
		store:         s,
		buffer:        make([]*model.UsageRecord, 0, batchSize),
		flushInterval: flushInterval,
		batchSize:     batchSize,
		stopCh:        make(chan struct{}),
	}

	go r.loop()
	return r
}

// Record 记录一条用量
func (r *Recorder) Record(record *model.UsageRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buffer = append(r.buffer, record)

	if len(r.buffer) >= r.batchSize {
		go r.flush()
	}
}

// Stop 停止记录器，等待缓冲数据写入
func (r *Recorder) Stop() {
	close(r.stopCh)
	r.flush()
}

func (r *Recorder) loop() {
	ticker := time.NewTicker(r.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.flush()
		case <-r.stopCh:
			return
		}
	}
}

func (r *Recorder) flush() {
	r.mu.Lock()
	if len(r.buffer) == 0 {
		r.mu.Unlock()
		return
	}

	batch := make([]*model.UsageRecord, len(r.buffer))
	copy(batch, r.buffer)
	r.buffer = r.buffer[:0]
	r.mu.Unlock()

	if err := r.store.BatchCreate(batch); err != nil {
		log.Printf("[Usage] batch create failed: %v", err)
		// 失败不回退，简化处理
		return
	}
	log.Printf("[Usage] flushed %d records", len(batch))
}

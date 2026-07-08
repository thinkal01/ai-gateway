package store

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// NewDB 初始化 SQLite 数据库并执行自动迁移
func NewDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite 不支持并发写入
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	log.Println("[DB] database initialized successfully")
	return db, nil
}

func migrate(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS tenants (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id),
			name TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			key_hash TEXT NOT NULL UNIQUE,
			scopes TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			expires_at TEXT NOT NULL,
			last_used_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix)`,
		`CREATE TABLE IF NOT EXISTS usage_records (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			key_id TEXT NOT NULL,
			model TEXT NOT NULL,
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			request_id TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_records_tenant_id ON usage_records(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_records_created_at ON usage_records(created_at)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec migration: %w\nSQL: %s", err, stmt)
		}
	}

	return nil
}

// Package store 提供 SQLite 持久化实现：
// 全部实体（名称、发表、模式、关系、规则、观点、裁决、清单）落库，
// 支持重启后从数据库恢复并继续归并。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB 封装 sql.DB 并提供迁移能力。
type DB struct {
	sql *sql.DB
}

// Open 打开（或创建）SQLite 数据库并执行迁移。
// path 为空时使用内存库（供测试）。
func Open(path string) (*DB, error) {
	if path == "" {
		path = ":memory:"
	}
	dsn := path
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create db dir: %w", err)
			}
		}
		// 启用 WAL 以获得崩溃恢复与并发读。
		dsn = path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	}
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	raw.SetMaxOpenConns(1) // modernc sqlite 单写者；串行化避免锁竞争
	db := &DB{sql: raw}
	if err := db.migrate(); err != nil {
		raw.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// Close 关闭数据库连接。
func (d *DB) Close() error { return d.sql.Close() }

// SQL 暴露底层句柄给各 store 子模块。
func (d *DB) SQL() *sql.DB { return d.sql }

// migrate 建表：全部实体表 + 索引 + 唯一约束。
func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS names (
			id TEXT PRIMARY KEY,
			scientific_name TEXT NOT NULL,
			genus TEXT NOT NULL,
			specific_epithet TEXT NOT NULL,
			authors TEXT NOT NULL DEFAULT '',
			year_range_start INTEGER,
			year_range_end INTEGER,
			orthographic_key TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_names_ortho ON names(orthographic_key, authors)`,
		`CREATE TABLE IF NOT EXISTS publications (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			authors TEXT NOT NULL,
			journal TEXT NOT NULL,
			year_range_start INTEGER,
			year_range_end INTEGER,
			fingerprint TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS specimens (
			id TEXT PRIMARY KEY,
			collector TEXT NOT NULL,
			number TEXT NOT NULL,
			institution TEXT NOT NULL,
			fingerprint TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS name_links (
			id TEXT PRIMARY KEY,
			name_id TEXT NOT NULL,
			publication_id TEXT NOT NULL,
			specimen_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(name_id, publication_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_links_name ON name_links(name_id)`,
		`CREATE TABLE IF NOT EXISTS relations (
			id TEXT PRIMARY KEY,
			from_name_id TEXT NOT NULL,
			to_name_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			basis TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			view_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_relations_pair ON relations(from_name_id, to_name_id)`,
		`CREATE TABLE IF NOT EXISTS rule_versions (
			id TEXT PRIMARY KEY,
			version TEXT NOT NULL UNIQUE,
			priority_rule INTEGER NOT NULL DEFAULT 1,
			legitimacy_req INTEGER NOT NULL DEFAULT 1,
			homonym_rule INTEGER NOT NULL DEFAULT 1,
			orthography INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS views (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			rule_version TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS rulings (
			id TEXT PRIMARY KEY,
			view_id TEXT NOT NULL,
			relation_id TEXT NOT NULL,
			decision TEXT NOT NULL,
			rationale TEXT NOT NULL DEFAULT '',
			ruled_by TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS checklists (
			id TEXT PRIMARY KEY,
			view_id TEXT NOT NULL,
			rule_version TEXT NOT NULL,
			fingerprint TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS checklist_items (
			checklist_id TEXT NOT NULL,
			name_id TEXT NOT NULL,
			scientific_name TEXT NOT NULL,
			role TEXT NOT NULL,
			accepted_name_id TEXT NOT NULL DEFAULT '',
			PRIMARY KEY(checklist_id, name_id)
		)`,
	}
	for _, s := range stmts {
		if _, err := d.sql.Exec(s); err != nil {
			return fmt.Errorf("exec %q: %w", s, err)
		}
	}
	return nil
}

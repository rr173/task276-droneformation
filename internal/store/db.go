// Package store 提供 SQLite 持久化：建表迁移与各领域实体的 CRUD。
// 使用纯 Go 驱动 modernc.org/sqlite（CGO 无关，离线可构建）。
package store

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// Store 封装数据库连接，所有业务实体的读写均通过其后缀方法完成。
// mu 覆盖 Begin→Commit 整个窗口，避免 SQLite 单写者被并发切开。
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// WithTx 在同一把锁内开启事务执行 fn；失败时回滚，成功才提交。
func (s *Store) WithTx(fn func(tx *sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Open 打开（必要时创建）SQLite 数据库并完成迁移。
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// 串行化写入，避免 WAL 下多连接并发写死锁。
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error { return s.db.Close() }

// DB 暴露底层连接，供需要在事务中组合多实体写操作的调用方使用。
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS formation_runs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			min_separation_m REAL NOT NULL,
			confidence_k REAL NOT NULL,
			rule_version INTEGER NOT NULL,
			sealed_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS aircraft (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			callsign TEXT NOT NULL,
			radius_m REAL NOT NULL,
			height_baseline_m REAL NOT NULL,
			status TEXT NOT NULL,
			last_seq INTEGER NOT NULL DEFAULT 0,
			last_intent_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS intent_segments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			aircraft_id TEXT NOT NULL,
			seq INTEGER NOT NULL,
			t_start INTEGER NOT NULL,
			t_end INTEGER NOT NULL,
			pos_x REAL, pos_y REAL, pos_z REAL,
			vel_x REAL, vel_y REAL, vel_z REAL,
			sig_x REAL, sig_y REAL, sig_z REAL,
			sig_rate_x REAL, sig_rate_y REAL, sig_rate_z REAL,
			ref_height_baseline REAL,
			status TEXT NOT NULL DEFAULT 'raw',
			created_at INTEGER NOT NULL,
			UNIQUE(run_id, aircraft_id, seq)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_intent_run_aircraft ON intent_segments(run_id, aircraft_id, seq)`,
		`CREATE TABLE IF NOT EXISTS aircraft_covariance (
			run_id TEXT NOT NULL,
			aircraft_id TEXT NOT NULL,
			sig_x REAL, sig_y REAL, sig_z REAL,
			sig_rate_x REAL, sig_rate_y REAL, sig_rate_z REAL,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(run_id, aircraft_id)
		)`,
		`CREATE TABLE IF NOT EXISTS avoidance_relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			snapshot_id INTEGER NOT NULL DEFAULT 0,
			aircraft_a TEXT NOT NULL,
			aircraft_b TEXT NOT NULL,
			status TEXT NOT NULL,
			min_eff_distance REAL NOT NULL,
			required_gap REAL NOT NULL,
			worst_t INTEGER NOT NULL,
			verified_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rel_run_snap ON avoidance_relations(run_id, snapshot_id)`,
		`CREATE TABLE IF NOT EXISTS safety_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			rule_version INTEGER NOT NULL,
			status TEXT NOT NULL,
			snapshot_status TEXT NOT NULL,
			conflict_count INTEGER NOT NULL DEFAULT 0,
			safe_count INTEGER NOT NULL DEFAULT 0,
			frozen_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return err
		}
	}
	return nil
}

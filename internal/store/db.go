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

// WithTx 在同一把锁内开启事务、执行 fn 并提交；失败时回滚。
// 锁覆盖 Begin→fn→Commit 整个生命周期：SQLite 是单写者模型，即便在 WAL
// 下也只允许一个写事务。若在 Begin 后立即放锁，并发上报会在 fn 的写操作上
// 互相踩中对方的活跃写事务，触发 SQLITE_BUSY。全程持锁可串行化写事务，
// 保证每条意图都能落库而不会互相中断。
func (s *Store) WithTx(fn func(tx *sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Open 打开（必要时创建）SQLite 数据库并完成迁移。
func Open(path string) (*Store, error) {
	// busy_timeout 通过 DSN 的 _pragma 下发：sql.DB 的连接池会按需新建连接，
	// 每个 fresh 连接都会重放该 pragma，确保繁忙时阻塞等待而非立即报
	// SQLITE_BUSY。journal_mode/foreign_keys 同理，随连接各自生效。
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// 串行化写者由 WithTx 的锁保证；连接数过多只会徒增对同一文件的争抢，
	// 反而放大 SQLITE_BUSY 的概率。单写连接即可，读并发由连接池承接。
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
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

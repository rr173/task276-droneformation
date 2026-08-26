package store

import (
	"database/sql"

	"task276-droneformation/internal/model"
)

// CreateSnapshot 插入一条安全快照并返回自增主键。
func (s *Store) CreateSnapshot(snap *model.SafetySnapshot) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO safety_snapshots
		 (run_id,rule_version,status,snapshot_status,conflict_count,safe_count,frozen_at,created_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		snap.RunID, snap.RuleVersion, snap.Status, snap.SnapshotStatus, snap.ConflictCount, snap.SafeCount, snap.FrozenAt, snap.CreatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) GetSnapshot(id int64) (*model.SafetySnapshot, error) {
	snap := &model.SafetySnapshot{}
	err := s.db.QueryRow(
		`SELECT id,run_id,rule_version,status,snapshot_status,conflict_count,safe_count,frozen_at,created_at
		 FROM safety_snapshots WHERE id=?`, id).
		Scan(&snap.ID, &snap.RunID, &snap.RuleVersion, &snap.Status, &snap.SnapshotStatus,
			&snap.ConflictCount, &snap.SafeCount, &snap.FrozenAt, &snap.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrRunNotFound
	}
	if err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Store) ListSnapshots(runID string) ([]model.SafetySnapshot, error) {
	rows, err := s.db.Query(
		`SELECT id,run_id,rule_version,status,snapshot_status,conflict_count,safe_count,frozen_at,created_at
		 FROM safety_snapshots WHERE run_id=? ORDER BY created_at DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SafetySnapshot
	for rows.Next() {
		snap := model.SafetySnapshot{}
		if err := rows.Scan(&snap.ID, &snap.RunID, &snap.RuleVersion, &snap.Status, &snap.SnapshotStatus,
			&snap.ConflictCount, &snap.SafeCount, &snap.FrozenAt, &snap.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

func (s *Store) SetSnapshotStatus(id int64, snapStatus, status string, frozenAt int64) error {
	_, err := s.db.Exec(
		`UPDATE safety_snapshots SET snapshot_status=?, status=?, frozen_at=? WHERE id=?`,
		snapStatus, status, frozenAt, id)
	return err
}

// SupersedeSnapshotsExcept 将同一运行下除 keepID 之外的已发布快照置为 superseded。
func (s *Store) SupersedeSnapshotsExcept(runID string, keepID int64) error {
	_, err := s.db.Exec(
		`UPDATE safety_snapshots SET snapshot_status=? WHERE run_id=? AND id<>? AND snapshot_status=?`,
		model.SnapSuperseded, runID, keepID, model.SnapPublished)
	return err
}

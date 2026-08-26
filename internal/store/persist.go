package store

import (
	"database/sql"
	"fmt"

	"task276-droneformation/internal/model"
)

// IntentBundle 是一次意图入库要一起落盘的三件套：意图段、协方差、飞行器游标。
type IntentBundle struct {
	Intent     *model.IntentSegment
	Covariance *model.Covariance
	LastSeq    int64
	LastAt     int64
}

// PersistIntentBundle 原子写入意图段、定位协方差与飞行器最近序号。
func (s *Store) PersistIntentBundle(b IntentBundle) error {
	if b.Intent == nil || b.Covariance == nil {
		return fmt.Errorf("intent bundle incomplete")
	}
	return s.WithTx(func(tx *sql.Tx) error {
		res, err := tx.Exec(
			`INSERT INTO intent_segments
			 (run_id,aircraft_id,seq,t_start,t_end,pos_x,pos_y,pos_z,vel_x,vel_y,vel_z,
			  sig_x,sig_y,sig_z,sig_rate_x,sig_rate_y,sig_rate_z,ref_height_baseline,status,created_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			b.Intent.RunID, b.Intent.AircraftID, b.Intent.Seq, b.Intent.TStart, b.Intent.TEnd,
			b.Intent.PosX, b.Intent.PosY, b.Intent.PosZ, b.Intent.VelX, b.Intent.VelY, b.Intent.VelZ,
			b.Intent.SigX, b.Intent.SigY, b.Intent.SigZ, b.Intent.SigRateX, b.Intent.SigRateY, b.Intent.SigRateZ,
			b.Intent.RefHeightBaseline, b.Intent.Status, b.Intent.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert intent: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("intent id: %w", err)
		}
		b.Intent.ID = id
		if _, err := tx.Exec(
			`INSERT INTO aircraft_covariance (run_id,aircraft_id,sig_x,sig_y,sig_z,sig_rate_x,sig_rate_y,sig_rate_z,updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?)
			 ON CONFLICT(run_id,aircraft_id) DO UPDATE SET
			   sig_x=excluded.sig_x, sig_y=excluded.sig_y, sig_z=excluded.sig_z,
			   sig_rate_x=excluded.sig_rate_x, sig_rate_y=excluded.sig_rate_y, sig_rate_z=excluded.sig_rate_z,
			   updated_at=excluded.updated_at`,
			b.Covariance.RunID, b.Covariance.AircraftID, b.Covariance.SigX, b.Covariance.SigY, b.Covariance.SigZ,
			b.Covariance.SigRateX, b.Covariance.SigRateY, b.Covariance.SigRateZ, b.Covariance.UpdatedAt); err != nil {
			return fmt.Errorf("upsert covariance: %w", err)
		}
		if _, err := tx.Exec(`UPDATE aircraft SET last_seq=?, last_intent_at=? WHERE id=?`,
			b.LastSeq, b.LastAt, b.Intent.AircraftID); err != nil {
			return fmt.Errorf("update last intent: %w", err)
		}
		return nil
	})
}

// IntentStatusPatch 描述一次验证中对某架飞行器意图状态的批量改写。
type IntentStatusPatch struct {
	AircraftID string
	BulkStatus string
	ValidID    int64
}

// VerificationWrite 是一次验证结果的原子落盘单元。
type VerificationWrite struct {
	RunID     string
	RunStatus string
	Snapshot  *model.SafetySnapshot
	Relations []model.AvoidanceRelation
	Patches   []IntentStatusPatch
}

// PersistVerification 在同一事务中写入意图状态、草稿快照、避碰关系与运行状态。
func (s *Store) PersistVerification(w VerificationWrite) (int64, error) {
	if w.Snapshot == nil {
		return 0, fmt.Errorf("verification snapshot missing")
	}
	var snapID int64
	err := s.WithTx(func(tx *sql.Tx) error {
		for _, p := range w.Patches {
			if _, err := tx.Exec(
				`UPDATE intent_segments SET status=? WHERE run_id=? AND aircraft_id=?`,
				p.BulkStatus, w.RunID, p.AircraftID); err != nil {
				return fmt.Errorf("bulk intent status: %w", err)
			}
			if p.ValidID > 0 {
				if _, err := tx.Exec(`UPDATE intent_segments SET status=? WHERE id=?`,
					model.IntentValid, p.ValidID); err != nil {
					return fmt.Errorf("valid intent status: %w", err)
				}
			}
		}
		res, err := tx.Exec(
			`INSERT INTO safety_snapshots
			 (run_id,rule_version,status,snapshot_status,conflict_count,safe_count,frozen_at,created_at)
			 VALUES (?,?,?,?,?,?,?,?)`,
			w.Snapshot.RunID, w.Snapshot.RuleVersion, w.Snapshot.Status, w.Snapshot.SnapshotStatus,
			w.Snapshot.ConflictCount, w.Snapshot.SafeCount, w.Snapshot.FrozenAt, w.Snapshot.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert snapshot: %w", err)
		}
		snapID, err = res.LastInsertId()
		if err != nil {
			return fmt.Errorf("snapshot id: %w", err)
		}
		for i := range w.Relations {
			w.Relations[i].SnapshotID = snapID
			r := w.Relations[i]
			relRes, err := tx.Exec(
				`INSERT INTO avoidance_relations
				 (run_id,snapshot_id,aircraft_a,aircraft_b,status,min_eff_distance,required_gap,worst_t,verified_at)
				 VALUES (?,?,?,?,?,?,?,?,?)`,
				r.RunID, r.SnapshotID, r.AircraftA, r.AircraftB, r.Status, r.MinEffDistance, r.RequiredGap, r.WorstT, r.VerifiedAt)
			if err != nil {
				return fmt.Errorf("insert relation: %w", err)
			}
			rid, err := relRes.LastInsertId()
			if err != nil {
				return fmt.Errorf("relation id: %w", err)
			}
			w.Relations[i].ID = rid
		}
		if _, err := tx.Exec(`UPDATE formation_runs SET status=?, sealed_at=? WHERE id=?`,
			w.RunStatus, int64(0), w.RunID); err != nil {
			return fmt.Errorf("set run status: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return snapID, nil
}

// PersistPublish 原子发布草稿快照、替代旧发布件并封存运行。
func (s *Store) PersistPublish(runID string, snapID int64, snapStatus string, frozenAt int64) error {
	return s.WithTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(
			`UPDATE safety_snapshots SET snapshot_status=? WHERE run_id=? AND id<>? AND snapshot_status=?`,
			model.SnapSuperseded, runID, snapID, model.SnapPublished); err != nil {
			return fmt.Errorf("supersede snapshots: %w", err)
		}
		if _, err := tx.Exec(
			`UPDATE safety_snapshots SET snapshot_status=?, status=?, frozen_at=? WHERE id=?`,
			model.SnapPublished, snapStatus, frozenAt, snapID); err != nil {
			return fmt.Errorf("publish snapshot: %w", err)
		}
		if _, err := tx.Exec(`UPDATE formation_runs SET status=?, sealed_at=? WHERE id=?`,
			model.RunSealed, frozenAt, runID); err != nil {
			return fmt.Errorf("seal run: %w", err)
		}
		return nil
	})
}

// PersistReinstate 将隔离飞行器恢复为活跃，但不改写历史意图状态。
func (s *Store) PersistReinstate(aircraftID string) error {
	return s.WithTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE aircraft SET status=? WHERE id=?`,
			model.AircraftActive, aircraftID); err != nil {
			return fmt.Errorf("reinstate aircraft: %w", err)
		}
		return nil
	})
}

// LatestPublishedSnapshot 返回某运行当前仍处于 published 的快照；没有则返回 nil。
func (s *Store) LatestPublishedSnapshot(runID string) (*model.SafetySnapshot, error) {
	snaps, err := s.ListSnapshots(runID)
	if err != nil {
		return nil, err
	}
	for i := range snaps {
		if snaps[i].SnapshotStatus == model.SnapPublished {
			cp := snaps[i]
			return &cp, nil
		}
	}
	return nil, nil
}

// PersistIsolation 原子隔离飞行器并排除其全部意图。
func (s *Store) PersistIsolation(runID, aircraftID string) error {
	return s.WithTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE aircraft SET status=? WHERE id=?`,
			model.AircraftIsolated, aircraftID); err != nil {
			return fmt.Errorf("isolate aircraft: %w", err)
		}
		if _, err := tx.Exec(`UPDATE intent_segments SET status=? WHERE run_id=? AND aircraft_id=?`,
			model.IntentExcluded, runID, aircraftID); err != nil {
			return fmt.Errorf("exclude intents: %w", err)
		}
		return nil
	})
}

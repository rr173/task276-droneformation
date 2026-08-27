package store

import (
	"database/sql"

	"task276-droneformation/internal/model"
)

func (s *Store) CreateAircraft(a *model.Aircraft) error {
	_, err := s.db.Exec(
		`INSERT INTO aircraft (id,run_id,callsign,radius_m,height_baseline_m,status,last_seq,last_intent_at,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		a.ID, a.RunID, a.Callsign, a.RadiusM, a.HeightBaselineM, a.Status, a.LastSeq, a.LastIntentAt, a.CreatedAt)
	return err
}

func (s *Store) GetAircraft(id string) (*model.Aircraft, error) {
	a := &model.Aircraft{}
	err := s.db.QueryRow(
		`SELECT id,run_id,callsign,radius_m,height_baseline_m,status,last_seq,last_intent_at,created_at
		 FROM aircraft WHERE id=?`, id).
		Scan(&a.ID, &a.RunID, &a.Callsign, &a.RadiusM, &a.HeightBaselineM, &a.Status, &a.LastSeq, &a.LastIntentAt, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrAircraftNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Store) ListAircraft(runID string) ([]model.Aircraft, error) {
	rows, err := s.db.Query(
		`SELECT id,run_id,callsign,radius_m,height_baseline_m,status,last_seq,last_intent_at,created_at
		 FROM aircraft WHERE run_id=? ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// 每次调用都返回独立分配的切片，避免共享底层数组被后续调用改写，
	// 否则"先列举飞行器、再做避碰验证"会让列表丢机位。
	acs := make([]model.Aircraft, 0)
	for rows.Next() {
		a := model.Aircraft{}
		if err := rows.Scan(&a.ID, &a.RunID, &a.Callsign, &a.RadiusM, &a.HeightBaselineM, &a.Status, &a.LastSeq, &a.LastIntentAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		acs = append(acs, a)
	}
	return acs, rows.Err()
}

func (s *Store) UpdateAircraftStatus(id, status string) error {
	_, err := s.db.Exec(`UPDATE aircraft SET status=? WHERE id=?`, status, id)
	return err
}

func (s *Store) UpdateAircraftLastIntent(id string, seq, at int64) error {
	_, err := s.db.Exec(`UPDATE aircraft SET last_seq=?, last_intent_at=? WHERE id=?`, seq, at, id)
	return err
}

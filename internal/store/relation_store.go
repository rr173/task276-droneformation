package store

import (
	"database/sql"

	"task276-droneformation/internal/model"
)

func (s *Store) InsertRelation(r *model.AvoidanceRelation) error {
	res, err := s.db.Exec(
		`INSERT INTO avoidance_relations
		 (run_id,snapshot_id,aircraft_a,aircraft_b,status,min_eff_distance,required_gap,worst_t,verified_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		r.RunID, r.SnapshotID, r.AircraftA, r.AircraftB, r.Status, r.MinEffDistance, r.RequiredGap, r.WorstT, r.VerifiedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		r.ID = id
	}
	return nil
}

func (s *Store) ListRelations(runID string, snapshotID int64) ([]model.AvoidanceRelation, error) {
	rows, err := s.db.Query(
		`SELECT id,run_id,snapshot_id,aircraft_a,aircraft_b,status,min_eff_distance,required_gap,worst_t,verified_at
		 FROM avoidance_relations WHERE run_id=? AND snapshot_id=? ORDER BY id ASC`, runID, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AvoidanceRelation
	for rows.Next() {
		r := model.AvoidanceRelation{}
		if err := rows.Scan(&r.ID, &r.RunID, &r.SnapshotID, &r.AircraftA, &r.AircraftB, &r.Status,
			&r.MinEffDistance, &r.RequiredGap, &r.WorstT, &r.VerifiedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetRelation(id int64) (*model.AvoidanceRelation, error) {
	r := &model.AvoidanceRelation{}
	err := s.db.QueryRow(
		`SELECT id,run_id,snapshot_id,aircraft_a,aircraft_b,status,min_eff_distance,required_gap,worst_t,verified_at
		 FROM avoidance_relations WHERE id=?`, id).
		Scan(&r.ID, &r.RunID, &r.SnapshotID, &r.AircraftA, &r.AircraftB, &r.Status,
			&r.MinEffDistance, &r.RequiredGap, &r.WorstT, &r.VerifiedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrAircraftNotFound
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

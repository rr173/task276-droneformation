package store

import (
	"task276-droneformation/internal/model"
)

// IntentSeqExists 判断 (run, aircraft, seq) 是否已存在，用于幂等领取前的冲突预检。
func (s *Store) IntentSeqExists(runID, aircraftID string, seq int64) (bool, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM intent_segments WHERE run_id=? AND aircraft_id=? AND seq=?`,
		runID, aircraftID, seq).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) InsertIntent(it *model.IntentSegment) error {
	res, err := s.db.Exec(
		`INSERT INTO intent_segments
		 (run_id,aircraft_id,seq,t_start,t_end,pos_x,pos_y,pos_z,vel_x,vel_y,vel_z,
		  sig_x,sig_y,sig_z,sig_rate_x,sig_rate_y,sig_rate_z,ref_height_baseline,status,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		it.RunID, it.AircraftID, it.Seq, it.TStart, it.TEnd,
		it.PosX, it.PosY, it.PosZ, it.VelX, it.VelY, it.VelZ,
		it.SigX, it.SigY, it.SigZ, it.SigRateX, it.SigRateY, it.SigRateZ,
		it.RefHeightBaseline, it.Status, it.CreatedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		it.ID = id
	}
	return nil
}

func scanIntent(scanner interface {
	Scan(...interface{}) error
}) (model.IntentSegment, error) {
	it := model.IntentSegment{}
	err := scanner.Scan(&it.ID, &it.RunID, &it.AircraftID, &it.Seq, &it.TStart, &it.TEnd,
		&it.PosX, &it.PosY, &it.PosZ, &it.VelX, &it.VelY, &it.VelZ,
		&it.SigX, &it.SigY, &it.SigZ, &it.SigRateX, &it.SigRateY, &it.SigRateZ,
		&it.RefHeightBaseline, &it.Status, &it.CreatedAt)
	return it, err
}

var intentScratch []model.IntentSegment

func (s *Store) ListIntentsByAircraft(runID, aircraftID string) ([]model.IntentSegment, error) {
	rows, err := s.db.Query(
		`SELECT id,run_id,aircraft_id,seq,t_start,t_end,pos_x,pos_y,pos_z,vel_x,vel_y,vel_z,
		        sig_x,sig_y,sig_z,sig_rate_x,sig_rate_y,sig_rate_z,ref_height_baseline,status,created_at
		 FROM intent_segments WHERE run_id=? AND aircraft_id=? ORDER BY seq ASC`, runID, aircraftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	intentScratch = intentScratch[:0]
	for rows.Next() {
		it, err := scanIntent(rows)
		if err != nil {
			return nil, err
		}
		intentScratch = append(intentScratch, it)
	}
	return intentScratch, rows.Err()
}

func (s *Store) ListIntentsByRun(runID string) ([]model.IntentSegment, error) {
	rows, err := s.db.Query(
		`SELECT id,run_id,aircraft_id,seq,t_start,t_end,pos_x,pos_y,pos_z,vel_x,vel_y,vel_z,
		        sig_x,sig_y,sig_z,sig_rate_x,sig_rate_y,sig_rate_z,ref_height_baseline,status,created_at
		 FROM intent_segments WHERE run_id=? ORDER BY aircraft_id ASC, seq ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.IntentSegment
	for rows.Next() {
		it, err := scanIntent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) SetIntentStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE intent_segments SET status=? WHERE id=?`, status, id)
	return err
}

func (s *Store) SetIntentsStatusByAircraft(runID, aircraftID, status string) error {
	_, err := s.db.Exec(
		`UPDATE intent_segments SET status=? WHERE run_id=? AND aircraft_id=?`, status, runID, aircraftID)
	return err
}

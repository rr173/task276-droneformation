package store

import (
	"database/sql"

	"task276-droneformation/internal/model"
)

// UpsertCovariance 写入或覆盖某飞行器在编队中的定位不确定度。
func (s *Store) UpsertCovariance(c *model.Covariance) error {
	_, err := s.db.Exec(
		`INSERT INTO aircraft_covariance (run_id,aircraft_id,sig_x,sig_y,sig_z,sig_rate_x,sig_rate_y,sig_rate_z,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(run_id,aircraft_id) DO UPDATE SET
		   sig_x=excluded.sig_x, sig_y=excluded.sig_y, sig_z=excluded.sig_z,
		   sig_rate_x=excluded.sig_rate_x, sig_rate_y=excluded.sig_rate_y, sig_rate_z=excluded.sig_rate_z,
		   updated_at=excluded.updated_at`,
		c.RunID, c.AircraftID, c.SigX, c.SigY, c.SigZ, c.SigRateX, c.SigRateY, c.SigRateZ, c.UpdatedAt)
	return err
}

func (s *Store) GetCovariance(runID, aircraftID string) (*model.Covariance, error) {
	c := &model.Covariance{}
	err := s.db.QueryRow(
		`SELECT run_id,aircraft_id,sig_x,sig_y,sig_z,sig_rate_x,sig_rate_y,sig_rate_z,updated_at
		 FROM aircraft_covariance WHERE run_id=? AND aircraft_id=?`, runID, aircraftID).
		Scan(&c.RunID, &c.AircraftID, &c.SigX, &c.SigY, &c.SigZ, &c.SigRateX, &c.SigRateY, &c.SigRateZ, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

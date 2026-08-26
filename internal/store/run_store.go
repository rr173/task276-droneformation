package store

import (
	"database/sql"

	"task276-droneformation/internal/model"
)

func (s *Store) CreateRun(r *model.FormationRun) error {
	_, err := s.db.Exec(
		`INSERT INTO formation_runs (id,name,status,min_separation_m,confidence_k,rule_version,sealed_at,created_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		r.ID, r.Name, r.Status, r.MinSeparationM, r.ConfidenceK, r.RuleVersion, r.SealedAt, r.CreatedAt)
	return err
}

func (s *Store) GetRun(id string) (*model.FormationRun, error) {
	r := &model.FormationRun{}
	err := s.db.QueryRow(
		`SELECT id,name,status,min_separation_m,confidence_k,rule_version,sealed_at,created_at
		 FROM formation_runs WHERE id=?`, id).
		Scan(&r.ID, &r.Name, &r.Status, &r.MinSeparationM, &r.ConfidenceK, &r.RuleVersion, &r.SealedAt, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrRunNotFound
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) ListRuns() ([]model.FormationRun, error) {
	rows, err := s.db.Query(
		`SELECT id,name,status,min_separation_m,confidence_k,rule_version,sealed_at,created_at
		 FROM formation_runs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	var out []model.FormationRun
	for rows.Next() {
		r := model.FormationRun{}
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.MinSeparationM, &r.ConfidenceK, &r.RuleVersion, &r.SealedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateRunConfig 修改间隔与置信系数，并使规则版本自增（用于快照可追溯）。
func (s *Store) UpdateRunConfig(id string, minSep, k float64) error {
	_, err := s.db.Exec(
		`UPDATE formation_runs SET min_separation_m=?, confidence_k=?, rule_version=rule_version+1 WHERE id=?`,
		minSep, k, id)
	return err
}

func (s *Store) SetRunStatus(id, status string, sealedAt int64) error {
	_, err := s.db.Exec(`UPDATE formation_runs SET status=?, sealed_at=? WHERE id=?`, status, sealedAt, id)
	return err
}

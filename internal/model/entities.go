package model

// FormationRun 是一次编队避碰验证的运行上下文。
type FormationRun struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	MinSeparationM float64 `json:"min_separation_m"`
	ConfidenceK    float64 `json:"confidence_k"`
	RuleVersion    int64   `json:"rule_version"`
	SealedAt       int64   `json:"sealed_at"`
	CreatedAt      int64   `json:"created_at"`
}

// Aircraft 是编队中的一架飞行器。
type Aircraft struct {
	ID              string  `json:"id"`
	RunID           string  `json:"run_id"`
	Callsign        string  `json:"callsign"`
	RadiusM         float64 `json:"radius_m"`
	HeightBaselineM float64 `json:"height_baseline_m"`
	Status          string  `json:"status"`
	LastSeq         int64   `json:"last_seq"`
	LastIntentAt    int64   `json:"last_intent_at"`
	CreatedAt       int64   `json:"created_at"`
}

// IntentSegment 是一架飞行器发布的未来轨迹意图段。
// 位置在 TStart 时刻给出，其后按定常速度外推；定位不确定度随时间线性增长。
type IntentSegment struct {
	ID                int64   `json:"id"`
	RunID             string  `json:"run_id"`
	AircraftID        string  `json:"aircraft_id"`
	Seq               int64   `json:"seq"`
	TStart            int64   `json:"t_start"`
	TEnd              int64   `json:"t_end"`
	PosX              float64 `json:"pos_x"`
	PosY              float64 `json:"pos_y"`
	PosZ              float64 `json:"pos_z"`
	VelX              float64 `json:"vel_x"`
	VelY              float64 `json:"vel_y"`
	VelZ              float64 `json:"vel_z"`
	SigX              float64 `json:"sig_x"`
	SigY              float64 `json:"sig_y"`
	SigZ              float64 `json:"sig_z"`
	SigRateX          float64 `json:"sig_rate_x"`
	SigRateY          float64 `json:"sig_rate_y"`
	SigRateZ          float64 `json:"sig_rate_z"`
	RefHeightBaseline float64 `json:"ref_height_baseline"`
	Status            string  `json:"status"`
	CreatedAt         int64   `json:"created_at"`
}

// AvoidanceRelation 是两架飞行器在一次验证中的避碰关系。
type AvoidanceRelation struct {
	ID             int64   `json:"id"`
	RunID          string  `json:"run_id"`
	SnapshotID     int64   `json:"snapshot_id"`
	AircraftA      string  `json:"aircraft_a"`
	AircraftB      string  `json:"aircraft_b"`
	Status         string  `json:"status"`
	MinEffDistance float64 `json:"min_eff_distance"`
	RequiredGap    float64 `json:"required_gap"`
	WorstT         int64   `json:"worst_t"`
	VerifiedAt     int64   `json:"verified_at"`
}

// SafetySnapshot 是一次验证结果被冻结后的不可变安全快照。
type SafetySnapshot struct {
	ID             int64  `json:"id"`
	RunID          string `json:"run_id"`
	RuleVersion    int64  `json:"rule_version"`
	Status         string `json:"status"`          // safe / conflict
	SnapshotStatus string `json:"snapshot_status"` // draft / published / superseded
	ConflictCount  int    `json:"conflict_count"`
	SafeCount      int    `json:"safe_count"`
	FrozenAt       int64  `json:"frozen_at"`
	CreatedAt      int64  `json:"created_at"`
}

// Covariance 是某飞行器在编队中的定位不确定度（基准标准差与线性增长率）。
type Covariance struct {
	RunID                            string  `json:"run_id"`
	AircraftID                       string  `json:"aircraft_id"`
	SigX              float64 `json:"sig_x"`
	SigY              float64 `json:"sig_y"`
	SigZ              float64 `json:"sig_z"`
	SigRateX          float64 `json:"sig_rate_x"`
	SigRateY          float64 `json:"sig_rate_y"`
	SigRateZ          float64 `json:"sig_rate_z"`
	UpdatedAt                        int64   `json:"updated_at"`
}

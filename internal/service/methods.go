package service

import (
	"context"
	"fmt"
	"math"

	"task276-droneformation/internal/conflict"
	"task276-droneformation/internal/intent"
	"task276-droneformation/internal/model"
	"task276-droneformation/internal/snapshot"
	"task276-droneformation/internal/state"
	"task276-droneformation/internal/store"
)

// guardCtx 在落盘前检查调用方上下文是否已取消，取消后不再继续写入。
func guardCtx(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// CreateRun 新建一次编队验证运行。
func (a *App) CreateRun(ctx context.Context, name string) (*model.FormationRun, error) {
	if err := guardCtx(ctx); err != nil {
		return nil, err
	}
	if name == "" {
		name = fmt.Sprintf("run-%d", a.now())
	}
	r := &model.FormationRun{
		ID:             uniqueID("run"),
		Name:           name,
		Status:         model.RunReceiving,
		MinSeparationM: model.DefaultMinSepM,
		ConfidenceK:    model.DefaultConfidenceK,
		RuleVersion:    1,
		SealedAt:       0,
		CreatedAt:      a.now(),
	}
	if err := a.store.CreateRun(r); err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}
	return r, nil
}

// RegisterAircraft 在运行中注册一架飞行器。
func (a *App) RegisterAircraft(ctx context.Context, runID, callsign string, radiusM, heightBaselineM float64) (*model.Aircraft, error) {
	if err := guardCtx(ctx); err != nil {
		return nil, err
	}
	if _, err := a.store.GetRun(runID); err != nil {
		return nil, fmt.Errorf("register aircraft: %w", err)
	}
	if radiusM <= 0 {
		radiusM = model.DefaultRadiusM
	}
	if heightBaselineM < 0 {
		heightBaselineM = 0
	}
	ac := &model.Aircraft{
		ID:              uniqueID("ac"),
		RunID:           runID,
		Callsign:        callsign,
		RadiusM:         radiusM,
		HeightBaselineM: heightBaselineM,
		Status:         model.AircraftActive,
		LastSeq:         0,
		LastIntentAt:    0,
		CreatedAt:       a.now(),
	}
	if err := a.store.CreateAircraft(ac); err != nil {
		return nil, fmt.Errorf("create aircraft: %w", err)
	}
	return ac, nil
}

// IngestIntent 接收并持久化一条飞行器意图段（按 seq 幂等）。
func (a *App) IngestIntent(ctx context.Context, runID, aircraftID string, in IntentInput) (*model.IntentSegment, error) {
	if err := guardCtx(ctx); err != nil {
		return nil, err
	}
	run, err := a.store.GetRun(runID)
	if err != nil {
		return nil, fmt.Errorf("ingest get run: %w", err)
	}
	if state.IsSealed(*run) {
		return nil, model.ErrRunSealed
	}
	ac, err := a.store.GetAircraft(aircraftID)
	if err != nil {
		return nil, fmt.Errorf("ingest get aircraft: %w", err)
	}
	if ac.RunID != runID {
		return nil, model.ErrAircraftNotFound
	}
	if in.TEnd <= in.TStart {
		return nil, model.ErrIntentInvalid
	}
	if in.TStart < 0 || in.TEnd < 0 {
		return nil, model.ErrIntentInvalid
	}
	if math.IsNaN(in.PosX) || math.IsNaN(in.PosY) || math.IsNaN(in.PosZ) {
		return nil, model.ErrIntentInvalid
	}
	if in.SigX < 0 || in.SigY < 0 || in.SigZ < 0 {
		return nil, model.ErrCovarianceIllegal
	}
	// 高度基准一致性：同一飞行器不同意图的高度基准不应出现跳变。
	if in.RefHeightBaseline != 0 {
		if segs, err := a.store.ListIntentsByAircraft(runID, aircraftID); err == nil && len(segs) > 0 {
			for _, s := range segs {
				if s.RefHeightBaseline != 0 && math.Abs(s.RefHeightBaseline-in.RefHeightBaseline) > 1e-6 {
					return nil, model.ErrHeightMismatch
				}
			}
		}
	}
	exists, err := a.store.IntentSeqExists(runID, aircraftID, in.Seq)
	if err != nil {
		return nil, fmt.Errorf("check intent seq: %w", err)
	}
	if exists {
		return nil, model.ErrDuplicateSeq
	}
	it := &model.IntentSegment{
		RunID:             runID,
		AircraftID:        aircraftID,
		Seq:               in.Seq,
		TStart:            in.TStart,
		TEnd:              in.TEnd,
		PosX:              in.PosX, PosY: in.PosY, PosZ: in.PosZ,
		VelX: in.VelX, VelY: in.VelY, VelZ: in.VelZ,
		SigX: in.SigX, SigY: in.SigY, SigZ: in.SigZ,
		SigRateX: in.SigRateX, SigRateY: in.SigRateY, SigRateZ: in.SigRateZ,
		RefHeightBaseline: in.RefHeightBaseline,
		Status:            model.IntentRaw,
		CreatedAt:         a.now(),
	}
	if err := guardCtx(ctx); err != nil {
		return nil, err
	}
	c := &model.Covariance{
		RunID: runID, AircraftID: aircraftID,
		SigX: in.SigX, SigY: in.SigY, SigZ: in.SigZ,
		SigRateX: in.SigRateX, SigRateY: in.SigRateY, SigRateZ: in.SigRateZ,
		UpdatedAt: a.now(),
	}
	if err := a.store.PersistIntentBundle(store.IntentBundle{
		Intent: it, Covariance: c, LastSeq: in.Seq, LastAt: a.now(),
	}); err != nil {
		return nil, fmt.Errorf("persist intent bundle: %w", err)
	}
	return it, nil
}

// BatchIngestIntent 批量接收意图段，遇到第一个错误即返回；取消后停止后续写入。
func (a *App) BatchIngestIntent(ctx context.Context, runID string, items map[string]IntentInput) error {
	for aircraftID, in := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := a.IngestIntent(ctx, runID, aircraftID, in); err != nil {
			return err
		}
	}
	return nil
}

// UpdateConfig 调整编队最小间隔与置信系数（规则版本自增）。
func (a *App) UpdateConfig(ctx context.Context, runID string, minSep, k float64) error {
	if err := guardCtx(ctx); err != nil {
		return err
	}
	run, err := a.store.GetRun(runID)
	if err != nil {
		return fmt.Errorf("update config: %w", err)
	}
	if state.IsSealed(*run) {
		return model.ErrRunSealed
	}
	if minSep < 0 || k <= 0 {
		return model.ErrIntentInvalid
	}
	return a.store.UpdateRunConfig(runID, minSep, k)
}

// VerificationResult 是 VerifyRun 的返回。
type VerificationResult struct {
	RunID         string
	SnapshotID    int64
	Status        string
	ConflictCount int
	SafeCount     int
	Relations     []model.AvoidanceRelation
}

func (a *App) createSnapshotAndRels(ctx context.Context, run *model.FormationRun, rels []model.AvoidanceRelation, patches []store.IntentStatusPatch) (int64, string, error) {
	if err := guardCtx(ctx); err != nil {
		return 0, "", err
	}
	sum := snapshot.BuildSummary(rels)
	snap := &model.SafetySnapshot{
		RunID:          run.ID,
		RuleVersion:    run.RuleVersion,
		Status:         sum.Status,
		SnapshotStatus: model.SnapDraft,
		ConflictCount:  sum.ConflictCount,
		SafeCount:      sum.SafeCount,
		FrozenAt:       0,
		CreatedAt:      a.now(),
	}
	newStatus := state.NextVerificationStatus(sum.ConflictCount > 0)
	id, err := a.store.PersistVerification(store.VerificationWrite{
		RunID: run.ID, RunStatus: newStatus, Snapshot: snap, Relations: rels, Patches: patches,
	})
	if err != nil {
		return 0, "", fmt.Errorf("persist verification: %w", err)
	}
	return id, sum.Status, nil
}

// VerifyRun 统一时间窗、扩张可达包络、判定两两间隔并冻结为草稿快照。
func (a *App) VerifyRun(ctx context.Context, runID string) (*VerificationResult, error) {
	if err := guardCtx(ctx); err != nil {
		return nil, err
	}
	run, err := a.store.GetRun(runID)
	if err != nil {
		return nil, fmt.Errorf("verify get run: %w", err)
	}
	if !state.CanVerify(*run) {
		return nil, model.ErrRunSealed
	}
	now := a.now()
	acs, err := a.store.ListAircraft(runID)
	if err != nil {
		return nil, fmt.Errorf("verify list aircraft: %w", err)
	}

	type active struct {
		ac model.Aircraft
		it model.IntentSegment
	}
	var actives []active
	var patches []store.IntentStatusPatch
	for _, ac := range acs {
		if err := guardCtx(ctx); err != nil {
			return nil, err
		}
		if ac.Status == model.AircraftIsolated {
			continue
		}
		segs, err := a.store.ListIntentsByAircraft(runID, ac.ID)
		if err != nil {
			return nil, fmt.Errorf("verify list intents: %w", err)
		}
		it, ok := intent.LatestActive(segs, now)
		if !ok {
			if len(segs) > 0 {
				patches = append(patches, store.IntentStatusPatch{AircraftID: ac.ID, BulkStatus: model.IntentLost})
			}
			continue
		}
		cov, covErr := a.store.GetCovariance(runID, ac.ID)
		if covErr != nil {
			return nil, fmt.Errorf("verify covariance: %w", covErr)
		}
		overlayLiveCovariance(&it, cov)
		patches = append(patches, store.IntentStatusPatch{AircraftID: ac.ID, BulkStatus: model.IntentExcluded, ValidID: it.ID})
		actives = append(actives, active{ac: ac, it: it})
	}

	var rels []model.AvoidanceRelation
	intents := make([]model.IntentSegment, 0, len(actives))
	for _, x := range actives {
		intents = append(intents, x.it)
	}

	if len(actives) >= 2 {
		t0, t1, ok := intent.CommonWindow(intents)
		if ok {
			step := intent.SampleStep(t0, t1)
			for i := 0; i < len(actives); i++ {
				for j := i + 1; j < len(actives); j++ {
					a1, a2 := actives[i], actives[j]
					gap := run.MinSeparationM + a1.ac.RadiusM + a2.ac.RadiusM
					pr := conflict.EvaluatePair(a1.it, a2.it, t0, t1, step, run.ConfidenceK, gap)
					rels = append(rels, model.AvoidanceRelation{
						RunID: runID, SnapshotID: 0,
						AircraftA: pr.AircraftA, AircraftB: pr.AircraftB,
						Status: pr.Status, MinEffDistance: pr.MinEffDistance,
						RequiredGap: pr.RequiredGap, WorstT: pr.WorstT, VerifiedAt: now,
					})
				}
			}
		} else {
			for i := 0; i < len(actives); i++ {
				for j := i + 1; j < len(actives); j++ {
					a1, a2 := actives[i], actives[j]
					gap := run.MinSeparationM + a1.ac.RadiusM + a2.ac.RadiusM
					rels = append(rels, model.AvoidanceRelation{
						RunID: runID, SnapshotID: 0,
						AircraftA: a1.it.AircraftID, AircraftB: a2.it.AircraftID,
						Status: model.RelationInsufficient, MinEffDistance: -1e9,
						RequiredGap: gap, WorstT: a1.it.TEnd, VerifiedAt: now,
					})
				}
			}
		}
	}

	if err := guardCtx(ctx); err != nil {
		return nil, err
	}
	snapID, status, err := a.createSnapshotAndRels(ctx, run, rels, patches)
	if err != nil {
		return nil, err
	}
	conf, safe := 0, 0
	for _, r := range rels {
		if r.Status == model.RelationInsufficient {
			conf++
		} else if r.Status == model.RelationSafe {
			safe++
		}
	}
	return &VerificationResult{
		RunID:         runID,
		SnapshotID:    snapID,
		Status:        status,
		ConflictCount: conf,
		SafeCount:     safe,
		Relations:     rels,
	}, nil
}

// PublishSnapshot 将草稿快照发布并封存运行（旧发布快照置为 superseded）。
func (a *App) PublishSnapshot(ctx context.Context, runID string, snapID int64) error {
	if err := guardCtx(ctx); err != nil {
		return err
	}
	run, err := a.store.GetRun(runID)
	if err != nil {
		return fmt.Errorf("publish get run: %w", err)
	}
	if state.IsSealed(*run) {
		return model.ErrRunSealed
	}
	snap, err := a.store.GetSnapshot(snapID)
	if err != nil {
		return fmt.Errorf("publish get snapshot: %w", err)
	}
	if snap.RunID != runID {
		return model.ErrRunNotFound
	}
	if snap.SnapshotStatus != model.SnapDraft {
		return model.ErrSnapshotNotDraft
	}
	if err := a.store.PersistPublish(runID, snapID, snap.Status, a.now()); err != nil {
		return fmt.Errorf("persist publish: %w", err)
	}
	return nil
}

// IsolateAircraft 隔离失联/风险飞行器，使其退出编队验证。
func (a *App) IsolateAircraft(ctx context.Context, runID, aircraftID string) error {
	if err := guardCtx(ctx); err != nil {
		return err
	}
	run, err := a.store.GetRun(runID)
	if err != nil {
		return fmt.Errorf("isolate get run: %w", err)
	}
	if state.IsSealed(*run) {
		return model.ErrRunSealed
	}
	ac, err := a.store.GetAircraft(aircraftID)
	if err != nil {
		return fmt.Errorf("isolate get aircraft: %w", err)
	}
	if ac.RunID != runID {
		return model.ErrAircraftNotFound
	}
	if err := a.store.PersistIsolation(runID, aircraftID); err != nil {
		return fmt.Errorf("persist isolation: %w", err)
	}
	return nil
}

// ReinstateAircraft 将已隔离飞行器恢复为活跃。
func (a *App) ReinstateAircraft(ctx context.Context, runID, aircraftID string) error {
	if err := guardCtx(ctx); err != nil {
		return err
	}
	run, err := a.store.GetRun(runID)
	if err != nil {
		return fmt.Errorf("reinstate get run: %w", err)
	}
	if state.IsSealed(*run) {
		return model.ErrRunSealed
	}
	ac, err := a.store.GetAircraft(aircraftID)
	if err != nil {
		return fmt.Errorf("reinstate get aircraft: %w", err)
	}
	if ac.RunID != runID {
		return model.ErrAircraftNotFound
	}
	if err := a.store.PersistReinstate(aircraftID); err != nil {
		return fmt.Errorf("persist reinstate: %w", err)
	}
	return nil
}

// RunStats 是运行的多维统计。
type RunStats struct {
	RunID         string `json:"run_id"`
	Status        string `json:"status"`
	AircraftCount int    `json:"aircraft_count"`
	IntentCount   int    `json:"intent_count"`
	RelationCount int    `json:"relation_count"`
	SnapshotCount int    `json:"snapshot_count"`
	ConflictCount int    `json:"conflict_count"`
}

// GetStats 汇总运行的健康度指标。
func (a *App) GetStats(runID string) (*RunStats, error) {
	run, err := a.store.GetRun(runID)
	if err != nil {
		return nil, err
	}
	acs, err := a.store.ListAircraft(runID)
	if err != nil {
		return nil, fmt.Errorf("stats aircraft: %w", err)
	}
	intents, err := a.store.ListIntentsByRun(runID)
	if err != nil {
		return nil, fmt.Errorf("stats intents: %w", err)
	}
	snaps, err := a.store.ListSnapshots(runID)
	if err != nil {
		return nil, fmt.Errorf("stats snapshots: %w", err)
	}
	conf := 0
	relCount := 0
	relSnapID := int64(0)
	if pub, pubErr := a.store.LatestPublishedSnapshot(runID); pubErr != nil {
		return nil, fmt.Errorf("stats published: %w", pubErr)
	} else if pub != nil {
		relSnapID = pub.ID
	} else if len(snaps) > 0 {
		relSnapID = snaps[0].ID
	}
	if relSnapID > 0 {
		rels, relErr := a.store.ListRelations(runID, relSnapID)
		if relErr != nil {
			return nil, fmt.Errorf("stats relations: %w", relErr)
		}
		relCount = len(rels)
		for _, r := range rels {
			if r.Status == model.RelationInsufficient {
				conf++
			}
		}
	}
	return &RunStats{
		RunID:         runID,
		Status:        run.Status,
		AircraftCount: len(acs),
		IntentCount:   len(intents),
		RelationCount: relCount,
		SnapshotCount: len(snaps),
		ConflictCount: conf,
	}, nil
}

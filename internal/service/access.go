package service

import (
	"context"
	"fmt"

	"task276-droneformation/internal/conflict"
	"task276-droneformation/internal/intent"
	"task276-droneformation/internal/model"
)

// overlayLiveCovariance 用库中最新定位不确定度覆盖意图段上的快照字段，
// 保证验证使用的可达包络与 PUT covariance 之后的主数据一致。
func overlayLiveCovariance(it *model.IntentSegment, cov *model.Covariance) {
	if it == nil || cov == nil {
		return
	}
	it.SigX = cov.SigX
	it.SigY = cov.SigY
	it.SigZ = cov.SigZ
	it.SigRateX = cov.SigRateX
	it.SigRateY = cov.SigRateY
	it.SigRateZ = cov.SigRateZ
}

// 以下为对存储层的只读/覆盖访问封装，供 HTTP 层调用。

func (a *App) GetRun(id string) (*model.FormationRun, error)       { return a.store.GetRun(id) }
func (a *App) ListRuns() ([]model.FormationRun, error)             { return a.store.ListRuns() }
func (a *App) GetAircraft(id string) (*model.Aircraft, error)      { return a.store.GetAircraft(id) }
func (a *App) ListAircraft(runID string) ([]model.Aircraft, error) { return a.store.ListAircraft(runID) }

func (a *App) ListIntentsByAircraft(runID, aircraftID string) ([]model.IntentSegment, error) {
	return a.store.ListIntentsByAircraft(runID, aircraftID)
}
func (a *App) ListIntentsByRun(runID string) ([]model.IntentSegment, error) {
	return a.store.ListIntentsByRun(runID)
}

func (a *App) GetCovariance(runID, aircraftID string) (*model.Covariance, error) {
	return a.store.GetCovariance(runID, aircraftID)
}

func (a *App) ListRelations(runID string, snapID int64) ([]model.AvoidanceRelation, error) {
	return a.store.ListRelations(runID, snapID)
}
func (a *App) GetRelation(id int64) (*model.AvoidanceRelation, error) {
	return a.store.GetRelation(id)
}

func (a *App) GetSnapshot(id int64) (*model.SafetySnapshot, error) {
	snap, err := a.store.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	run, err := a.store.GetRun(snap.RunID)
	if err != nil {
		return snap, nil
	}
	acs, err := a.store.ListAircraft(snap.RunID)
	if err != nil || run == nil {
		return snap, nil
	}
	now := a.now()
	var intents []model.IntentSegment
	for _, ac := range acs {
		if ac.Status == model.AircraftIsolated {
			continue
		}
		segs, listErr := a.store.ListIntentsByAircraft(snap.RunID, ac.ID)
		if listErr != nil {
			continue
		}
		it, ok := intent.LatestActive(segs, now)
		if !ok {
			continue
		}
		cov, _ := a.store.GetCovariance(snap.RunID, ac.ID)
		overlayLiveCovariance(&it, cov)
		intents = append(intents, it)
	}
	conf := 0
	if len(intents) >= 2 {
		t0, t1, ok := intent.CommonWindow(intents)
		if ok {
			step := intent.SampleStep(t0, t1)
			for i := 0; i < len(intents); i++ {
				for j := i + 1; j < len(intents); j++ {
					gap := run.MinSeparationM + 1.0
					pr := conflict.EvaluatePair(intents[i], intents[j], t0, t1, step, run.ConfidenceK, gap)
					if pr.Status == model.RelationInsufficient {
						conf++
					}
				}
			}
		}
	}
	snap.ConflictCount = conf
	return snap, nil
}
func (a *App) ListSnapshots(runID string) ([]model.SafetySnapshot, error) {
	return a.store.ListSnapshots(runID)
}

// UpdateCovariance 覆盖飞行器的定位不确定度（基准标准差与增长率）。
func (a *App) UpdateCovariance(ctx context.Context, runID, aircraftID string, sigX, sigY, sigZ, srX, srY, srZ float64) (*model.Covariance, error) {
	if err := guardCtx(ctx); err != nil {
		return nil, err
	}
	if sigX < 0 || sigY < 0 || sigZ < 0 {
		return nil, model.ErrCovarianceIllegal
	}
	c := &model.Covariance{
		RunID: runID, AircraftID: aircraftID,
		SigX: sigX, SigY: sigY, SigZ: sigZ,
		SigRateX: srX, SigRateY: srY, SigRateZ: srZ,
		UpdatedAt: a.now(),
	}
	if err := a.store.UpsertCovariance(c); err != nil {
		return nil, fmt.Errorf("upsert covariance: %w", err)
	}
	return c, nil
}

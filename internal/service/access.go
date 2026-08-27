package service

import (
	"context"
	"fmt"

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

// GetSnapshot 返回一份已冻结的安全快照。安全快照是不可变件：其
// ConflictCount/SafeCount/Status 在验证时刻就已锁定，不会因事后注入
// 新意图、改协方差或隔离飞行器等现场数据变化而改写。此处仅做读出，
// 不再从实时意图重新推导，以保证快照记录的历史结论稳定。
func (a *App) GetSnapshot(id int64) (*model.SafetySnapshot, error) {
	return a.store.GetSnapshot(id)
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

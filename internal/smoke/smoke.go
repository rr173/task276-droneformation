// Package smoke 实现 --smoke-test 自检：建立真实实体、触发验证、封存快照，
// 随后关闭并重新打开数据库验证持久化与重启恢复，以 0 退出码结束。
package smoke

import (
	"context"
	"fmt"

	"task276-droneformation/internal/model"
	"task276-droneformation/internal/service"
	"task276-droneformation/internal/store"
)

func intent(aircraftID string, t0, t1 int64, px, py, pz, vx, vy, vz, sig float64) service.IntentInput {
	return service.IntentInput{
		Seq:               1,
		TStart:            t0,
		TEnd:              t1,
		PosX:              px, PosY: py, PosZ: pz,
		VelX: vx, VelY: vy, VelZ: vz,
		SigX: sig, SigY: sig, SigZ: sig,
		SigRateX: 0, SigRateY: 0, SigRateZ: 0,
		RefHeightBaseline: 0,
	}
}

// Run 执行完整自检流程。
func Run(dbPath string) error {
	baseTime := int64(1_700_000_000_000) // 固定基准时间，保证确定性

	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	app := service.NewApp(st)
	app.SetClock(func() int64 { return baseTime })
	ctx := context.Background()

	// 1. 建立运行与三架飞行器
	run, err := app.CreateRun(ctx, "smoke-run")
	if err != nil {
		return err
	}
	a, err := app.RegisterAircraft(ctx, run.ID, "ALPHA", 0.5, 0)
	if err != nil {
		return err
	}
	b, err := app.RegisterAircraft(ctx, run.ID, "BRAVO", 0.5, 0)
	if err != nil {
		return err
	}
	c, err := app.RegisterAircraft(ctx, run.ID, "CHARLIE", 0.5, 0)
	if err != nil {
		return err
	}

	// 2. 注入意图段（窗口 1000..5000ms）
	t0 := baseTime + 1000
	t1 := baseTime + 5000
	if _, err := app.IngestIntent(ctx, run.ID, a.ID, intent(a.ID, t0, t1, 0, 0, 0, 1, 0, 0, 0.5)); err != nil {
		return err
	}
	if _, err := app.IngestIntent(ctx, run.ID, b.ID, intent(b.ID, t0, t1, 100, 0, 0, 0, 0, 0, 0.5)); err != nil {
		return err
	}
	if _, err := app.IngestIntent(ctx, run.ID, c.ID, intent(c.ID, t0, t1, 5, 0, 0, 0, 0, 0, 0.5)); err != nil {
		return err
	}

	// 3. 验证：预期 A-B / B-C 安全，A-C 冲突
	res, err := app.VerifyRun(ctx, run.ID)
	if err != nil {
		return err
	}
	if res.Status != model.RunConflict {
		return fmt.Errorf("expected conflict, got %s", res.Status)
	}
	if res.ConflictCount != 1 {
		return fmt.Errorf("expected 1 conflict, got %d", res.ConflictCount)
	}

	// 4. 发布快照（封存运行）
	if err := app.PublishSnapshot(ctx, run.ID, res.SnapshotID); err != nil {
		return err
	}

	// 5. 关闭并重新打开数据库，验证持久化与重启恢复
	if err := st.Close(); err != nil {
		return err
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st2.Close()
	app2 := service.NewApp(st2)
	app2.SetClock(func() int64 { return baseTime })

	r2, err := app2.GetRun(run.ID)
	if err != nil {
		return err
	}
	if r2.Status != model.RunSealed {
		return fmt.Errorf("run not sealed after reopen: %s", r2.Status)
	}
	snaps, err := app2.ListSnapshots(run.ID)
	if err != nil {
		return err
	}
	if len(snaps) != 1 || snaps[0].SnapshotStatus != model.SnapPublished {
		return fmt.Errorf("snapshot not published after reopen")
	}
	rels, err := app2.ListRelations(run.ID, snaps[0].ID)
	if err != nil {
		return err
	}
	if len(rels) != 3 {
		return fmt.Errorf("expected 3 relations, got %d", len(rels))
	}
	ia, err := app2.ListIntentsByAircraft(run.ID, a.ID)
	if err != nil {
		return err
	}
	if len(ia) != 1 || ia[0].Status != model.IntentValid {
		return fmt.Errorf("intent status not persisted correctly")
	}
	return nil
}

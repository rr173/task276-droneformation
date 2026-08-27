package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"task276-droneformation/internal/model"
	"task276-droneformation/internal/store"
)

func newTestApp(t *testing.T) (*App, func()) {
	t.Helper()
	db := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	app := NewApp(st)
	app.SetClock(func() int64 { return 1_700_000_000_000 })
	return app, func() { _ = st.Close() }
}

func TestFullFlowConflictThenSeal(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	ctx := context.Background()
	run, err := app.CreateRun(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := app.RegisterAircraft(ctx, run.ID, "A", 0.5, 0)
	b, _ := app.RegisterAircraft(ctx, run.ID, "B", 0.5, 0)
	c, _ := app.RegisterAircraft(ctx, run.ID, "C", 0.5, 0)

	base := int64(1_700_000_000_000)
	_, err = app.IngestIntent(ctx, run.ID, a.ID, IntentInput{Seq: 1, TStart: base + 1000, TEnd: base + 5000, PosX: 0, VelX: 1, SigX: 0.5, SigY: 0.5, SigZ: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.IngestIntent(ctx, run.ID, b.ID, IntentInput{Seq: 1, TStart: base + 1000, TEnd: base + 5000, PosX: 100, SigX: 0.5, SigY: 0.5, SigZ: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.IngestIntent(ctx, run.ID, c.ID, IntentInput{Seq: 1, TStart: base + 1000, TEnd: base + 5000, PosX: 5, SigX: 0.5, SigY: 0.5, SigZ: 0.5})
	if err != nil {
		t.Fatal(err)
	}

	res, err := app.VerifyRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.RunConflict || res.ConflictCount != 1 {
		t.Fatalf("expected conflict with 1 conflict, got %s/%d", res.Status, res.ConflictCount)
	}

	if err := app.PublishSnapshot(ctx, run.ID, res.SnapshotID); err != nil {
		t.Fatal(err)
	}
	run2, err := app.GetRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run2.Status != model.RunSealed {
		t.Fatalf("expected sealed, got %s", run2.Status)
	}

	// 封存后再次验证应被拒绝。
	if _, err := app.VerifyRun(ctx, run.ID); !errors.Is(err, model.ErrRunSealed) {
		t.Fatalf("expected sealed error, got %v", err)
	}
}

// TestVerifyUsesLatestCovarianceAfterUpdate 复现：IngestIntent 时小不确定度被判安全，
// 随后 PUT 调大定位不确定度，再次验证必须用库里最新值——否则包络仍按旧值算，
// 本应判间隔不足的一对会被误判为安全。
func TestVerifyUsesLatestCovarianceAfterUpdate(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	ctx := context.Background()
	run, err := app.CreateRun(ctx, "cov")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := app.RegisterAircraft(ctx, run.ID, "A", 0.5, 0)
	b, _ := app.RegisterAircraft(ctx, run.ID, "B", 0.5, 0)

	base := int64(1_700_000_000_000)
	// 中心距 7m，小不确定度 sig=0.5：包络半径各 1.5，有效间隔 4.0 ≥ gap 3.0 → 安全。
	_, err = app.IngestIntent(ctx, run.ID, a.ID, IntentInput{Seq: 1, TStart: base + 1000, TEnd: base + 5000, PosX: 0, SigX: 0.5, SigY: 0.5, SigZ: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.IngestIntent(ctx, run.ID, b.ID, IntentInput{Seq: 1, TStart: base + 1000, TEnd: base + 5000, PosX: 7, SigX: 0.5, SigY: 0.5, SigZ: 0.5})
	if err != nil {
		t.Fatal(err)
	}

	res, err := app.VerifyRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.RunSafe || res.ConflictCount != 0 {
		t.Fatalf("expected safe before covariance update, got %s/%d", res.Status, res.ConflictCount)
	}

	// 调大两架无人机的定位不确定度：sig=2.0 → 包络半径各 6.0，有效间隔 -5.0 < gap 3.0 → 间隔不足。
	if _, err := app.UpdateCovariance(ctx, run.ID, a.ID, 2.0, 2.0, 2.0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateCovariance(ctx, run.ID, b.ID, 2.0, 2.0, 2.0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}

	res2, err := app.VerifyRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Status != model.RunConflict || res2.ConflictCount != 1 {
		t.Fatalf("expected conflict after covariance update, got %s/%d (minEff=%.3f)",
			res2.Status, res2.ConflictCount, minEffOf(res2))
	}
}

// minEffOf 取验证结果中唯一一对关系的最小有效间隔，便于失败诊断。
func minEffOf(r *VerificationResult) float64 {
	for _, rel := range r.Relations {
		return rel.MinEffDistance
	}
	return 0
}

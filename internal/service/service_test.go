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

// TestSnapshotConflictCountFrozenAfterNewIntents 验证已生成的安全快照是不可变件：
// 事后注入新意图（不重新验证）时，再读该快照的 ConflictCount 不得被现场数据改写。
func TestSnapshotConflictCountFrozenAfterNewIntents(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	ctx := context.Background()
	run, err := app.CreateRun(ctx, "frozen")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := app.RegisterAircraft(ctx, run.ID, "A", 0.5, 0)
	b, _ := app.RegisterAircraft(ctx, run.ID, "B", 0.5, 0)

	base := int64(1_700_000_000_000)
	// 两架飞行器相距 100、静止，可达包络远小于要求间隔 → 安全，0 冲突。
	for _, in := range []struct {
		id   string
		posX float64
	}{
		{a.ID, 0},
		{b.ID, 100},
	} {
		if _, err := app.IngestIntent(ctx, run.ID, in.id, IntentInput{
			Seq: 1, TStart: base + 1000, TEnd: base + 5000,
			PosX: in.posX, SigX: 0.5, SigY: 0.5, SigZ: 0.5,
		}); err != nil {
			t.Fatal(err)
		}
	}

	res, err := app.VerifyRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.RunSafe || res.ConflictCount != 0 {
		t.Fatalf("expected safe/0 conflict at verify, got %s/%d", res.Status, res.ConflictCount)
	}
	frozen := res.ConflictCount

	// 注入新意图（seq=2）：把 A 挪到紧挨 B，现场数据下将产生一个冲突，
	// 但未重新验证，已生成的快照结论不应改变。
	if _, err := app.IngestIntent(ctx, run.ID, a.ID, IntentInput{
		Seq: 2, TStart: base + 1000, TEnd: base + 5000,
		PosX: 99, SigX: 0.5, SigY: 0.5, SigZ: 0.5,
	}); err != nil {
		t.Fatal(err)
	}

	snap, err := app.GetSnapshot(res.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ConflictCount != frozen {
		t.Fatalf("frozen snapshot ConflictCount mutated: got %d, want %d", snap.ConflictCount, frozen)
	}
}


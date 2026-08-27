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

// TestIngestDuplicateSeqPreservesErrorChain 验证同一飞行器再次上报相同序号时，
// 服务层返回的错误经 %w 包装后仍可由 errors.Is 识别为 ErrDuplicateSeq，
// 保证领域冲突沿错误链传递至接口层而非被掩盖。
func TestIngestDuplicateSeqPreservesErrorChain(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	ctx := context.Background()
	run, err := app.CreateRun(ctx, "dup")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := app.RegisterAircraft(ctx, run.ID, "A", 0.5, 0)

	base := int64(1_700_000_000_000)
	in := IntentInput{Seq: 1, TStart: base + 1000, TEnd: base + 5000, PosX: 0, VelX: 1, SigX: 0.5, SigY: 0.5, SigZ: 0.5}
	if _, err := app.IngestIntent(ctx, run.ID, a.ID, in); err != nil {
		t.Fatalf("first ingest: %v", err)
	}

	// 再次上报相同序号：错误链中必须可经 errors.Is 命中 ErrDuplicateSeq。
	_, err = app.IngestIntent(ctx, run.ID, a.ID, in)
	if !errors.Is(err, model.ErrDuplicateSeq) {
		t.Fatalf("expected errors.Is(err, ErrDuplicateSeq), got %v", err)
	}
}


package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
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

// TestIngestIntentConcurrentNoRace 模拟二十个协程同时给不同飞行器写入最新意图，
// 验证：race detector 不报警、全部意图安全落库、最新缓存命中一致。
func TestIngestIntentConcurrentNoRace(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	ctx := context.Background()
	run, err := app.CreateRun(ctx, "concurrent")
	if err != nil {
		t.Fatal(err)
	}

	const n = 20
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ac, err := app.RegisterAircraft(ctx, run.ID, "AC", 0.5, 0)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = ac.ID
	}

	base := int64(1_700_000_000_000)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			in := IntentInput{
				Seq:    int64(i + 1),
				TStart: base + 1000,
				TEnd:   base + 5000,
				PosX:   float64(i),
				SigX:   0.5, SigY: 0.5, SigZ: 0.5,
			}
			if _, err := app.IngestIntent(ctx, run.ID, ids[i], in); err != nil {
				t.Errorf("aircraft %d ingest: %v", i, err)
			}
		}()
	}
	wg.Wait()

	// 落库计数：二十架各一条。
	segs, err := app.ListIntentsByRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != n {
		t.Fatalf("expected %d persisted intents, got %d", n, len(segs))
	}

	// 缓存命中：每架都能读到自己写入的那条。
	for i := 0; i < n; i++ {
		got, ok := app.latestIntent(run.ID, ids[i])
		if !ok {
			t.Fatalf("cache miss for aircraft %d", i)
		}
		if got.Seq != int64(i+1) {
			t.Fatalf("aircraft %d: cached seq=%d, want %d", i, got.Seq, i+1)
		}
	}
}

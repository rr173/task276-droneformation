package service

import (
	"context"
	"errors"
	"fmt"
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

// TestIngestIntentConcurrent 模拟编队内二十架飞行器同时上报各自的意图段：
// 全部必须成功落库，并发不得互相踩事务或卡死。
func TestIngestIntentConcurrent(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	ctx := context.Background()
	run, err := app.CreateRun(ctx, "swarm")
	if err != nil {
		t.Fatal(err)
	}
	const n = 20
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ac, err := app.RegisterAircraft(ctx, run.ID, fmt.Sprintf("AC%d", i), 0.5, 0)
		if err != nil {
			t.Fatalf("register %d: %v", i, err)
		}
		ids[i] = ac.ID
	}

	base := int64(1_700_000_000_000)
	var wg sync.WaitGroup
	errs := make(chan error, n)
	wg.Add(n)
	for i := range ids {
		i := i
		go func() {
			defer wg.Done()
			_, err := app.IngestIntent(ctx, run.ID, ids[i], IntentInput{
				Seq: 1, TStart: base + 1000, TEnd: base + 5000,
				PosX: float64(i), SigX: 0.5, SigY: 0.5, SigZ: 0.5,
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	var failed int
	for err := range errs {
		if err != nil {
			failed++
			t.Logf("ingest failed: %v", err)
		}
	}
	if failed > 0 {
		t.Fatalf("%d/%d concurrent ingests failed", failed, n)
	}

	its, err := app.ListIntentsByRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(its) != n {
		t.Fatalf("expected %d persisted intents, got %d", n, len(its))
	}
}

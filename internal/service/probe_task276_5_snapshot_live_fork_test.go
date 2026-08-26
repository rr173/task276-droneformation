package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"task276-droneformation/internal/store"
)

func newProbeApp(t *testing.T) *App {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	app := NewApp(st)
	app.SetClock(func() int64 { return 1_700_000_000_000 })
	return app
}

func mustRunAC(t *testing.T, app *App, n int) (string, []string) {
	t.Helper()
	ctx := context.Background()
	run, err := app.CreateRun(ctx, "probe")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ac, err := app.RegisterAircraft(ctx, run.ID, fmt.Sprintf("AC-%d", i), 0.5, 0)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, ac.ID)
	}
	return run.ID, ids
}

func farIntent(seq int64, x float64) IntentInput {
	base := int64(1_700_000_000_000)
	return IntentInput{Seq: seq, TStart: base + 1000, TEnd: base + 5000, PosX: x, SigX: 0.5, SigY: 0.5, SigZ: 0.5}
}

func TestDraftSnapshotNotRecomputedFromLive(t *testing.T) {
	app := newProbeApp(t)
	ctx := context.Background()
	run, err := app.CreateRun(ctx, "snap")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := app.RegisterAircraft(ctx, run.ID, "A", 0.5, 0)
	b, _ := app.RegisterAircraft(ctx, run.ID, "B", 0.5, 0)
	if _, err := app.IngestIntent(ctx, run.ID, a.ID, farIntent(1, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.IngestIntent(ctx, run.ID, b.ID, farIntent(1, 100)); err != nil {
		t.Fatal(err)
	}
	res, err := app.VerifyRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.ConflictCount != 0 {
		t.Fatalf("setup want safe, got %d", res.ConflictCount)
	}
	c, err := app.RegisterAircraft(ctx, run.ID, "C", 0.5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.IngestIntent(ctx, run.ID, c.ID, farIntent(1, 1)); err != nil {
		t.Fatal(err)
	}
	snap, err := app.GetSnapshot(res.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ConflictCount != 0 {
		t.Fatalf("draft snapshot mutated by live intents: conflict=%d", snap.ConflictCount)
	}
}

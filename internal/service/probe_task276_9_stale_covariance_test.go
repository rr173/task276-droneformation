package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"task276-droneformation/internal/model"
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

func TestVerifyUsesLiveCovariance(t *testing.T) {
	app := newProbeApp(t)
	ctx := context.Background()
	run, err := app.CreateRun(ctx, "cov")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := app.RegisterAircraft(ctx, run.ID, "A", 0.5, 0)
	b, _ := app.RegisterAircraft(ctx, run.ID, "B", 0.5, 0)
	if _, err := app.IngestIntent(ctx, run.ID, a.ID, farIntent(1, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.IngestIntent(ctx, run.ID, b.ID, farIntent(1, 12)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateCovariance(ctx, run.ID, a.ID, 8, 8, 8, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateCovariance(ctx, run.ID, b.ID, 8, 8, 8, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	res, err := app.VerifyRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.ConflictCount < 1 || res.Status != model.RunConflict {
		t.Fatalf("live covariance must create conflict, got status=%s conflicts=%d", res.Status, res.ConflictCount)
	}
}

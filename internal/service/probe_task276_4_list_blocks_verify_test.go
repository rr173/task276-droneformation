package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
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

func TestListIntentsDoesNotBlockVerify(t *testing.T) {
	app := newProbeApp(t)
	ctx := context.Background()
	runID, ids := mustRunAC(t, app, 2)
	if _, err := app.IngestIntent(ctx, runID, ids[0], farIntent(1, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.IngestIntent(ctx, runID, ids[1], farIntent(1, 80)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ListIntentsByRun(runID); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := app.VerifyRun(ctx, runID)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("verify blocked after listing intents")
	}
}

package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
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

func TestIntentCacheNoRace(t *testing.T) {
	app := newProbeApp(t)
	runID, ids := mustRunAC(t, app, 20)
	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := app.IngestIntent(context.Background(), runID, ids[i], farIntent(1, float64(i)*8)); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}

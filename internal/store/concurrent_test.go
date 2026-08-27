package store

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"task276-droneformation/internal/model"
)

// TestPersistIntentBundleConcurrent 模拟编队内多架飞行器并发上报意图段：
// 所有写入必须全部成功落库，不能因并发互相踩事务或卡死。
func TestPersistIntentBundleConcurrent(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "concurrent.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	const n = 20
	// 预置一个运行与 n 架飞行器，作为并发上报的落脚点。
	if err := st.CreateRun(&model.FormationRun{
		ID: "run-1", Name: "swarm", Status: model.RunReceiving,
		MinSeparationM: 2, ConfidenceK: 3, RuleVersion: 1, CreatedAt: 1,
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		aid := fmt.Sprintf("ac-%d", i)
		if err := st.CreateAircraft(&model.Aircraft{
			ID: aid, RunID: "run-1", Callsign: aid,
			RadiusM: 0.5, Status: model.AircraftActive, CreatedAt: 1,
		}); err != nil {
			t.Fatalf("create aircraft %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			aid := fmt.Sprintf("ac-%d", i)
			it := &model.IntentSegment{
				RunID: "run-1", AircraftID: aid, Seq: 1,
				TStart: 10, TEnd: 20,
				Status: model.IntentRaw, CreatedAt: 1,
			}
			c := &model.Covariance{
				RunID: "run-1", AircraftID: aid,
				SigX: 0.5, SigY: 0.5, SigZ: 0.5, UpdatedAt: 1,
			}
			errs <- st.PersistIntentBundle(IntentBundle{Intent: it, Covariance: c, LastSeq: 1, LastAt: 1})
		}()
	}
	wg.Wait()
	close(errs)

	var failed int
	for err := range errs {
		if err != nil {
			failed++
			t.Logf("concurrent persist failed: %v", err)
		}
	}
	if failed > 0 {
		t.Fatalf("%d/%d concurrent intent bundles failed to persist", failed, n)
	}

	its, err := st.ListIntentsByRun("run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(its) != n {
		t.Fatalf("expected %d persisted intents, got %d", n, len(its))
	}
}

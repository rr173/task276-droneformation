package service

import (
	"context"
	"errors"
	"testing"

	"task276-droneformation/internal/model"
)

// TestBatchIngestIntentHonorsCancellation 验证批量上报意图途中调用方取消后，
// 服务停止后续写入、不再落盘任何剩余意图段。
func TestBatchIngestIntentHonorsCancellation(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	ctx := context.Background()
	run, err := app.CreateRun(ctx, "cancel-batch")
	if err != nil {
		t.Fatal(err)
	}
	// 注册多架飞行器，构造一个会被取消的批量请求。
	const n = 6
	aircraftIDs := make([]string, 0, n)
	items := make(map[string]IntentInput, n)
	base := int64(1_700_000_000_000)
	for i := 0; i < n; i++ {
		ac, err := app.RegisterAircraft(ctx, run.ID, "AC", 0.5, 0)
		if err != nil {
			t.Fatal(err)
		}
		aircraftIDs = append(aircraftIDs, ac.ID)
		items[ac.ID] = IntentInput{
			Seq:    1,
			TStart: base + 1000,
			TEnd:   base + 5000,
			PosX:   float64(i * 10),
			SigX:   0.5, SigY: 0.5, SigZ: 0.5,
		}
	}

	// 调用方在发起批量上报前即取消：服务必须立即停止，不落盘任何剩余意图。
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = app.BatchIngestIntent(cancelCtx, run.ID, items)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	persisted, err := app.ListIntentsByRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 0 {
		t.Fatalf("canceled batch must not persist intents, got %d", len(persisted))
	}
	_ = aircraftIDs
	_ = model.IntentRaw
}

// TestBatchIngestIntentCompletesWhenNotCancelled 验证未取消时批量上报全部落盘。
func TestBatchIngestIntentCompletesWhenNotCancelled(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	ctx := context.Background()
	run, err := app.CreateRun(ctx, "happy-batch")
	if err != nil {
		t.Fatal(err)
	}
	const n = 4
	items := make(map[string]IntentInput, n)
	base := int64(1_700_000_000_000)
	for i := 0; i < n; i++ {
		ac, err := app.RegisterAircraft(ctx, run.ID, "AC", 0.5, 0)
		if err != nil {
			t.Fatal(err)
		}
		items[ac.ID] = IntentInput{
			Seq:    1,
			TStart: base + 1000,
			TEnd:   base + 5000,
			PosX:   float64(i * 10),
			SigX:   0.5, SigY: 0.5, SigZ: 0.5,
		}
	}

	if err := app.BatchIngestIntent(ctx, run.ID, items); err != nil {
		t.Fatalf("batch ingest: %v", err)
	}
	persisted, err := app.ListIntentsByRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != n {
		t.Fatalf("expected %d persisted intents, got %d", n, len(persisted))
	}
}

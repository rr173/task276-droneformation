package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task276-droneformation/internal/service"
	"task276-droneformation/internal/store"
)

func TestCanceledBatchStopsWrites(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "cancel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	app := service.NewApp(st)
	app.SetClock(func() int64 { return 1_700_000_000_000 })
	h := NewHandler(app)
	ctx := context.Background()
	run, err := app.CreateRun(ctx, "cancel")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		ac, err := app.RegisterAircraft(ctx, run.ID, "C", 0.5, 0)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, ac.ID)
	}
	items := make([]map[string]any, 0, 8)
	for i, id := range ids {
		items = append(items, map[string]any{
			"aircraft_id": id, "seq": 1, "t_start": 1700000001000, "t_end": 1700000005000,
			"pos_x": float64(i) * 10, "sig_x": 0.5, "sig_y": 0.5, "sig_z": 0.5,
		})
	}
	raw, _ := json.Marshal(map[string]any{"items": items})
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/runs/"+run.ID+"/intents/batch", bytes.NewReader(raw))
	req = req.WithContext(canceled)
	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code == 200 {
		t.Fatalf("canceled batch must not succeed: %s", rec.Body.String())
	}
	got, err := app.ListIntentsByRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("canceled batch still persisted %d intents", len(got))
	}
}

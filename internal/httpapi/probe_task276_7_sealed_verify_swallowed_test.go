package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"task276-droneformation/internal/service"
	"task276-droneformation/internal/store"
)

func TestSealedVerifyNotSwallowed(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "seal.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	app := service.NewApp(st)
	app.SetClock(func() int64 { return 1_700_000_000_000 })
	h := NewHandler(app)
	ctx := context.Background()
	run, err := app.CreateRun(ctx, "seal")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := app.RegisterAircraft(ctx, run.ID, "A", 0.5, 0)
	b, _ := app.RegisterAircraft(ctx, run.ID, "B", 0.5, 0)
	body := `{"seq":1,"t_start":1700000001000,"t_end":1700000005000,"pos_x":0,"sig_x":0.5,"sig_y":0.5,"sig_z":0.5}`
	bodyB := `{"seq":1,"t_start":1700000001000,"t_end":1700000005000,"pos_x":80,"sig_x":0.5,"sig_y":0.5,"sig_z":0.5}`
	for aid, raw := range map[string]string{a.ID: body, b.ID: bodyB} {
		req := httptest.NewRequest(http.MethodPost, "/api/runs/"+run.ID+"/aircraft/"+aid+"/intents", strings.NewReader(raw))
		rec := httptest.NewRecorder()
		h.Router().ServeHTTP(rec, req)
		if rec.Code != 201 {
			t.Fatalf("ingest %d %s", rec.Code, rec.Body.String())
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/runs/"+run.ID+"/verify", nil)
	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("verify %d %s", rec.Code, rec.Body.String())
	}
	var vr map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &vr); err != nil {
		t.Fatal(err)
	}
	snapID := int64(vr["SnapshotID"].(float64))
	pub, _ := json.Marshal(map[string]int64{"snapshot_id": snapID})
	req = httptest.NewRequest(http.MethodPost, "/api/runs/"+run.ID+"/publish", strings.NewReader(string(pub)))
	rec = httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("publish %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/runs/"+run.ID+"/verify", nil)
	rec = httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code != 409 {
		t.Fatalf("sealed verify want 409, got %d body=%s", rec.Code, rec.Body.String())
	}
}

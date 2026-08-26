package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"task276-droneformation/internal/service"
	"task276-droneformation/internal/store"
)

func TestDuplicateSeqMapsConflict(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "dup.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	app := service.NewApp(st)
	app.SetClock(func() int64 { return 1_700_000_000_000 })
	h := NewHandler(app)
	ctx := context.Background()
	run, err := app.CreateRun(ctx, "dup")
	if err != nil {
		t.Fatal(err)
	}
	ac, err := app.RegisterAircraft(ctx, run.ID, "A", 0.5, 0)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"seq":1,"t_start":1700000001000,"t_end":1700000005000,"pos_x":0,"sig_x":0.5,"sig_y":0.5,"sig_z":0.5}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/"+run.ID+"/aircraft/"+ac.ID+"/intents", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("first ingest %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/runs/"+run.ID+"/aircraft/"+ac.ID+"/intents", strings.NewReader(body))
	rec = httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code != 409 {
		t.Fatalf("duplicate want 409, got %d body=%s", rec.Code, rec.Body.String())
	}
}

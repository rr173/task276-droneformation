package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"task276-droneformation/internal/model"
	"task276-droneformation/internal/service"
	"task276-droneformation/internal/store"
)

func mustCreateRun(t *testing.T, app *service.App, name string) *model.FormationRun {
	t.Helper()
	run, err := app.CreateRun(context.Background(), name)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

func mustRegisterAircraft(t *testing.T, app *service.App, runID, callsign string, radius float64) *model.Aircraft {
	t.Helper()
	ac, err := app.RegisterAircraft(context.Background(), runID, callsign, radius, 0)
	if err != nil {
		t.Fatalf("register aircraft %s: %v", callsign, err)
	}
	return ac
}

func mustIngest(t *testing.T, app *service.App, runID, aircraftID string, in service.IntentInput) {
	t.Helper()
	if _, err := app.IngestIntent(context.Background(), runID, aircraftID, in); err != nil {
		t.Fatalf("ingest intent %s: %v", aircraftID, err)
	}
}

func mustVerify(t *testing.T, app *service.App, runID string) *service.VerificationResult {
	t.Helper()
	res, err := app.VerifyRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	return res
}

func newHandler(t *testing.T) *Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "http.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return NewHandler(service.NewApp(st))
}

func TestHealthOK(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("health status %d", rec.Code)
	}
}

func TestCreateAndGetRun(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/runs", strings.NewReader(`{"name":"alpha"}`))
	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create status %d body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing run id")
	}
	req = httptest.NewRequest(http.MethodGet, "/api/runs/"+id, nil)
	rec = httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("get status %d", rec.Code)
	}
}

// 封存后再次验证必须作为错误返回，而非 200 成功，否则调用方会误以为验证完成。
func TestVerifyAfterSealIsError(t *testing.T) {
	h := newHandler(t)
	app := h.app

	// 建运行 + 两架飞行器（互相安全，足以发布封存）。
	run := mustCreateRun(t, app, "seal-verify")
	base := int64(1_700_000_000_000)
	app.SetClock(func() int64 { return base })
	ctx := t.Context()

	a := mustRegisterAircraft(t, app, run.ID, "A", 0.5)
	b := mustRegisterAircraft(t, app, run.ID, "B", 0.5)
	t0, t1 := base+1000, base+5000
	mustIngest(t, app, run.ID, a.ID, service.IntentInput{Seq: 1, TStart: t0, TEnd: t1, PosX: 0, SigX: 0.5, SigY: 0.5, SigZ: 0.5})
	mustIngest(t, app, run.ID, b.ID, service.IntentInput{Seq: 1, TStart: t0, TEnd: t1, PosX: 100, SigX: 0.5, SigY: 0.5, SigZ: 0.5})

	res := mustVerify(t, app, run.ID)
	if res.Status != model.RunSafe {
		t.Fatalf("expected safe before seal, got %s", res.Status)
	}
	if err := app.PublishSnapshot(ctx, run.ID, res.SnapshotID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// 封存后再次验证：HTTP 层必须返回非 2xx（409），而不是 200。
	req := httptest.NewRequest(http.MethodPost, "/api/runs/"+run.ID+"/verify", nil)
	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)
	if rec.Code == 200 {
		t.Fatalf("verify after seal returned 200, want non-2xx; body=%s", rec.Body.String())
	}
	if rec.Code != 409 {
		t.Fatalf("verify after seal status %d, want 409", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["error"] == nil {
		t.Fatalf("missing error field in body=%s", rec.Body.String())
	}
}

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"task276-droneformation/internal/service"
	"task276-droneformation/internal/store"
)

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

// TestIngestDuplicateIntentSeqConflict 验证同一飞行器再次上报相同序号的意图段时，
// 接口层按领域冲突（409）返回而非掩盖为内部错误（500）。
func TestIngestDuplicateIntentSeqConflict(t *testing.T) {
	h := newHandler(t)
	mux := h.Router()

	// 建立 run 与一架飞行器。
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/runs", strings.NewReader(`{"name":"dup"}`)))
	if rec.Code != 201 {
		t.Fatalf("create run status %d body=%s", rec.Code, rec.Body.String())
	}
	var run map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &run)
	runID, _ := run["id"].(string)

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/runs/"+runID+"/aircraft", strings.NewReader(`{"callsign":"A","radius_m":0.5}`)))
	if rec.Code != 201 {
		t.Fatalf("register aircraft status %d body=%s", rec.Code, rec.Body.String())
	}
	var ac map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &ac)
	aid, _ := ac["id"].(string)

	intentBody := `{"seq":1,"t_start":1000,"t_end":5000,"pos_x":0,"vel_x":1,"sig_x":0.5,"sig_y":0.5,"sig_z":0.5}`
	endpoint := "/api/runs/" + runID + "/aircraft/" + aid + "/intents"

	// 首次上报：应成功。
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(intentBody)))
	if rec.Code != 201 {
		t.Fatalf("first ingest status %d body=%s", rec.Code, rec.Body.String())
	}

	// 再次上报相同序号：应作为领域冲突（409）沿错误链传到接口层，而非 500。
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(intentBody)))
	if rec.Code != 409 {
		t.Fatalf("duplicate seq want 409 conflict, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestBatchIngestDuplicateIntentSeqConflict 验证批量上报接口同样按冲突返回。
func TestBatchIngestDuplicateIntentSeqConflict(t *testing.T) {
	h := newHandler(t)
	mux := h.Router()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/runs", strings.NewReader(`{"name":"dup-batch"}`)))
	if rec.Code != 201 {
		t.Fatalf("create run status %d body=%s", rec.Code, rec.Body.String())
	}
	var run map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &run)
	runID, _ := run["id"].(string)

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/runs/"+runID+"/aircraft", strings.NewReader(`{"callsign":"A","radius_m":0.5}`)))
	if rec.Code != 201 {
		t.Fatalf("register aircraft status %d body=%s", rec.Code, rec.Body.String())
	}
	var ac map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &ac)
	aid, _ := ac["id"].(string)

	intent := `{"aircraft_id":"` + aid + `","seq":1,"t_start":1000,"t_end":5000,"pos_x":0,"vel_x":1,"sig_x":0.5,"sig_y":0.5,"sig_z":0.5}`
	batchBody := `{"items":[` + intent + `]}`
	endpoint := "/api/runs/" + runID + "/intents/batch"

	// 首次批量上报：应成功。
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(batchBody)))
	if rec.Code != 200 {
		t.Fatalf("first batch status %d body=%s", rec.Code, rec.Body.String())
	}

	// 再次上报相同序号：应作为领域冲突（409）返回。
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(batchBody)))
	if rec.Code != 409 {
		t.Fatalf("batch duplicate seq want 409 conflict, got %d body=%s", rec.Code, rec.Body.String())
	}
}

package httpapi

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) createRun(w http.ResponseWriter, r *http.Request) {
	var body struct{ Name string }
	_ = json.NewDecoder(r.Body).Decode(&body)
	run, err := h.app.CreateRun(r.Context(), body.Name)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 201, run)
}

func (h *Handler) listRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := h.app.ListRuns()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, runs)
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := h.app.GetRun(id)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, run)
}

func (h *Handler) updateConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		MinSeparationM float64 `json:"min_separation_m"`
		ConfidenceK    float64 `json:"confidence_k"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := h.app.UpdateConfig(r.Context(), id, body.MinSeparationM, body.ConfidenceK); err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	run, _ := h.app.GetRun(id)
	writeJSON(w, 200, run)
}

func (h *Handler) verifyRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.app.VerifyRun(r.Context(), id)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, res)
}

func (h *Handler) publishRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		SnapshotID int64 `json:"snapshot_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := h.app.PublishSnapshot(r.Context(), id, body.SnapshotID); err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	run, _ := h.app.GetRun(id)
	writeJSON(w, 200, run)
}

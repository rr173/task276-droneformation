package httpapi

import (
	"encoding/json"
	"net/http"

	"task276-droneformation/internal/service"
)

func (h *Handler) ingestIntent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	var b service.IntentInput
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, 400, err)
		return
	}
	it, err := h.app.IngestIntent(r.Context(), id, aid, b)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 201, it)
}

func (h *Handler) listIntents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	its, err := h.app.ListIntentsByAircraft(id, aid)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, its)
}

func (h *Handler) listRunIntents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	its, err := h.app.ListIntentsByRun(id)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, its)
}

func (h *Handler) batchIntents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Items []struct {
			AircraftID string `json:"aircraft_id"`
			service.IntentInput
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, err)
		return
	}
	items := make(map[string]service.IntentInput, len(body.Items))
	for _, it := range body.Items {
		items[it.AircraftID] = it.IntentInput
	}
	if err := h.app.BatchIngestIntent(r.Context(), id, items); err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"status": "ok", "count": len(items)})
}

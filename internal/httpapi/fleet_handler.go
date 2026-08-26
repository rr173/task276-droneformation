package httpapi

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) registerAircraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Callsign        string  `json:"callsign"`
		RadiusM         float64 `json:"radius_m"`
		HeightBaselineM float64 `json:"height_baseline_m"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, err)
		return
	}
	ac, err := h.app.RegisterAircraft(r.Context(), id, body.Callsign, body.RadiusM, body.HeightBaselineM)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 201, ac)
}

func (h *Handler) listAircraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	acs, err := h.app.ListAircraft(id)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, acs)
}

func (h *Handler) isolateAircraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	if err := h.app.IsolateAircraft(r.Context(), id, aid); err != nil {
		writeJSON(w, 200, map[string]string{"status": "isolated", "aircraft_id": aid})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "isolated", "aircraft_id": aid})
}

func (h *Handler) reinstateAircraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	if err := h.app.ReinstateAircraft(r.Context(), id, aid); err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "active", "aircraft_id": aid})
}

func (h *Handler) getCovariance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	c, err := h.app.GetCovariance(id, aid)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	if c == nil {
		writeJSON(w, 404, map[string]string{"error": "covariance not found"})
		return
	}
	writeJSON(w, 200, c)
}

func (h *Handler) putCovariance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	var body struct {
		SigX     float64 `json:"sig_x"`
		SigY     float64 `json:"sig_y"`
		SigZ     float64 `json:"sig_z"`
		SigRateX float64 `json:"sig_rate_x"`
		SigRateY float64 `json:"sig_rate_y"`
		SigRateZ float64 `json:"sig_rate_z"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, err)
		return
	}
	c, err := h.app.UpdateCovariance(r.Context(), id, aid, body.SigX, body.SigY, body.SigZ, body.SigRateX, body.SigRateY, body.SigRateZ)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, c)
}

package httpapi

import (
	"net/http"
	"strconv"
)

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]interface{}{
		"status":  "ok",
		"service": "droneformation",
	})
}

func (h *Handler) runStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stats, err := h.app.GetStats(id)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, stats)
}

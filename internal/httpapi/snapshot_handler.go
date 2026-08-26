package httpapi

import "net/http"

func (h *Handler) listSnapshots(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	snaps, err := h.app.ListSnapshots(id)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	writeJSON(w, 200, snaps)
}

func (h *Handler) getSnapshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sid := r.PathValue("sid")
	snapID, err := parseID(sid)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	snap, err := h.app.GetSnapshot(snapID)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	_ = id
	writeJSON(w, 200, snap)
}

package httpapi

import (
	"net/http"
	"strconv"

	"task276-droneformation/internal/model"
)

func (h *Handler) listRelations(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	snapID := int64(0)
	if v := r.URL.Query().Get("snapshot"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			snapID = n
		}
	}
	if snapID == 0 {
		if snaps, err := h.app.ListSnapshots(id); err == nil && len(snaps) > 0 {
			snapID = snaps[0].ID
		}
	}
	rels, err := h.app.ListRelations(id, snapID)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, rels)
}

func (h *Handler) getRelation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sid := r.PathValue("sid")
	rid, err := strconv.ParseInt(sid, 10, 64)
	if err != nil {
		writeErr(w, 400, model.ErrAircraftNotFound)
		return
	}
	rel, err := h.app.GetRelation(rid)
	if err != nil {
		writeErr(w, httpCode(err), err)
		return
	}
	_ = id
	writeJSON(w, 200, rel)
}

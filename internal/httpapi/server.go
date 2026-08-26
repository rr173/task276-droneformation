// Package httpapi 暴露基于 /api 前缀的 JSON HTTP 接口。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task276-droneformation/internal/model"
	"task276-droneformation/internal/service"
)

// Handler 持有应用对象并提供路由。
type Handler struct {
	app *service.App
}

// NewHandler 构造 HTTP 处理器。
func NewHandler(app *service.App) *Handler {
	return &Handler{app: app}
}

// Router 返回配置了全部端点的路由表。
func (h *Handler) Router() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("POST /api/runs", h.createRun)
	m.HandleFunc("GET /api/runs", h.listRuns)
	m.HandleFunc("GET /api/runs/{id}", h.getRun)
	m.HandleFunc("POST /api/runs/{id}/config", h.updateConfig)
	m.HandleFunc("POST /api/runs/{id}/verify", h.verifyRun)
	m.HandleFunc("POST /api/runs/{id}/publish", h.publishRun)
	m.HandleFunc("GET /api/runs/{id}/stats", h.runStats)

	m.HandleFunc("POST /api/runs/{id}/aircraft", h.registerAircraft)
	m.HandleFunc("GET /api/runs/{id}/aircraft", h.listAircraft)
	m.HandleFunc("POST /api/runs/{id}/aircraft/{aid}/isolate", h.isolateAircraft)
	m.HandleFunc("POST /api/runs/{id}/aircraft/{aid}/reinstate", h.reinstateAircraft)
	m.HandleFunc("GET /api/runs/{id}/aircraft/{aid}/covariance", h.getCovariance)
	m.HandleFunc("PUT /api/runs/{id}/aircraft/{aid}/covariance", h.putCovariance)

	m.HandleFunc("POST /api/runs/{id}/aircraft/{aid}/intents", h.ingestIntent)
	m.HandleFunc("GET /api/runs/{id}/aircraft/{aid}/intents", h.listIntents)
	m.HandleFunc("POST /api/runs/{id}/intents/batch", h.batchIntents)
	m.HandleFunc("GET /api/runs/{id}/intents", h.listRunIntents)

	m.HandleFunc("GET /api/runs/{id}/relations", h.listRelations)
	m.HandleFunc("GET /api/runs/{id}/relations/{rid}", h.getRelation)

	m.HandleFunc("GET /api/runs/{id}/snapshots", h.listSnapshots)
	m.HandleFunc("GET /api/runs/{id}/snapshots/{sid}", h.getSnapshot)

	m.HandleFunc("GET /api/health", h.health)
	return m
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]interface{}{"error": err.Error()})
}

// httpCode 将领域错误映射到合适的 HTTP 状态码。
func httpCode(err error) int {
	switch {
	case errors.Is(err, model.ErrRunNotFound), errors.Is(err, model.ErrAircraftNotFound):
		return 404
	case errors.Is(err, model.ErrRunSealed), errors.Is(err, model.ErrDuplicateSeq),
		errors.Is(err, model.ErrIntentInvalid), errors.Is(err, model.ErrCovarianceIllegal),
		errors.Is(err, model.ErrSnapshotNotDraft), errors.Is(err, model.ErrNoCommonWindow),
		errors.Is(err, model.ErrHeightMismatch):
		return 409
	default:
		return 500
	}
}

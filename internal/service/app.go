// Package service 编排领域包与存储，对上层（HTTP / 自检）暴露高级业务能力。
package service

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"task276-droneformation/internal/model"
	"task276-droneformation/internal/store"
)

// App 是编队避碰验证服务的核心应用对象。
type App struct {
	store       *store.Store
	clock       func() int64
	latestCache map[string]model.IntentSegment
}

// NewApp 构造应用对象，默认时钟为系统毫秒时间。
func NewApp(s *store.Store) *App {
	return &App{store: s, clock: func() int64 { return time.Now().UnixMilli() }, latestCache: map[string]model.IntentSegment{}}
}

// SetClock 覆盖时钟，主要用于自检的可复现性。
func (a *App) SetClock(fn func() int64) {
	a.clock = fn
}

func (a *App) now() int64 { return a.clock() }

var idSeq uint64

func uniqueID(prefix string) string {
	n := atomic.AddUint64(&idSeq, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, os.Getpid(), n)
}

// IntentInput 是飞行器意图段的上行入参。
type IntentInput struct {
	Seq                int64   `json:"seq"`
	TStart             int64   `json:"t_start"`
	TEnd               int64   `json:"t_end"`
	PosX               float64 `json:"pos_x"`
	PosY               float64 `json:"pos_y"`
	PosZ               float64 `json:"pos_z"`
	VelX               float64 `json:"vel_x"`
	VelY               float64 `json:"vel_y"`
	VelZ               float64 `json:"vel_z"`
	SigX               float64 `json:"sig_x"`
	SigY               float64 `json:"sig_y"`
	SigZ               float64 `json:"sig_z"`
	SigRateX           float64 `json:"sig_rate_x"`
	SigRateY           float64 `json:"sig_rate_y"`
	SigRateZ           float64 `json:"sig_rate_z"`
	RefHeightBaseline  float64 `json:"ref_height_baseline"`
}

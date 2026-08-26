// Package intent 负责意图段的时间窗统一与有效意图筛选。
package intent

import (
	"math"

	"task276-droneformation/internal/model"
)

// LatestActive 从同一飞行器的意图序列中挑选最新一条作为活跃意图。
// 返回 (意图, 是否活跃)。若序列为空或该飞行器已失联则返回 (最新意图, false)。
func LatestActive(segs []model.IntentSegment, now int64) (model.IntentSegment, bool) {
	if len(segs) == 0 {
		return model.IntentSegment{}, false
	}
	latest := segs[0]
	for _, s := range segs {
		if s.Seq > latest.Seq {
			latest = s
		}
	}
	// 失联判定：意图窗口结束已久，或距上次上报超过失联阈值。
	if now > latest.TEnd+model.LostTimeoutMs || now-latest.CreatedAt > model.LostTimeoutMs {
		return latest, false
	}
	return latest, true
}

// CommonWindow 返回多个活跃意图段分析窗口的交集 [start,end]；无交集返回 false。
func CommonWindow(intents []model.IntentSegment) (int64, int64, bool) {
	if len(intents) == 0 {
		return 0, 0, false
	}
	start := int64(math.MinInt64)
	end := int64(math.MaxInt64)
	for _, it := range intents {
		if it.TStart > start {
			start = it.TStart
		}
		if it.TEnd < end {
			end = it.TEnd
		}
	}
	if start >= end {
		return 0, 0, false
	}
	return start, end, true
}

// SampleStep 返回窗口 [t0,t1] 上的合理采样步长（毫秒），兼顾精度与性能。
func SampleStep(t0, t1 int64) int64 {
	d := t1 - t0
	step := d / 200
	if step < 50 {
		step = 50
	}
	if step > 1000 {
		step = 1000
	}
	return step
}

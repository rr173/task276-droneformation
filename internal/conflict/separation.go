// Package conflict 实现两两飞行器在未来时间窗内的最小有效间隔判定。
package conflict

import (
	"math"

	"task276-droneformation/internal/envelope"
	"task276-droneformation/internal/model"
)

// PairResult 是一对飞行器在一次验证中的结果。
type PairResult struct {
	AircraftA      string
	AircraftB      string
	MinEffDistance float64
	RequiredGap    float64
	WorstT         int64
	Status         string
}

// EvaluatePair 在 [t0,t1] 上按 step（毫秒）采样，计算两意图段间最小有效间隔。
// 有效间隔 = 中心距 − (可达半径A + 可达半径B)；若任一采样点 < 要求间隔 gap，则判定间隔不足。
func EvaluatePair(a, b model.IntentSegment, t0, t1, step int64, k, gap float64) PairResult {
	if step <= 0 {
		step = 1
	}
	minEff := math.MaxFloat64
	worstT := t0
	for t := t0; t <= t1; t += step {
		ca := envelope.CenterAt(a, t)
		cb := envelope.CenterAt(b, t)
		ra := envelope.RadiusAt(a, t, k)
		rb := envelope.RadiusAt(b, t, k)
		center := ca.Sub(cb).Norm()
		eff := center - (ra + rb)
		if eff < minEff {
			minEff = eff
			worstT = t
		}
	}
	status := model.RelationSafe
	if minEff < gap {
		status = model.RelationInsufficient
	}
	return PairResult{
		AircraftA:      a.AircraftID,
		AircraftB:      b.AircraftID,
		MinEffDistance: minEff,
		RequiredGap:    gap,
		WorstT:         worstT,
		Status:         status,
	}
}

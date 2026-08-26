// Package envelope 计算意图段在未来时刻的可达中心与可达包络半径。
// 定位不确定度随时间线性增长，包络以 kσ 为保守半径（椭球近似为球）。
package envelope

import "task276-droneformation/internal/model"

// CenterAt 返回意图段在绝对时刻 t（毫秒）的可达中心位置。
func CenterAt(it model.IntentSegment, t int64) model.Vec3 {
	dt := float64(t-it.TStart) / 1000.0 // 换算为秒
	if dt < 0 {
		dt = 0
	}
	return model.Vec3{
		X: it.PosX + it.VelX*dt,
		Y: it.PosY + it.VelY*dt,
		Z: it.PosZ + it.VelZ*dt,
	}
}

// SigmaAt 返回意图段在时刻 t 的各轴定位标准差。
func SigmaAt(it model.IntentSegment, t int64) model.Vec3 {
	dt := float64(t-it.TStart) / 1000.0
	if dt < 0 {
		dt = 0
	}
	return model.Vec3{
		X: it.SigX + it.SigRateX*dt,
		Y: it.SigY + it.SigRateY*dt,
		Z: it.SigZ + it.SigRateZ*dt,
	}
}

// RadiusAt 返回时刻 t 的保守可达包络半径（取各轴 kσ 最大值）。
func RadiusAt(it model.IntentSegment, t int64, k float64) float64 {
	s := SigmaAt(it, t)
	m := s.X
	if s.Y > m {
		m = s.Y
	}
	if s.Z > m {
		m = s.Z
	}
	if m < 0 {
		m = 0
	}
	return k * m
}

// Package model 定义无人机编队避碰意图一致性验证服务的领域实体、状态常量与几何工具。
package model

import "math"

// Vec3 是三维欧几里得向量，单位为米。
type Vec3 struct {
	X, Y, Z float64
}

// Add 返回向量加法结果。
func (v Vec3) Add(o Vec3) Vec3 { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }

// Sub 返回向量减法结果。
func (v Vec3) Sub(o Vec3) Vec3 { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }

// Scale 返回标量乘法结果。
func (v Vec3) Scale(s float64) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }

// Norm 返回向量的欧几里得长度。
func (v Vec3) Norm() float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }

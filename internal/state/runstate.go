// Package state 封装编队运行与飞行器状态机的合法性判定。
package state

import "task276-droneformation/internal/model"

// IsSealed 判定运行是否已封存。
func IsSealed(run model.FormationRun) bool {
	return run.Status == model.RunSealed
}

// CanVerify 判定运行当前是否允许触发验证（封存后禁止）。
func CanVerify(run model.FormationRun) bool {
	return run.Status != model.RunSealed
}

// NextVerificationStatus 由是否存在冲突关系推导运行的新状态。
func NextVerificationStatus(hasConflict bool) string {
	if hasConflict {
		return model.RunConflict
	}
	return model.RunSafe
}

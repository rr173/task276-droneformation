// Package snapshot 负责安全快照的结果汇总。
package snapshot

import "task276-droneformation/internal/model"

// Summary 是一次验证的关系汇总。
type Summary struct {
	ConflictCount int
	SafeCount     int
	Status        string
}

// BuildSummary 根据避碰关系列表统计冲突/安全数量并推导快照状态。
func BuildSummary(rels []model.AvoidanceRelation) Summary {
	s := Summary{}
	for _, r := range rels {
		switch r.Status {
		case model.RelationInsufficient:
			s.ConflictCount++
		case model.RelationSafe:
			s.SafeCount++
		}
	}
	if s.ConflictCount > 0 {
		s.Status = model.RunConflict
	} else {
		s.Status = model.RunSafe
	}
	return s
}

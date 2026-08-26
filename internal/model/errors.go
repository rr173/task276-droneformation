package model

import "errors"

// 领域错误，供上层按类型判定与映射 HTTP 状态码。
var (
	ErrRunNotFound      = errors.New("formation run not found")
	ErrRunSealed        = errors.New("formation run is sealed")
	ErrAircraftNotFound = errors.New("aircraft not found")
	ErrIntentInvalid    = errors.New("intent invalid: time or geometry constraints violated")
	ErrHeightMismatch   = errors.New("height baseline mismatch across intents")
	ErrNoCommonWindow   = errors.New("no common analysis window across active intents")
	ErrDuplicateSeq     = errors.New("duplicate intent sequence for aircraft")
	ErrCovarianceIllegal = errors.New("illegal positioning covariance")
	ErrSnapshotNotDraft = errors.New("snapshot is not a draft")
)

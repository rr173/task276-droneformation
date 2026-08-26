package model

// 编队运行状态机：接收中 → 待验证/存在冲突/安全 → 封存。
const (
	RunReceiving = "receiving"
	RunPending   = "pending_verification"
	RunConflict  = "conflict"
	RunSafe      = "safe"
	RunSealed    = "sealed"
)

// 飞行器状态。
const (
	AircraftActive   = "active"
	AircraftIsolated = "isolated"
)

// 意图段状态：原始 → 有效；过期/失联/排除为失效状态。
const (
	IntentRaw      = "raw"
	IntentValid    = "valid"
	IntentExpired  = "expired"
	IntentLost     = "lost"
	IntentExcluded = "excluded"
)

// 避碰关系状态。
const (
	RelationCandidate    = "candidate"
	RelationSafe         = "safe"
	RelationInsufficient = "insufficient"
	RelationConfirmed    = "confirmed"
)

// 安全快照状态。
const (
	SnapDraft      = "draft"
	SnapPublished  = "published"
	SnapSuperseded = "superseded"
)

// 默认参数与硬约束。
const (
	DefaultConfidenceK = 3.0   // 3σ 可达包络置信系数
	DefaultRadiusM     = 0.5   // 飞行器物理半径（米）
	DefaultMinSepM     = 2.0   // 编队最小间隔（米）
	LostTimeoutMs      = 10000 // 失联判定：超过该时长无新意图即判失联
)

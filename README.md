# task276-droneformation 无人机编队避碰意图一致性验证服务

面向飞行控制工程师的编队安全分析后端：在真实起飞前，验证各无人机已发布的未来轨迹意图
在统一时间窗内能否共同满足三维安全间隔，并将结论冻结为不可变安全快照。

## 核心概念

- **编队运行**：一次验证任务，状态机 `receiving → pending_verification/conflict/safe → sealed`。
- **飞行器**：编队成员，状态 `active ⇄ isolated`；隔离后退出验证，恢复后重新参与。
- **意图段**：带定位不确定度的未来轨迹，按 `(run_id, aircraft_id, seq)` 幂等入库。
- **定位协方差**：基准标准差与线性增长率，验证时叠加到可达包络。
- **避碰关系**：两两最小有效间隔判定，状态 `candidate → safe/insufficient`。
- **安全快照**：验证结果冻结件，状态 `draft → published → superseded`。

## 快速开始

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
go run ./cmd/droneformation --smoke-test

go run ./cmd/droneformation --addr :8080 --db task276-droneformation.db
```

## API 入口（前缀 /api）

- 运行：`POST /api/runs`、`GET /api/runs`、`GET /api/runs/{id}`、`POST /api/runs/{id}/config`、
  `POST /api/runs/{id}/verify`、`POST /api/runs/{id}/publish`、`GET /api/runs/{id}/stats`
- 飞行器：`POST /api/runs/{id}/aircraft`、`GET /api/runs/{id}/aircraft`、
  `POST /api/runs/{id}/aircraft/{aid}/isolate`、`POST /api/runs/{id}/aircraft/{aid}/reinstate`
- 定位不确定度：`GET /api/runs/{id}/aircraft/{aid}/covariance`、`PUT /api/runs/{id}/aircraft/{aid}/covariance`
- 意图：`POST /api/runs/{id}/aircraft/{aid}/intents`、`GET /api/runs/{id}/aircraft/{aid}/intents`、
  `POST /api/runs/{id}/intents/batch`、`GET /api/runs/{id}/intents`
- 避碰关系：`GET /api/runs/{id}/relations`、`GET /api/runs/{id}/relations/{rid}`
- 安全快照：`GET /api/runs/{id}/snapshots`、`GET /api/runs/{id}/snapshots/{sid}`
- 其它：`GET /api/health`

## 目录结构

```
cmd/droneformation/   入口（--addr/--db/--smoke-test）
internal/model/       实体、状态机与几何向量
internal/store/       SQLite 持久化（6 张表）
internal/envelope/    可达中心与不确定度包络
internal/intent/      时间窗统一与失联筛选
internal/conflict/    两两最小有效间隔判定
internal/state/       运行状态机
internal/snapshot/    快照汇总
internal/service/     业务编排
internal/httpapi/     HTTP 层（路由前缀 /api）
internal/smoke/       --smoke-test 自检
```

## 持久化与重启恢复

SQLite 单文件持久化。`--smoke-test` 建立真实实体、触发验证并发布快照后，关闭并重新打开
同一数据库，校验运行封存、快照发布、避碰关系与意图状态全部恢复。

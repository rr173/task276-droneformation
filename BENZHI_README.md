# BENZHI 评测说明

基于 Go 实现的无人机编队避碰意图一致性验证后端服务，一款后端服务，完成飞行器意图段接收、定位不确定度扩张下的可达包络计算、两两最小有效间隔判定与不可变安全快照封存。

## 启动

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/droneformation --addr :8080 --db task276-droneformation.db
```

## 自检（不启动长驻服务）

```bash
go run ./cmd/droneformation --smoke-test
```

`--smoke-test` 会真实创建编队运行与三架飞行器、注入意图段、触发验证、发布安全快照，关闭并重新打开数据库验证持久化与重启恢复，最后以 0 退出码结束。

## 构建门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
go run ./cmd/droneformation --smoke-test
```

## HTTP API（前缀 /api）

运行：`POST /api/runs`、`GET /api/runs`、`GET /api/runs/{id}`、`POST /api/runs/{id}/config`、`POST /api/runs/{id}/verify`、`POST /api/runs/{id}/publish`、`GET /api/runs/{id}/stats`
飞行器：`POST /api/runs/{id}/aircraft`、`GET /api/runs/{id}/aircraft`、`POST /api/runs/{id}/aircraft/{aid}/isolate`、`POST /api/runs/{id}/aircraft/{aid}/reinstate`
定位不确定度：`GET /api/runs/{id}/aircraft/{aid}/covariance`、`PUT /api/runs/{id}/aircraft/{aid}/covariance`
意图：`POST /api/runs/{id}/aircraft/{aid}/intents`、`GET /api/runs/{id}/aircraft/{aid}/intents`、`POST /api/runs/{id}/intents/batch`、`GET /api/runs/{id}/intents`
避碰关系：`GET /api/runs/{id}/relations`、`GET /api/runs/{id}/relations/{rid}`
安全快照：`GET /api/runs/{id}/snapshots`、`GET /api/runs/{id}/snapshots/{sid}`
自检：`GET /api/health`

## 持久化

SQLite（modernc.org/sqlite，CGO 无关）。建表：formation_runs、aircraft、intent_segments、aircraft_covariance、avoidance_relations、safety_snapshots。意图按 (run_id, aircraft_id, seq) 幂等；发布快照走 superseded 而非覆盖。

基于 Go 实现的地球化学年代测定空白校正 Web 项目，一款后端服务，对样品测次做空白扣除、按标准漂移恢复比值并传播不确定度以给出同位素年龄与异常判定。

# BENZHI 评测说明

## 1. 项目类型

地球化学年代测定空白校正后端服务（非 OA/工单/预约，非数据看板消费类应用）。提供 JSON 形态的 `/api` 接口，供实验人员以程序化方式导入测次、匹配空白与漂移、计算年龄并发布不可变年代版本。

## 2. 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/blankcorr --smoke-test
go run ./cmd/blankcorr --addr :8080 --db blankcorr.db
```

- `--addr`：HTTP 监听地址，默认 `:8080`
- `--db`：SQLite 数据库文件路径，默认 `blankcorr.db`
- `--smoke-test`：不常驻；跑完端到端场景后关闭并重新打开数据库，退出码 0 表示通过

## 3. 评测镜像

`Dockerfile` 与 `benzhi.Dockerfile` 内容完全一致；使用 Go 1.26.3 Bookworm builder 和 Alpine 3.20 runtime 的多阶段构建，产物为 `/app/blankcorr`。脚本第二个参数为目标平台。镜像不声明固定端口，服务监听地址由 `--addr` 指定。

```bash
./build_benzhi_docker.sh task248-blankcorr:amd64 linux/amd64
docker run --rm task248-blankcorr:amd64 --smoke-test

./build_benzhi_docker.sh task248-blankcorr:arm64 linux/arm64
docker run --rm task248-blankcorr:arm64 --smoke-test

docker run --rm -P task248-blankcorr:amd64 --addr :8080 --db ./app.db
```

## 4. 冒烟自测契约（--smoke-test）

创建临时数据库 → 写入测定批次与测次 → 初次匹配误用受污染空白导致年龄异常 → 排除该空白并重算得到合理年龄 → 确认、发布并封存版本 → 关闭并重新打开数据库，校验批次已封存、年龄结果存在且不再异常后退出 0。

## 5. 核心 API（`/api` 前缀）

- 批次：`POST /api/batches`、`GET /api/batches`、`GET /api/batches/{id}`
- 测次：`POST /api/batches/{id}/measurements`、`GET /api/batches/{id}/measurements`、`PATCH /api/measurements/{id}/status`、`POST /api/measurements/{id}/exclude`
- 校正：`POST /api/batches/{id}/match`、`GET /api/batches/{id}/relations`、`POST /api/relations/{id}/status`、`POST /api/batches/{id}/correct`、`POST /api/batches/{id}/recompute`、`GET /api/batches/{id}/results`
- 版本：`POST /api/batches/{id}/publish`、`GET /api/batches/{id}/versions`、`GET /api/versions/{id}`、`POST /api/versions/{id}/seal`
- 自检：`GET /api/batches/{id}/selfcheck`、`GET /api/stats`、`GET /api/health`

## 6. 环境与组件

- Go 1.26.3（GOTOOLCHAIN=local，CGO_ENABLED=0）
- SQLite 3.46.1（modernc.org/sqlite v1.52.0，CGO 无关）

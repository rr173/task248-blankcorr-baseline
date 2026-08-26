# task248-blankcorr · 地球化学年代测定空白校正服务

面向地球化学实验人员，判断一个异常同位素年龄究竟是样品真实差异，还是空白选择不当或仪器漂移所致。服务导入样品/空白/标准三类测次，按时间窗口把样品匹配到最近的空白与批次级仪器漂移模型，执行空白扣除与漂移恢复，线性传播不确定度后得到年龄及 2σ 区间，并据批次预期区间标记异常；支持排除受污染空白后重算，并将结果发布、封存为不可变版本。

## 业务闭环

1. 登记测定批次（含衰变系统常数 λ、初始比值 R0、预期年龄区间）。
2. 导入样品、空白、标准测次（按指纹幂等）。
3. 匹配：每个样品绑定最近窗口内合格空白 + 批次级漂移模型在样品时刻的恢复因子。
4. 校正与定年：R = (样品 − 空白) × 漂移因子；t = (1/λ)·ln(R0/R)，并传播不确定度。
5. 异常判定：2σ 年龄区间是否落在预期带内；否则标记需复核。
6. 排除受污染空白 → 重匹配 → 重算；确认校正关系 → 发布版本 → 封存（冻结批次）。

## 状态机

- 测定批次：`receiving → pending → needs_review → published → sealed`
- 测次：`raw → usable → contaminated / expired / excluded`（污染/过期/排除为终态）
- 校正关系：`candidate → valid / conflict → confirmed`
- 年代版本：`draft → published → sealed / superseded`

## 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/blankcorr --smoke-test
go run ./cmd/blankcorr --addr :8080 --db blankcorr.db
```

## 目录结构

```
cmd/blankcorr/main.go        入口（--smoke-test / HTTP 服务）
internal/model/              实体、状态机、校验
internal/store/              SQLite 持久化（7 张表）
internal/measure/            测次导入与状态变更
internal/match/              时间窗口匹配 + 漂移绑定
internal/correct/            漂移拟合、空白+漂移校正、年龄不确定度传播
internal/review/             排除受污染测次
internal/version/            版本发布与封存
internal/service/            编排层（HTTP 与冒烟共用）
internal/httpapi/            /api HTTP 层
```

## 技术约束

- Go 1.26.3，`CGO_ENABLED=0`，SQLite 走 `modernc.org/sqlite`（离线可构建）。
- 所有时间戳以整数毫秒存储，便于时间窗口计算与排序。

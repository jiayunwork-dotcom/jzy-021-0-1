# CPM 计划作业服务

关键线路法（CPM）进度计算内核：提交一张工序网络，正推最早开始/最早完成，
反推最晚开始/最晚完成，标出总时差为 0 的关键线路；给三点历时与目标日期时，
按 PERT 估计完工把握。只经 HTTP 对外提供**计算**与**取回**，计划作业落本地
文件库，不另起数据库进程。

## 模块划分

| 文件 | 职责 |
|---|---|
| `validate.go` | 入参检查：缺项、重复标识、未知紧前、自环、非正历时、三点顺序 |
| `topo.go` | 建图（含虚拟源点/汇点）与 Kahn 拓扑排序，排不完即报有向环 |
| `passes.go` | 正推（ES/EF）与反推（LS/LF），自由时差 |
| `critical.go` | 关键通路枚举、PERT 方差汇总、完工概率正态近似 |
| `compute.go` | 编排：校验 → 排序 → 正反推 → 时差 → 关键通路 → PERT |
| `model.go` / `errors.go` | 数据模型 / 带类型的错误 |
| `store.go` | 本地文件作业库（每份作业一个 JSON 文件，原子写入） |
| `server.go` / `main.go` | HTTP 路由（Go 1.22 模式路由）/ 入口 |
| `demo.go` | 内置土建示范网络 |

## 计算约定

- 无紧前的工序从虚拟源点出发，无紧后的工序进入虚拟汇点；源汇点只为闭合
  网络，**不出现在**关键线路名单与工序输出里。
- 正推：`EF = ES + 历时`，汇合点 ES 取各紧前 EF 的最大值，汇点 EF 即项目工期。
- 反推：`LS = LF - 历时`，LF 取各紧后 LS 的最小值。
- 总时差 `TF = LS - ES = LF - EF`，两式同时成立（计算中有内部断言）；整数历时
  时 TF 恰为 0 或正整数。自由时差 `FF = min(紧后 ES) - EF`，不大于 TF。
- 三点历时：期望 `(a+4m+b)/6`，方差 `((b-a)/6)²`；必须满足 `a ≤ m ≤ b`。
- 项目工期方差沿**一条选定的关键通路**求和；多条关键通路取方差最大者，
  并在 `pert.selected_path` 标明。非关键工序的三点宽度不影响项目方差。
- 完工概率：`P(T≤d) = Φ((d-μ)/σ)`（公式随结果返回）；目标早于均值时概率 < 0.5。
- 只给确定历时不给三点时，`pert` 整体留空，不编造方差与完工概率。
- 零历时「里程碑」不是合法工序，历时必须为正。

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/api/v1/jobs` | 提交网络，计算并落一份作业，返回完整结果（201） |
| `GET` | `/api/v1/jobs/{id}` | 按作业编号取回（不存在返回 404） |
| `GET` | `/api/v1/demo` | 内置土建示范网络定义 |
| `GET` | `/healthz` | 健康检查 |

提交体：

```json
{
  "name": "可选名称",
  "target_duration": 25,
  "activities": [
    {"id": "A", "duration": 3},
    {"id": "B", "three_point": {"optimistic": 2, "most_likely": 4, "pessimistic": 12},
     "predecessors": ["A"]}
  ]
}
```

每道工序给 `duration`（确定天数）或 `three_point`（三点）之一；
`target_duration` 可选，仅当网络含三点历时的时候参与完工概率计算。

错误响应带类型：`{"error": {"type": "cycle", "message": "..."}}`，类型包括
`missing_field`、`duplicate_id`、`unknown_predecessor`、`self_loop`、`cycle`、
`invalid_duration`、`invalid_three_point`、`bad_request`、`not_found`。

## 示范网络手算核对

`GET /api/v1/demo` 的小型土建工程（两对并行工序 B‖C、F‖G）：

| 工序 | 内容 | 历时 | 紧前 | ES | EF | LS | LF | TF | FF | 关键 |
|---|---|---|---|---|---|---|---|---|---|---|
| A | 场地平整 | 3 | — | 0 | 3 | 0 | 3 | 0 | 0 | ✓ |
| B | 基础开挖 | 4 | A | 3 | 7 | 3 | 7 | 0 | 0 | ✓ |
| C | 材料进场 | 2 | A | 3 | 5 | 5 | 7 | 2 | 2 | |
| D | 基础浇筑 | 5 | B,C | 7 | 12 | 7 | 12 | 0 | 0 | ✓ |
| E | 主体结构 | 6 | D | 12 | 18 | 12 | 18 | 0 | 0 | ✓ |
| F | 屋面工程 | 3 | E | 18 | 21 | 19 | 22 | 1 | 1 | |
| G | 装饰装修 | 4 | E | 18 | 22 | 18 | 22 | 0 | 0 | ✓ |
| H | 竣工验收 | 2 | F,G | 22 | 24 | 22 | 24 | 0 | 0 | ✓ |

关键线路 **A→B→D→E→G→H**，工期 24。汇合点 D 的 ES 取 max(EF_B, EF_C)=7，
H 的 ES 取 max(EF_F, EF_G)=22；较晚完工仍有时差的支（C、F）不关键。

## 运行

```bash
# 本地（Go 1.22）
go test ./...
go build -o cpmserver . && ./cpmserver        # 监听 :8080，作业库 ./data

# Docker（单容器，golang:1.22 构建）
docker build -t cpm .
docker run -p 8080:8080 -v cpm-data:/data cpm
# 或 docker compose up --build
```

环境变量：`CPM_ADDR`（默认 `:8080`）、`CPM_DATA_DIR`（默认 `./data`，
容器内 `/data`）。

```bash
curl -X POST localhost:8080/api/v1/jobs -d @network.json
curl localhost:8080/api/v1/jobs/job-000001
```

## 测试

`go test ./...` 以断言锁定：时差两式相等、自由时差不大于总时差、整数历时
总时差为整数、关键通路贯穿源到汇且只含调用方标识、多条关键通路全部列出、
有向环/自环/未知紧前/重复标识/非正历时/三点颠倒均被拒、非关键边改方差项目
方差不变、关键边改方差项目方差必变、目标早于均值概率小于一半、纯确定历时
不产出方差与概率、并行计算的多份作业关键通路互不渗透、文件库重开编号不回退。

# AGENTS.md - 开发规则

> 本文件是**常驻规则**，只放"每次改代码都必须知道"的约束，目标 100 行级别。
> 架构原理与已知偏离 → [`docs/architecture.md`](docs/architecture.md)；安全模型 → [`docs/security.md`](docs/security.md)；领域词 → [`docs/glossary.md`](docs/glossary.md)。
> 进入某个包工作时，该目录的 `AGENTS.md` 会自动加载（`frontend/`、`parser/`、`proxy/`、`manager/`、`store/`、`sql/`），那里写该包的做法。
> 与本文件的红线冲突时以本文件为准；代码现状与规则不符时，**新代码按规则写**，偏离记入 `docs/architecture.md` §4，不要顺手大重构。

## 0. 项目一句话

`aiapi` 是轻量级大模型 API 反向代理：多协议接入 → 按配置路由到上游 Provider → 透传响应（含 SSE）→ 记录用量与请求日志 → 提供管理接口。

已实现协议：`openai`、`anthropic`、`openai-responses`。**`gemini` 未实现**（`parser.GetParser` 返回 `nil`，补它还需处理"模型名在 URL path"，不只是加解析器）。

## 1. 速查

| 目的 | 命令 |
|------|------|
| 安装依赖 | `make install`（`go mod tidy`） |
| 跑测试 | `make test`（等价 `go test ./...`） |
| 全量检查 | `make check`（格式 + vet + 测试，含所有门禁） |
| 构建 | `make build` |
| 全量构建（含前端） | `make build-all` |
| 前端调试 | `make dev-ui`（3000，代理到 Go） |
| 本地运行 | `make run`（8887） |
| 格式化 | `make fmt` |

- **工具链**：Go 版本以 `go.mod` 为准（当前 1.25.4）；前端 Vue 3 + Vite，Node 需 18 / 20 / 22+（Vite 6 要求）。
- **仓库没有 CI、没有 linter**，但部分红线由 `go test` 里的门禁机械保证：架构依赖与权限种子（`arch_test.go`）、文档链接/锚点/体积（`docs_test.go`）、schema 版本一致性（`store/schema_version_test.go`）。其余靠自觉，改完必须本地 `make check`。
- **提交信息**：Conventional Commits，`<type>(<scope>): <中文描述>`，破坏性变更加 `!`（如 `feat(proxy)!:`）；type 用 feat / fix / refactor / docs / style / chore / perf。
- **禁改与生成物**：根目录 `aiapi` 二进制、`frontend/dist/assets` 是构建物；`frontend/dist/index.html` 被 git 跟踪且是 `go:embed all:frontend/dist` 的必需文件，**不可删除**。临时草稿放 `temp/`（已忽略）。
- **文档分工**：规则 → 本文件 + `docs/`；用法 → `README.md`；变更历史 → `CHANGELOG.md`；已识别未实施的优化 → `TODO.md`。

## 2. 目录地图（改哪里）

| 层次 | 职责 | 位置 | 越界示例 |
|------|------|------|----------|
| 入口 | 参数解析、日志/DB 初始化、启动 | `main.go` | — |
| 静态资源 | 前端 `embed` 与 SPA fallback | `frontend.go` | 在它之后注册新路由（会被 SPA 吞掉） |
| 框架适配 | 建路由组 + 全局中间件 | `framework/echo.go` | 往里写业务逻辑 |
| 路由适配 | 注册路由、参数提取、响应写入 | `proxy/router/`、`manager/router/` | 在 handler 里读 `c.Param` 以外的框架细节 |
| 编排 | 组装并执行 Pipeline | `proxy/direct.go`、`proxy/pipeline.go` | 在 direct.go 实现具体业务 |
| 代理处理 | 单一职责 handler | `proxy/handler/*.go` | 自建业务规则（应走 `service/`） |
| 协议解析 | 各厂商请求/响应解析、Key 提取 | `parser/*.go`、`parser/util/` | 操作数据库、写响应 |
| 后台接口 | HTTP 适配：校验、调 service、组装响应 | `manager/handler/*.go` | 写跨表事务 |
| 后台中间件 | 鉴权、权限判定、登录态注入 | `manager/middleware/*.go` | 写复杂业务 |
| 后台基础设施 | 分页类型、响应封装、业务码、`base.Wrap`、密钥 | `manager/base/*.go` | — |
| 业务服务 | 复用逻辑、多表事务、跨 Store 组装、业务判断 | `service/*.go` | 依赖 echo |
| 数据持久 | 单表 SQL、模型、QueryBuilder、分页拦截 | `store/`、`store/base/`、`store/driver/` | 跨表写、业务判断 |
| 通用工具 | 跨包常量、数据目录路径、密钥加载、日志 | `constant/`、`log/` | 业务常量堆进 `constant` 变上帝包 |

关键入口（先读这 5 个）：`main.go`、`framework/echo.go`、`proxy/direct.go`、`parser/interface.go`、`manager/router/router.go`。

## 3. 红线

**流程**

- **RED-01** 改完必须 `make check` + `go build` 通过；提交信息按 §1 规范，不写 `update xxx` 这类自由格式。
- **RED-02** 规则/架构变化同步本文件与 `docs/`，用户可见变化同步 `README.md`，两者都在 `CHANGELOG.md` 记录；未实施的优化写 `TODO.md`。

**分层**

- **RED-03** handler 失败时只设 `ctx.Err` + `ctx.Code`，由 Pipeline 统一输出错误；禁止在 handler 里 `c.JSON` / `c.String` 返回错误。
- **RED-04** 禁止在 handler 中检查 `ctx.Err` 决定是否执行；不可中断的收尾操作注册为 `AddFinally`。
- **RED-05** 事务体只能写在 `service/`；`store/` 不写跨表写操作、不写业务判断、不放纯函数业务工具。
- **RED-06** 依赖方向单向：`store/` 不 import echo / manager / parser；`parser/` 不 import store；`framework/echo.go` 不写业务。
- **RED-07** 可复用逻辑不重复实现：跨 handler 复用、跨表组装、业务不变量一律走 `service/`。
- **RED-08** 错误码编号唯一；manager 侧**不新增业务码**，用 `base.ErrBadReq` / `ErrNotFound`（中文消息）/ `ErrInternal`。proxy 对外错误消息英文，manager 中文。

**数据与安全**

- **RED-09** 金额/计数类"读后算再写"禁止"先 SELECT 再 UPDATE 写回"，也禁止 `UPDATE ... RETURNING` / `SELECT ... FOR UPDATE`（写法见 [`architecture.md` §6](docs/architecture.md)）。
- **RED-10** `usage_records` 仅在 `status_code < 300` 时记录；转发链路无论成败都记 `request_logs`，元数据端点（`v1/models`）不记。
- **RED-11** 日志、错误信息、响应中禁止出现完整 API Key；保存请求头前必须脱敏。
- **RED-12** 新增非超管 manager 接口必须同步 `sql/init-data.sql` 的 `role_permission`；新增页面还要同步 `menus` + `role_menus`，否则 403 或页面不可达。

**运行时不变量**（详见 [`architecture.md` §10](docs/architecture.md)）

- **RED-13** 不得给 HTTP Server 设 `WriteTimeout`、不得给上游 client 设总超时——长 SSE 会被误杀。
- **RED-14** 应用不自动建库/迁移：DB 文件须先手工执行 `sql/sqlite.sql`（+ `sql/init-data.sql`）。
- **RED-15** 转发不得跟随上游 3xx（`CheckRedirect` 返回 `http.ErrUseLastResponse`）：标准库只剥离 `Authorization` 等四个头，`x-api-key` 这类上游凭证会被送给重定向目标。
- **RED-16** 改 schema 必须同步三处：`sql/sqlite.sql`（含 `schema_meta` 版本行）、`constant.SchemaVersion`、`sql/migrations/` 台账；版本只能递增。

## 4. 边界判定式

| 场景 | 结论 |
|------|------|
| 单表 `Get/Select/Exec`、只读 JOIN/聚合 | 可留在 `store/` |
| 一次写多张表 / 事务体 | 必须 `service/` |
| 业务不变量、复用逻辑、纯函数业务工具 | 必须 `service/` |
| manager handler 里的单表读写 | 可直接调 `store`（现状主流） |
| manager handler 里的跨表校验、编排 | 移入 `service/` |
| proxy handler（鉴权、计费） | 调 `service/`，不自建业务逻辑 |
| 每个 Store 的命名空间入口 | 写在自己的 Store 文件内，不堆到 `store/base.go` |

## 5. 完成前自检

- [ ] `make check` 通过（含架构门禁、权限种子、文档链接与预算、schema 版本一致性）
- [ ] `go build` 通过（前端改动还要 `make build-all`）
- [ ] 新增 handler 优先覆盖失败路径，协议改动覆盖流式与非流式
- [ ] 数据层测试用内存 SQLite 并先调 `store.Init(db)`，不连真实数据目录
- [ ] 归属层次正确（对照 §2、§4），没有触碰 §3 红线
- [ ] 文档已同步：`docs/` / `README.md` / `CHANGELOG.md` / `TODO.md` 按 RED-02 判定
- [ ] 没有手改 `aiapi` 二进制、`frontend/dist/assets`

## 6. 什么时候读哪份文档

常驻规则在本文件；**进入某个包工作时该包的 `AGENTS.md` 会自动加载**，做法写在那里。

| 要做的事 | 读 |
|----------|-----|
| 理解分层、边界、事务、并发语义 | [`docs/architecture.md`](docs/architecture.md) |
| 查领域词（provider/format/pricing_snapshot/schema version…） | [`docs/glossary.md`](docs/glossary.md) |
| 改认证、鉴权、密钥、脱敏 | [`docs/security.md`](docs/security.md) |
| 想知道某个设计为什么这么做 | [`docs/decisions/`](docs/decisions/README.md) |
| 排查"这类 bug 为什么没被拦住" | [`docs/postmortem/`](docs/postmortem/README.md) |
| 加数据表 / 改 schema / 写迁移 | [`sql/AGENTS.md`](sql/AGENTS.md)、[`sql/migrations/README.md`](sql/migrations/README.md) |
| 文档该写在哪、写多长 | [`docs/AGENTS.md`](docs/AGENTS.md) |

# 架构与边界

> 按需阅读：改动跨层逻辑、新增模块、调整事务/并发语义前读本文件。
> 每次都要遵守的硬约束在 [`AGENTS.md`](../AGENTS.md)，各包的做法见该目录的 `AGENTS.md`（进入目录工作时自动加载）。
> 本文件描述**规则与理由**；代码是实现的唯一事实来源，发现两者不符时以规则为准并在 CHANGELOG 记录。

## 1. 项目定位与协议现状

`aiapi` 是轻量级 AI 大模型 API 反向代理：接收多协议客户端请求 → 按配置路由到上游 Provider → 透传响应（含 SSE）→ 记录用量与日志 → 提供管理接口。

**当前已实现的协议**（以 `parser.GetParser` 为准）：

| format | 状态 | 说明 |
|--------|------|------|
| `openai` | ✅ | Chat Completions |
| `anthropic` | ✅ | Messages API |
| `openai-responses` | ✅ | Responses API |
| `gemini` | ❌ **未实现** | `parser.GetParser` 返回 `nil`（`parser/interface.go`），请求无法处理；补齐还需解决"模型名在 URL path"的改写，不只是加一个解析器 |

新增协议见 [`../parser/AGENTS.md`](../parser/AGENTS.md)。

## 2. 分层与依赖方向

| 层次 | 职责 | 位置 |
|------|------|------|
| 入口层 | 参数解析、日志/数据库初始化、启动服务 | `main.go` |
| 静态资源层 | 前端 SPA 的 `go:embed` 与 fallback 路由 | `frontend.go` |
| 框架适配层 | 创建路由组与全局中间件；各业务在自己的 router 文件注册路由、做请求参数提取与响应写入 | `framework/echo.go`、`proxy/router/`、`manager/router/` |
| 业务编排层（proxy） | 组装并执行请求处理 Pipeline | `proxy/direct.go`、`proxy/pipeline.go` |
| 业务处理层（proxy） | 单一职责的 handler | `proxy/handler/*.go` |
| 协议解析层 | 各厂商请求/响应解析、Key 提取、模型名读写 | `parser/*.go`、`parser/util/` |
| 后台管理层 | 后端给前端的接口入口，只做 HTTP 适配 | `manager/handler/*.go` |
| 后台中间件层 | 登录态校验、接口级权限判定、登录态注入 | `manager/middleware/*.go` |
| 后台基础设施层 | 分页类型、响应封装、业务码、`base.Wrap`、密钥加载、登录态 context | `manager/base/*.go` |
| 业务服务层 | 跨 handler 复用逻辑、多表事务编排、跨 Store 组装、业务判断，不依赖 echo | `service/*.go` |
| 数据持久层 | 单表读写、模型定义、QueryBuilder、分页拦截 | `store/*.go`、`store/base/`、`store/driver/` |
| 通用工具层 | 跨包共享常量、数据目录路径、密钥加载、日志格式化 | `constant/`、`log/` |

**依赖方向只能自上而下**，反向依赖一律禁止：

```
main.go / frontend.go
  └─ framework ─┬─ proxy/router ── proxy (direct/pipeline/handler)
                └─ manager/router ─┬─ manager/handler ─┬─ service ── store ── store/base ── store/driver
                                   └─ manager/middleware┘     │
                                                             └─ parser（协议解析）
constant / log：最底层，谁都可以依赖；它们不反向依赖任何业务包
```

明确禁止的三条（前两条由 `gate_arch_test.go` 机械校验）：

- `store/` 不得 import echo / manager / parser（纯 SQL 层）
- `parser/` 不得 import store（协议层不碰数据库）
- `framework/echo.go` 不得写业务逻辑（只建路由组、挂全局中间件）

## 3. 关键边界

### framework / router

`framework/echo.go` 只做三件事：`Recover` + `BodyLimit` 全局中间件、创建 `/proxy` 与 `/manager` 两个路由组、委派各自的 `Register`。路由组根路径在这里直写，业务路由必须注册在业务自己的 router 文件里。

`frontend.go` 的 SPA fallback（`GET /*`）必须**最后**注册：任何新增的非 `/proxy`、`/manager` 前缀 GET 路由若排在它之后，会被 SPA 吞掉并返回 HTML。

### proxy

- `proxy/direct.go` 只组装 Pipeline，不实现业务（见 [`../proxy/AGENTS.md`](../proxy/AGENTS.md)）。
- `proxy/router/` 负责把 echo 上下文适配成 `types.ProxyRequest` 并区分入口；**入口区分靠路由注册，不在运行时按 path 判定**（echo 静态段优先于通配符 `*`：`GET /:provider/:format/v1/models` → `HandleModels`，其余 → `Handle`）。
- `Forward` 只负责 HTTP 转发与响应透传：发起上游请求、复制安全响应头、写回客户端、缓存原始响应；不做协议解析、用量统计或计费。响应完成后由独立 handler 消费缓存数据。
- **转发只使用 `providers.config.headers` 作为上游请求头**，客户端请求头不转发（含 `Content-Type`）；客户端请求头仅用于提取 Key 与写日志。

### parser

`parser/` 只做协议相关的解析与提取：不操作数据库、不写响应、不依赖 store。协议无关的纯工具放 `parser/util/`（无状态纯函数，且不反向依赖 `parser`）。协议相关的响应序列化（如模型列表）也归 `parser`：放不进 `Parser` 主接口的能力用可选接口 + 类型断言（参考 `ModelsFormatter` / `FormatModelList`），由业务层负责映射成 parser 中立结构。

### manager

`manager/handler/` 是后端给前端的唯一接口入口，只做 HTTP 适配（参数校验、调 service、组装响应）。`manager/middleware/` 只做鉴权与登录态注入。业务语义下沉 `service/`。

### service

`service/` 是 manager 与 proxy **共用**的业务层，承载：跨 handler 复用逻辑、涉及多表或带业务语义的事务编排、跨 Store 组装、业务判断。proxy 侧没有 manager 入口的场景（鉴权、计费）同样调 `service/`，不在 `proxy/handler` 自建业务逻辑。

### store

`store/` 是纯 SQL 包装层。判定式（新代码按此判断，见 §4 已知偏离）：

| 场景 | 允许位置 |
|------|----------|
| 单表 `Get/Select/Exec` 读写 | `store/` |
| 只读 JOIN / 聚合查询 | `store/`（现状如此，允许） |
| 一次写多张表 | `service/`（事务，`store.C().T(fn)` 包裹） |
| 事务体（组合多个 Store 调用 + 业务判断 + 回滚语义） | `service/`，禁止写在 `store/` |
| 可复用的业务不变量 / 纯函数业务工具（哈希、展示串、分组维度→SQL 表达式映射） | `service/` |
| 分组维度→排序方向这类业务判断 | `service/` |

其他约定：

- 按表/领域拆 Store，不同表的操作不混在同一 Store（如 `ModelAccessStore` 操作 `api_keys.model_policy` + `apikey_model_access`，独立于操作 `models` 的 `ModelStore`）。
- 每个 Store 的命名空间入口写在自己文件内（`func (s *Session) Xxx() *XxxStore`），不集中到 `store/base.go`。
- store 方法默认在调用方传入的 `*Session` 上执行，自身不感知是否在事务里：`Session.T(fn)` 包裹则进事务，`store.C()` 则非事务。
- `IN` 查询直接写 `IN (:ids)` 传 slice，QueryBuilder 内置 `sqlx.In` 自动展开，不手拼占位符。
- 不依赖 `UPDATE ... RETURNING` / `SELECT ... FOR UPDATE`（跨库不通用，见 §6）。
- 迁移不在应用内自动执行：DDL 以 `sql/sqlite.sql` 为准，需外部计算的由人工/脚本完成。

### constant / log

- 跨业务复用的基础设施常量（环境变量名、密钥长度下限、数据目录路径）→ 根目录 `constant/constant.go`；业务域自用常量（manager 会话 TTL、cookie 名）→ 留在本业务包（如 `manager/base/constant.go`）。不要堆进 constant 形成上帝包。
- 数据目录下的文件路径由 `constant` 的路径方法统一提供（`DBFilePath`/`LogFilePath`/`JWTKeyFilePath`），业务代码不自行拼接。
- 密钥加载统一走 `constant.LoadSecret`（env > 密钥文件 > 自动生成），业务层只做编排（参考 `manager/base/secret.go`）。
- 应用日志统一输出到 stdout 与 `constant.LogFilePath()`；初始化日志后 Echo 的 `e.Logger`、`e.StdLogger` 与 HTTP Server 错误日志复用同一个 writer，避免框架日志只出现在控制台。

### 命名与文件组织

- 业务包与文件名用小写 + 下划线（如 `parse_request.go`、`auth_key.go`），不用驼峰。
- 测试文件与源码同名，加 `_test.go` 后缀；测试与源码**同包**（见 §11）。
- 业务目录不混入框架适配代码：路由注册、参数提取、响应写入放 `proxy/router/`、`manager/router/`。

## 4. 已知偏离（文档规则 vs 代码现状）

新代码按规则写；存量按下列口径理解，不要顺手大重构（要收敛请单独立项并写 CHANGELOG）：

| 位置 | 现状 | 规则 |
|------|------|------|
| `store/usage.go` | 仍在做 `groupBy/mode → labelExpr/JOIN` 的业务映射 | 这类业务判断应下沉 `service/` |
| `proxy/handler/auth_model.go`、`budget.go` | 直接 `store.C()` 取数并内联额度规则 | 业务不变量应走 `service/`（同层 `auth_key.go`、`models.go`、`record.go` 已走 service） |
| `manager/handler/apikey.go` | `checkBudgetWithinUserLimit`（跨表不变量）、`createApiKeyRetry`（生成/哈希/加密/重试编排）留在 handler；self/admin 为两份近似实现 | 跨表不变量与编排应下沉 `service/`；self/admin 应合并为通用函数 |
| `store/base/page.go` | `PageResult[T]` 无引用（实际用的是 `manager/base.PageResult`）；`ClearPage` 无调用者 | `store` 只暴露 `PageContext`；分页结果类型用 `manager/base` |
| `manager/handler` | 直接 `store.C()` 约 74 处、调 `service.` 约 45 处 | 单表读写可直接调 store（这是允许的）；跨表/可复用逻辑必须走 service |

## 5. 事务边界

- `Session.T(fn)` 只提供事务执行入口；**事务体写在 `service/`**。判定标准：fn 内组合多个 Store 调用、含业务判断、或需要失败回滚语义 → 属于 `service/`。
- Session 保存 `tx`/`page` 状态字段，不用 `context.Value` 传业务参数（tx、分页）；context 只承载"请求级跨边界数据"（如 manager 登录态）。`store.C()` 无参；`Session.T(fn)` 在当前 Session 上切换 tx，嵌套调用复用当前 tx 不开新事务，保证原子性。
- 一个事务只编排「同一业务动作」需要的多张表，不要把无关写操作塞进同一事务。
- 参考实现：`service/charge.go`（充值：原子加余额 → 同事务读回 → 写流水）。

## 6. 并发安全（金额 / 计数类写操作）

涉及余额、计数等「读后算再写」的场景，必须保证读到的值是本次写入前的稳定值。

**适用判定**：仅针对"新值依赖当前值"的写操作（增量、扣减、扣费）。管理员直接设定绝对值（如 `UpdateProfile` 写 `budget=:budget`）不属于该模式，改它反而会引入错误。

通用做法（跨 SQLite / MySQL / PostgreSQL）：

1. 先 `UPDATE budget = budget + :amount`（同一行的 UPDATE 加行锁串行执行，金额不丢更新）
2. 再在同事务内 `SELECT budget`（事务内自己的修改对后续语句立即可见，拿到步骤 1 之后的新值）
3. 行锁持续到 COMMIT，期间其他事务的 UPDATE 阻塞，故步骤 2 读到的是稳定值
4. 需要记 before/after 流水时用 `before = after - amount` 反推

禁止的反模式：

- 先 `SELECT budget` 再 `UPDATE budget = :newValue`（读后写，快照读拿旧值 → 丢更新）
- 依赖 `UPDATE ... RETURNING`（MySQL 8 不支持）
- 依赖 `SELECT ... FOR UPDATE`（SQLite 不支持该语法）
- "先 SUM 校验再写 budget"的 check-then-write（超出上限的判定必须与写入在同一事务内完成，或改为写入后校验回滚）

参考实现：`store/charge.go`（原子增减）+ `service/charge.go`（事务内读回并写流水）、`service/billing.go`（扣费）。

**已知敞口**：`BudgetCheck` 先放行后扣费，并发下预算可能变为负值——属当前设计接受的现状，动手改造前需先定业务容忍度（接受负值 + 告警，或引入预扣机制）。

## 7. 注释风格

- `store/` 方法注释只说明「这个方法做什么」，不描述事务用法、调用时机、并发语义——这些属于 service/handler 的编排逻辑，写在 store 里是越界且误导。
- `service/`、`handler/` 的复杂逻辑（事务编排、并发安全原理、非显然的边界条件）才需要详细注释；`store/base/` 等基础设施的注释可以讲机制。
- 注释与文档一律中文。

## 8. 错误处理与错误码

- proxy 链路：业务错误通过 `ctx.Err` + `ctx.Code` 向上传递，由 Pipeline 统一输出；错误经 `log.WithStack` 挂栈，日志收敛在共享的 `logErrors`。
- proxy 侧错误码集中在 `proxy/types/bizcode.go`，新增码需编号唯一。
- proxy 链路**对外错误消息用英文**（网关客户端的错误契约），manager 链路**业务错误消息必须中文**；两类不要互换。
- manager 侧错误码在 `manager/base/bizcode.go`，与 proxy 解耦、两套独立编号。只保留有消费方按 code 分支的、或 HTTP 状态语义不同的通用码；**不为每种业务定义独立码**：业务错误统一 `base.ErrBadReq(中文消息)` / `base.ErrNotFound(中文消息)`，DB 等内部错误统一返回预置实例 `base.ErrInternal`。
- 前端唯一按 code 分支的是 `CodeTokenExpired`（1016，触发 refresh），编号不可变。
- 日志需包含足够上下文，但禁止记录完整 API Key 等敏感信息（见 [`security.md`](security.md)）。

## 9. 数据、用量与日志

- 仅当请求成功（`status_code < 300`）时记录 `usage_records`；流式与非流式统一入口记录，逻辑差异封装在解析层。
- Token 统计由专门的解析器或 handler 完成，避免多处重复计算。
- 各协议的流式用量解析归该协议：`Parser.ParseStreamUsage` 自行遍历 SSE 事件、自行合并字段并决定 `total_tokens` 口径（Anthropic = 完整输入 + 输出；OpenAI / Responses = 优先上游值，缺省回退 输入 + 输出）。`parser/util` 只提供 `EachSSEData`/`SplitSSEEvents`/`SSEParseData` 这类 SSE 框架级工具，不设跨协议事件结构与通用合并逻辑。
- 转发链路的请求无论成败都记入 `request_logs`；元数据端点（`v1/models`）不记录。
- 注意边界：被 echo 层拦截的请求（超过 `BodyLimit` 的 413、路由不匹配、panic）不进入 Pipeline，因此不会出现在 `request_logs`。
- 保存请求头前必须对敏感头（`Authorization` 等）脱敏；请求体目前原样入库，脱敏/截断开关仍待实现（见 `TODO.md`）。
- 日志字段应能支撑问题排查、用量审计与性能分析。

## 10. 运行时与配置不变量

这些是"顺手优化就会出事故"的点，改动前必须理解原因：

| 不变量 | 位置 | 原因 |
|--------|------|------|
| `WriteTimeout` 保持 `0` | `main.go` | 它是"请求头读完 → 响应写完"的总死线，任何固定值都会切断长 SSE 流 |
| 上游 client 不设 `Timeout`/`ResponseHeaderTimeout` | `proxy/handler/forward.go` | LLM 慢模型可能数分钟才吐第一个 token，固定死线会误杀健康长流；连接生命周期由请求 context 控制（客户端断开即取消）。只对建连/TLS 握手设超时 |
| `ReadTimeout` 10s / `ReadHeaderTimeout` 2s / `IdleTimeout` 90s | `main.go` | 只约束读取阶段，不影响响应流 |
| `BodyLimit` 10M | `framework/echo.go` | 全局请求体上限，加大需评估内存与转发行为 |
| 应用不自动建库、不自动迁移 | `store/driver/sqlite.go` | DB 文件不存在直接报错；须先手工执行 `sql/sqlite.sql`（+ `sql/init-data.sql`） |
| 客户端请求头不转发 | `proxy/handler/config.go` + `forward.go` | 上游请求头只取 `providers.config.headers`（含 `Content-Type`）；`config.domain` 不带尾斜杠，否则拼出 `//v1/...` |
| 上游 3xx 不跟随 | `proxy/handler/forward.go` 的 `CheckRedirect` | 标准库只在跨域时剥离 `Authorization`/`Cookie` 等四个头，`x-api-key`、`api-key` 这类自定义凭证头会被转发给重定向目标。理由与代价见 [`decisions/2026-09-24-refuse-upstream-redirects.md`](decisions/2026-09-24-refuse-upstream-redirects.md) |
| schema 版本三处一致 | `sql/sqlite.sql` 的 `schema_meta` 版本行、`constant.SchemaVersion`、`sql/migrations/` | 不一致会导致带着错误结构运行；门禁在 `store/gate_schema_test.go` |

## 11. 测试约定

- 测试与源码**同包**（`package handler` / `package store`，不加 `_test` 包后缀），跨层测试放 `manager/test`。
- 数据层测试自建内存 SQLite 并先调 `store.Init(db)`；不要连真实数据目录。
- 新增/修改列时同步 `sql/sqlite.sql`、`store/model/models.go`、`store/schema_test.go` 与 [`../sql/migrations/README.md`](../sql/migrations/README.md)。
- 涉及协议解析的改动必须覆盖流式与非流式两种场景；handler 优先覆盖失败路径。
- 仓库无 CI、无 linter，提交前必须本地跑 `make check`（格式 + `go vet` + `go test`）。
- **测试文件分两类，必须能一眼区分**：

  | 类型 | 命名 | 断言对象 | 怎么跑 |
  |------|------|----------|--------|
  | **门禁** | 文件 `gate_*.go`、函数 `TestGate*` | 仓库自身的一致性与规范 | `make gate`（只跑门禁）；`go test ./...` 也会跑到 |
  | **行为测试** | 其余 `*_test.go`、`Test*` | 生产代码的行为 | `make test` |

  门禁**刻意留在 `go test ./...` 里**：本仓库没有 CI，放进默认测试是让门禁不会被绕过的唯一保证；`make gate` 只是给"提交前快速过一遍"提供的快路径。
- **门禁的解析辅助函数有独立单测**（`repoparse_test.go`）：解析写错时门禁不会报错，只会**静默失效**（该拦的没拦），比没有门禁更危险。所以每个解析函数都要钉住"认哪些写法、不认哪些写法"，并验证"解析结果为空"会触发门禁自身的失效守卫。
- 路由解析用 `go/ast` 做语法感知提取，不用字符串匹配：注释里的调用、换行、缩进都不会影响结果（字符串匹配会把注释掉的注册当成真实路由）。

**门禁**：部分规则不靠自觉，由 `go test` 机械保证——

| 门禁 | 位置 | 覆盖 |
|------|------|------|
| 架构依赖 | `gate_arch_test.go` | `store` 不依赖 echo/manager/parser/service；`parser` 不依赖 store 与业务层；`service` 不依赖 echo/proxy；`manager/handler` 只有 `login.go` 允许用 echo；proxy handler 不出现 `c.JSON`/`c.String` |
| 权限种子 | `gate_arch_test.go` | 所有 `/self` 路由必须在 `sql/init-data.sql` 给 user 角色授权；种子里的路径必须是已注册路由（非 `/self` 的普通用户路由推导不出来，仍需人工判断） |
| 文档 | `gate_docs_test.go` | 相对链接与锚点可达；常驻/规则类文档不超字节预算；skill 的 YAML frontmatter 可解析 |
| 接口文档一致性 | `gate_arch_test.go` | `docs/api.md` 的接口表与 `manager/router` 的路由集合双向一致（目录型文档不比字节，比是否腐化） |
| schema 版本 | `store/gate_schema_test.go` | `sql/sqlite.sql` 写入 `schema_meta` 的版本等于 `constant.SchemaVersion`；整份 DDL 可重复执行 |

新增一条可机械判定的规则时，优先写成门禁，文档里只留一句指路（见 [`AGENTS.md`](AGENTS.md) 的放置判定）。

## 12. 数据库 schema 版本

- 版本权威是 `constant.SchemaVersion`；应用启动时与库里 `schema_meta` 表的版本行比对（`store/driver/schema.go`）：库旧 → 拒绝启动并指出要执行哪些迁移；库新 → 拒绝启动；没有版本标记（缺表或缺行）→ 告警但允许启动。
- **版本记录在元数据表里，不用 SQLite 专有的 `PRAGMA user_version`**：本项目要兼容 MySQL / PostgreSQL，元数据表是三者的公共做法。Go 侧读取只有一条 `SELECT version FROM schema_meta WHERE id = 1`，不按驱动分支；各数据库只有 DDL 语句不同（`sql/AGENTS.md` 有迁移到新库时的写法）。
- `sql/sqlite.sql` 是**当前完整 schema**，不承载迁移历史；迁移台账在 [`../sql/migrations/README.md`](../sql/migrations/README.md)。
- 版本只能递增：不支持降级，因为降级意味着丢列丢数据。决策理由见 [`decisions/2026-09-24-schema-version-and-migration-ledger.md`](decisions/2026-09-24-schema-version-and-migration-ledger.md)。

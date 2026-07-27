# AGENTS.md - 开发规则

> 本文件定义项目的大方向开发规则与架构约定。AI Agent 在修改或扩展代码前，应先理解并遵循这些规则。

## 1. 项目定位

`aiapi` 是一个轻量级 AI 大模型 API 反向代理，核心职责是：

- 接收多种客户端协议（OpenAI / Gemini / Anthropic 等）的请求
- 按配置路由到对应的上游 Provider
- 透传响应（含 SSE 流式）
- 记录 Token 用量与请求日志
- 提供 Provider 与 API Key 的管理接口

## 2. 架构原则

### 2.1 分层清晰，各司其责

| 层次 | 职责 | 代表位置 |
|------|------|----------|
| 入口层 | 参数解析、日志/数据库初始化、启动服务 | `main.go` |
| 框架适配层 | HTTP 路由注册、请求参数提取、响应写入 | `framework/echo.go` |
| 业务编排层 | 组装并执行请求处理 Pipeline | `proxy/direct.go` |
| 业务处理层 | 单一职责的 handler，完成具体业务逻辑 | `proxy/handler/*.go` |
| 协议解析层 | 不同厂商的请求/响应解析与 Key 提取 | `parser/*.go` |
| 后台管理层 | 后端给前端的接口入口，只做 HTTP 适配（参数校验、调 service、组装响应） | `manager/handler/*.go` |
| 业务服务层 | 跨 handler 复用的业务逻辑、多表事务编排，不依赖 echo | `service/*.go` |
| 后台中间件层 | 登录态校验、接口级权限判定，只做 HTTP 适配 | `manager/middleware/*.go` |
| 数据持久层 | 数据库访问、模型定义、查询构造 | `store/*.go` |
| 通用工具层 | 常量、日志格式化、SSE 包装等 | `constant/`、`log/`、`proxy/sse/` |

### 2.2 关键边界

- `framework/echo.go` 只做 HTTP 参数解析和响应适配，不写业务逻辑。
- `proxy/direct.go` 只负责 Pipeline 组装，不实现具体业务。
- `proxy/handler/` 是代理业务逻辑唯一入口，每个 handler 应尽量独立、可测试。
- `parser/` 只负责协议相关的解析与提取，不直接操作数据库或写响应。
- `store/` 是纯 SQL 包装层：只做单条/单表的数据读写（`Get/Select/Exec`），不处理 HTTP 或协议细节，**不写跨表编排、不写业务判断、不写事务体**；每个 Store 的命名空间入口写在自己文件内（如 `func (s *Session) Xxx() *XxxStore`），不集中到 `store/base.go`。按表/领域拆分 Store，不同表的操作不混在同一 Store（如操作 `api_keys.model_policy` + `apikey_model_access` 的 `ModelAccessStore` 独立于操作 `models` 表的 `ModelStore`）。`IN` 查询直接写 `IN (:ids)` 传 slice 参数，QueryBuilder 内置 `sqlx.In` 自动展开，不手动拼接占位符。
- `manager/handler/` 是后端给前端的接口唯一入口，只做 HTTP 适配：参数校验、调 service、组装响应；**不直接写跨表事务、不写可复用业务逻辑**。
- `service/` 是业务逻辑承载层（manager 与 proxy 共用）：跨 handler 复用、不依赖 echo 的业务逻辑、**涉及多表/带业务语义的事务编排**、跨 Store 组装、业务判断（如分组维度→排序方向）都放这里；handler 与 middleware 只调用 service 暴露的方法，不重复实现。proxy 侧无 manager 入口的场景（如鉴权、计费）也调 `service/`，不在 `proxy/handler` 自建业务逻辑。
- `manager/middleware/` 只做 HTTP 适配（鉴权、权限判定、登录态注入），复杂业务下沉到 service。

### 2.3 事务边界

- `Session.T(fn)` 只提供事务执行入口，**事务体（fn 内组合多个 Store 调用、业务判断、失败回滚语义）不应写在 `store/` 内**，应写在 `service/`（manager 与 proxy 共用的业务层）。
- `store/` 里的方法默认在调用方传入的 `*Session` 上执行：若调用方用 `Session.T(fn)` 包裹则自动进事务，若用 `store.C()` 则非事务，store 方法本身不感知是否事务。
- Session 保存 `tx`/`page` 状态字段，不再用 `context.Value` 传业务参数（tx、分页），context 回归“请求级跨边界数据”本职。`store.C()` 无参，`Session.T(fn)` 在当前 Session 上切换 tx（嵌套调用复用当前 tx，不开新事务，保证原子性）。
- 一个事务只编排「同一业务动作」需要的多张表，不要把无关写操作塞进同一事务。

### 2.4 并发安全（金额/计数类写操作）

涉及余额、计数等「读后算再写」的场景，必须保证读到的值是本次写入前的稳定值，禁止「先 SELECT 再 UPDATE 写回」的读后写模式（快照读拿旧值会丢更新）。

通用做法（跨 SQLite/MySQL/PostgreSQL）：

1. 先 `UPDATE budget = budget + :amount`（对同一行的 UPDATE 加行锁串行执行，金额不丢更新）
2. 再在同事务内 `SELECT budget`（事务内自己修改对后续语句立即可见，拿到的是步骤 1 之后的新值）
3. 行锁持续到 COMMIT，期间其他事务的 UPDATE 阻塞，故步骤 2 读到的是稳定值
4. 需要记 before/after 流水时，`before = after - amount` 反推

禁止的反模式：
- 先 `SELECT budget` 再 `UPDATE budget = :newValue`（读后写，快照读拿旧值 → 丢更新）
- 依赖 `UPDATE ... RETURNING`（MySQL 8 不支持，跨库不通用）
- 依赖 `SELECT ... FOR UPDATE`（SQLite 不支持该语法）

### 2.5 注释风格

- `store/` 方法注释只说明「这个方法做什么」，不描述事务用法、调用时机、并发语义等——这些属于 service/handler 的编排逻辑，写在 store 里是越界且误导。
- service/handler 的复杂逻辑（事务编排、并发安全原理）才需要详细注释。

## 3. 代码组织规则

### 3.1 新增功能时，先找对应层次

- 新增上游协议支持 → `parser/`
- 新增请求处理步骤 → `proxy/handler/`，并在 `proxy/direct.go` 的 Pipeline 中注册
- 新增管理接口 → `manager/handler/`（HTTP 适配）+ `service/`（业务逻辑）
- 新增业务逻辑（事务编排、跨表组装、业务判断）→ `service/`
- 新增数据表或查询 → `store/` 与 `sql/`
- 新增通用常量或工具 → `constant/` 或 `log/`

### 3.2 目录命名规则

- 业务包使用小写、下划线命名，如 `parse_request.go`
- 测试文件与源码文件同名，后缀 `_test.go`
- 避免在业务目录中混入框架适配代码

### 3.3 业务模型与数据模型分离

- 数据模型定义在 `store/model/models.go`
- 业务逻辑中不直接依赖 SQL 细节，应通过 `store` 提供的接口操作数据

## 4. Pipeline 设计规则

### 4.1 核心约定

- 使用 Pipeline 编排请求处理流程，每个 handler 只负责一个步骤。
- Handler 成功时直接返回，失败时设置 `ctx.Err` 与 `ctx.Code`，由 Pipeline 统一输出错误响应。
- 不可中断的收尾操作（如写日志）应注册为 `Finally` handler。
- 禁止在 handler 中自行检查 `ctx.Err` 来决定是否执行。

### 4.2 Handler 编写原则

- 单一职责：一个 handler 只做一件事。
- 无副作用：失败时只设置错误，不写入响应体。
- 可测试：handler 应能独立于 Echo 和真实数据库进行单元测试。

## 5. 错误处理规则

- 所有业务错误通过 `ctx.Err` + `ctx.Code` 向上传递。
- 错误码集中管理，新增错误码需保持编号唯一，并在 `proxy/types/bizcode.go` 中注册。
- 禁止在 handler 中直接调用 `c.JSON` 或 `c.String` 返回错误。
- 错误日志应包含足够上下文，但禁止记录完整的 API Key 等敏感信息。

## 6. 数据与日志规则

### 6.1 用量记录

- 仅当请求成功（`status_code < 300`）时记录 `usage_records`。
- Token 统计应由专门的解析器或 handler 完成，避免多处重复计算。
- 流式与非流式请求统一入口记录，逻辑差异封装在解析层。

### 6.2 请求日志

- 所有请求无论成败都记录到 `request_logs`。
- 保存请求头前必须对敏感头（如 `Authorization`）脱敏。
- 日志字段应能支撑问题排查、用量审计与性能分析。

## 7. 扩展规则

### 7.1 新增协议格式

1. 在 `parser/` 下新增解析器实现 `Parser` 接口。
2. 在 `parser/interface.go` 中注册并返回该解析器。
3. 保持与现有解析器一致的接口签名和行为语义。

### 7.2 新增 Pipeline Handler

1. 在 `proxy/handler/` 下新增文件。
2. 函数签名统一为 `func Xxx(ctx *types.Context)`。
3. 根据是否需要收尾执行，选择 `AddLast` 或 `AddFinally` 注册。
4. 优先将 handler 放在逻辑上合理的位置，避免破坏现有流程顺序。

### 7.3 新增 Manager API

`manager/` 是后端管理服务的总入口，不仅限于 Provider，还包括用户、充值、数据统计、模型配置等管理功能。

1. 在 `manager/handler/` 下按业务领域新增文件（如 `user.go`、`charge.go`）。同一领域的普通用户版与超管版合并到同一文件，不单独起 `*_admin.go` 文件。handler 只做 HTTP 适配：参数校验、调 service、组装响应，不写跨表事务。
2. 带业务语义的逻辑（尤其多表事务编排、跨 handler 复用逻辑）下沉到 `service/`，handler 只调用 service 暴露的方法。单表的简单读写可直接在 handler 里调 store，但仍建议统一走 service 以保持一致性。
3. 业务函数签名自由组合 `(context.Context, *Req)` 等入参，返回 `(Resp, *base.BizError)`；由 `base.Wrap` 动态包装成 echo.HandlerFunc。登录态从 `context.Context` 取（`base.CurrentUser`）。
4. 在 `manager/router/router.go` 的 `Register(g *echo.Group)` 中用 `base.Wrap(handler.Xxx)` 注册路由；路由组根路径 `/manager` 由 `framework/echo.go` 直写。
5. **接口命名**：列表查询统一用 `/list` 后缀（如 `/users/list`、`/providers/list`、`/models/list`、`/recharge/records/list`）。普通用户自助接口用 `/self` 后缀，超管接口不加 `/self`。
6. **中间件挂载**：`login` / `refresh` 不挂 Auth；`logout` 挂 Auth 不挂 Require；其余挂 Auth + Require。
7. **self / admin 合并模式**：同一业务的自助版与超管版合并为一个通用函数，self 入口只设当前用户 ID 后委托。如 `RechargeSelf` 设 `req.UserID = cur.ID` 后调 `Recharge`；`RechargeRecordsSelf` 设 `req.UserID = cur.ID` 后调 `RechargeRecords`。通用函数做参数校验 + 调 service，self 入口不重复校验逻辑。
8. **分页接口**：列表查询接口应支持分页，采用 Session 状态 + 显式控制模式：
   - 分页类型分层：`manager/base` 定义 `PageReq`（入参，内嵌到 Req 结构体）与 `PageResult[T]`（出参）；`store` 暴露 `PageContext`（传给 `store.C().SetPage` 的内部载体，拦截器写回 `Total`）。handler 不直接 import `store/base`。
   - handler 创建 `store.PageContext{Page, PageSize}`，用 `store.C().SetPage(pc).Charge().List(...)` 链式调用，store 方法写普通 `Select`，`QueryBuilder` 从 Session 读 `PageContext`，有则自动拦截：先 `SELECT COUNT(*) FROM (<原SQL>) t` 查总数写回 `pc.Total`，再追加 `LIMIT ? OFFSET ?` 查当前页
   - 非事务单次分页：链式调用，Session 用完即弃，不用 `ClearPage`
   - 事务内多次查询：用 `s.SetPage(pc)` / `s.ClearPage()` 显式控制分页作用于哪些语句
   - handler 从 `pc.Total` 拿总数，组装 `*base.PageResult[T]{Items, Total, Page, PageSize}` 返回
   - `page` 1-based，`page_size` 默认 20、上限 100
   - 未 `SetPage` 时 `Select` 退化为普通查询

### 7.4 新增数据库实体

1. 在 `sql/sqlite.sql` 中补充 DDL。
2. 在 `store/model/models.go` 中定义模型。
3. 在 `store/` 下新增对应 Store 文件，提供单表 CRUD 方法，不写跨表编排、不写业务判断。按表拆分 Store，不同表不混在同一 Store。
4. 命名空间入口写在该 Store 文件内（如 `func (s *Session) Xxx() *XxxStore`），不要集中放到 `store/base.go`。
5. 涉及该实体的多表事务编排、跨 Store 组装、业务判断写在 `service/`，用 `store.C().T(fn)` 包裹，在 fn 内组合多个 Store 调用。
6. `IN` 查询直接写 `IN (:ids)` 传 slice 参数，QueryBuilder 内置 `sqlx.In` 自动展开，不手动拼接占位符。

### 7.5 开发管理台前端

`frontend/` 是 Vue 3 + Vite + Naive UI 管理台前端，编译后通过 Go `embed` 嵌进二进制分发。

1. 开发期 `make dev-ui`（端口 3000），Vite 自动代理 `/manager` API 到 Go。
2. 新增页面：在 `frontend/src/views/` 下写 Vue 单文件组件，使用 Naive UI 组件（`n-card`/`n-button`/`n-data-table`/`n-modal` 等），图标库用 `@vicons/ionicons5`。`router/index.js` 加入 Home 路由的 children。
3. 新增接口调用：在 `frontend/src/api/index.js` 封装 `request(PATH, body)`。
4. 侧栏菜单：在 `frontend/src/views/Home.vue` 的 `<nav>` 中加 `<router-link>`。
5. 发布：`make build-all`，Go 二进制自动嵌入 `frontend/dist/`。

#### 前端公共模块

| 模块 | 路径 | 职责 |
|------|------|------|
| 工具函数 | `frontend/src/utils.js` | `fix4`（金额格式化，4 位小数去尾 0）、`formatTime`（时间格式化） |
| 图表选项 | `frontend/src/charts.js` | ECharts option 构建器、`metricConfig` 指标配置 |
| 图表生命周期 | `frontend/src/composables/useChart.js` | ECharts init/resize/dispose，`watch` 数据自动重绘 |
| 分页逻辑 | `frontend/src/composables/usePagination.js` | `n-data-table` 远程分页通用逻辑（`pagination`/`onPage`/`onPageSize`/`resetAndLoad`），含 `showTotal` 展示总数 + `showSizePicker` 切换每页条数 |

#### 表格风格统一

所有页面使用 `n-data-table` 时保持一致：
- `:bordered="false"` `size="small"` — 无边框紧凑风格
- 列宽超出容器时加 `:scroll-x="总宽"` 防溢出
- 金额列统一用 `fix4()` 格式化
- 时间列统一用 `formatTime()` + `ellipsis: { tooltip: true }` 防换行
- 空数据兜底 `value = (await xxx()) || []`
- 删除/禁用等危险操作用 `useDialog` 二次确认，不用 `window.confirm`

#### 菜单与权限模型

- **动态菜单**：侧栏菜单由后端 `/manager/self` 返回的 `menus` 树驱动，前端 `Home.vue` 动态渲染，不硬编码。菜单数据存于 `menus` 表，角色与菜单通过 `role_menus` 关联。
- **菜单形态**：有 `children` 为分组容器（点击展开/收起）；无 `children` 且 `path` 非空为直接跳转；`path` 为空为纯标题。
- **超管特权**：`role_permission` 中 `value='*'` 为超管通配权限，放行所有接口。admin 角色只需配一条 `('API', '*', '*')` 即可访问全部超管接口，无需为每个接口单独授权。
- **普通用户**：按接口路径精确授权（最小权限原则）。
- **超管页面**：路由 `/admin/*` 下，页面组件放 `frontend/src/views/admin/`，import 路径多一级 `../../`。

## 8. 测试与质量

- 新增 handler 应配套单元测试，优先覆盖失败路径。
- 数据层测试应使用 SQLite 内存数据库或事务回滚，避免污染真实数据。
- 涉及协议解析的变更，应覆盖流式与非流式两种场景。
- 提交前确保 `go test ./...` 与 `go build` 通过。

## 9. 文档维护规则

- 开发规则、架构约定或项目边界发生变化时，必须同步更新 `AGENTS.md`，并在 `CHANGELOG.md` 中说明影响。
- 用户可见的功能、接口、部署方式发生变化时，必须同步更新 `README.md`，并在 `CHANGELOG.md` 中记录。
- 破坏性变更、模块拆分、目录调整或接口行为变化，若可能影响使用者或其他开发者，应在 `CHANGELOG.md` 中说明。
- 日常的内部文件移动、函数重构、变量重命名等不影响外部使用的改动，不需要单独在 `CHANGELOG.md` 中记录。
- `AGENTS.md` 描述开发规则与架构约束，不罗列具体参数和接口返回示例。
- `README.md` 面向使用者，包含部署、调用示例和接口说明。
- 三者分工明确：规则在 AGENTS，用法在 README，变更历史在 CHANGELOG。

## 10. 安全与隐私

- API Key 只在认证阶段使用，禁止在日志、错误信息或响应中泄露完整 Key。
- 请求体中的敏感内容应谨慎记录，必要时提供脱敏开关。
- 管理接口应考虑访问控制，避免任意用户修改 Provider 配置。
- `manager/` 下所有需登录态的接口必须经过 `manager/middleware.Auth` 中间件（access JWT 校验 + 接口级权限 `role_permission(entity=API, value=path)` 判定 + 注入登录态），业务函数由 `base.Wrap` 做参数包装与响应输出。
- 登录态采用双 token 机制：access JWT（HS256 自实现，15min 无状态，经 `Authorization: Bearer` 头传递，前端存内存）+ refresh token（随机串哈希存 `user_sessions` 表，HttpOnly Secure cookie `refresh_token` 传递，滑动 7 天 / 绝对 30 天，带轮换与重用检测）。`/manager/login`、`/manager/refresh` 不挂 Auth 中间件（login 无需登录态；refresh 靠 refresh cookie 续期，不依赖 access JWT）。
- access JWT 签名密钥走环境变量 `AIAPI_JWT_SECRET`（≥32 字节），启动时由 `base.LoadJWTSecret` 校验；登录会话业务封装在 `service` 的 `SessionService`，改密 / 禁用用户 / 重置密码后吊销该用户所有会话。
- 登录态不复用 proxy 的 header 鉴权字段；账号不存在 / 禁用 / 密码错统一返回相同文案防枚举。
- `manager` 自有业务码定义在 `manager/base/bizcode.go`，与 `proxy/types/bizcode.go` 解耦，两套独立编号。

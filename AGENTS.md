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
| 后台管理层 | 登录/权限/充值等管理 API | `manager/handler/*.go` |
| 数据持久层 | 数据库访问、模型定义、查询构造 | `store/*.go` |
| 通用工具层 | 常量、日志格式化、SSE 包装等 | `constant/`、`log/`、`proxy/sse/` |

### 2.2 关键边界

- `framework/echo.go` 只做 HTTP 参数解析和响应适配，不写业务逻辑。
- `proxy/direct.go` 只负责 Pipeline 组装，不实现具体业务。
- `proxy/handler/` 是业务逻辑唯一入口，每个 handler 应尽量独立、可测试。
- `store/` 只负责数据读写，不处理 HTTP 或协议细节。
- `parser/` 只负责协议相关的解析与提取，不直接操作数据库或写响应。

## 3. 代码组织规则

### 3.1 新增功能时，先找对应层次

- 新增上游协议支持 → `parser/`
- 新增请求处理步骤 → `proxy/handler/`，并在 `proxy/direct.go` 的 Pipeline 中注册
- 新增管理接口 → `manager/`
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

1. 在 `manager/handler/` 下按业务领域新增文件（如 `user.go`、`charge.go`）。
2. 业务函数签名自由组合 `(context.Context, *Req)` 等入参，返回 `(Resp, *base.BizError)`；由 `base.Wrap` 动态包装成 echo.HandlerFunc。
3. 在 `manager/router/router.go` 的 `Register(g *echo.Group)` 中用 `base.Wrap(handler.Xxx)` 注册路由；路由组根路径 `/manager` 由 `framework/echo.go` 直写。

### 7.4 新增数据库实体

1. 在 `sql/sqlite.sql` 中补充 DDL。
2. 在 `store/model/models.go` 中定义模型。
3. 在 `store/` 下新增对应 Store 文件，提供 CRUD 方法。
4. 命名空间入口写在该 Store 文件内（如 `func (s *Session) Xxx() *XxxStore`），不要集中放到 `store/base.go`。

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
- `manager/` 下所有需登录态的接口必须经过 `manager/middleware.Auth` 中间件（登录校验 + 接口级权限 `role_permission(entity=API, value=path)` 判定 + 注入登录态），业务函数由 `base.Wrap` 做参数包装与响应输出。
- 登录态只走 Cookie + 服务端 `user_sessions` 表，不使用 `Authorization` 头、不复用 proxy 的 header 鉴权字段。
- `manager` 自有业务码定义在 `manager/base/bizcode.go`，与 `proxy/types/bizcode.go` 解耦，两套独立编号。

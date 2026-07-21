# CHANGELOG

> 记录项目的版本变更、功能新增、破坏性变更以及对用户/开发者有影响的改动。
> 按时间倒序排列，当前尚未发布版本的内容放在 `[Unreleased]` 下。

## [Unreleased]

### 2026-07-20

- Token 用量统计增强。
  - 统计新增字段：缓存命中 Token (`cached_tokens`)、缓存未命中 Token (`cache_miss_tokens`)、推理 Token (`reasoning_tokens`)，表格与顶部指标均已覆盖。
  - 接口响应结构调整：`/usage/stats/self` 改为返回 `{summary, rows}`，顶部汇总由后端计算。
  - 按 api_key 分组时，已删除的 key 名称标记为红色（「已删除」）。
  - 日期查询改为左闭右闭（`00:00:00 ~ 23:59:59`），按天模式含结束日全天数据。
  - 空结果时 `rows` 返回 `[]` 而非 `null`，前端兜底处理。
  - 移除 `/usage/detail/self` 接口相关代码。

- 新增 Token 用量与费用统计功能。
  - 新增管理接口（均需登录 + 接口授权）：
    `POST /manager/usage/stats/self`、`/usage/filters/self`。
  - 支持按天/月聚合统计（天模式限31天），可按 api_key / 模型 / 提供商组合筛选。
  - api_key 筛选用 id 而非完整 key（防泄露）。
  - `store/usage.go` 新增 `StatsByUser` / `DistinctModelsByUser` / `DistinctProvidersByUser`。
  - 前端新增「使用统计」页面（`frontend/src/views/Usage.vue`）。
  - `sql/sqlite.sql` 种子权限注释补充 2 条 `/manager/usage/*/self` 路径。

- 新增个人 API Key 管理功能（用户自助）：列表 / 创建 / 启停 / 删除 / 重命名 / 修改额度。
  - 新增管理接口（均需登录 + 接口授权）：
    `POST /manager/apikeys/list/self`、`/apikeys/create/self`、`/apikeys/toggle/self`、`/apikeys/delete/self`、`/apikeys/rename/self`、`/apikeys/budget/self`。
  - 所有读写接口先校验 key 归属当前用户，防止越权。
  - API Key 生成采用 `crypto/rand` 32 字节随机 + `sk-` 前缀（256bit 熵），唯一性由 `uq_api_keys_key` 兜底，冲突重试 3 次。
  - 列表接口对 key 做脱敏（`sk-xxxx****xxxx`），明文 key 仅在创建响应中返回一次。
  - 额度约束：所有「有限额」密钥的 budget 之和不能超过账户余额；创建与修改额度接口均做该校验，无限账户跳过。新增 `CodeBudgetExceeded`（1010）。
  - `store/apikey.go` 补充 `GetByID` / `ListByUser` / `Create` / `SetEnabled` / `UpdateName` / `UpdateBudget` / `SumLimitedBudgetByUser`，并暴露 `IsUniqueConstraintErr`。
  - `manager/base/bizcode.go` 新增 `CodeApiKeyNotFound`（1009）、`CodeBudgetExceeded`（1010）。
  - 前端新增「API 密钥」页面（`frontend/src/views/ApiKeys.vue`），含创建弹窗、明文 key 一次性展示与复制、名称列内联改名图标、修改额度弹窗（限制/无限制二选一，限制时填金额且不超账户可用额度）、启停/删除操作。
  - `App.vue` 包裹 `n-message-provider` / `n-config-provider` 以支持 `useMessage`。
  - `sql/sqlite.sql` 种子权限注释同步补充 6 条 `/manager/apikeys/*/self` 路径（admin + user 各一份）。

### 2026-07-19

- 新增后台管理前端（`frontend/`）：Vue 3 + Vite 管理台，Go `embed` 嵌进二进制。
  - 页面：登录 / 充值（弹窗充值 + 流水表格）。
  - 开发期 `make dev-ui`（端口 3000），自动代理 API 到 Go 后端。
  - 生产构建 `make build-all`，一个 `aiapi` 文件启动即带前端页面。
- 新增后台管理功能（`manager/`）：账号密码登录 / 登出、接口级权限 `(role_id, entity, action, value)`、普通用户自充值、管理员给任意用户充值。
  - 登录态基于服务端 `user_sessions` 表 + HttpOnly Cookie，不使用 `Authorization` 头。
  - 多角色（用户-角色多对多）；接口级 ACL，`role_permission` 表用 `entity=API, value=接口路径` 授权，请求路径命中即放行；`action` 保留暂作通配。
  - 所有管理接口一律 POST + JSON body；manager 自有业务码（`manager/base/bizcode.go`），与 proxy 解耦。
  - 复用已有 `recharge_records` 表与 `store.Charge()`，新增 `RechargeWithRecord` 在事务内完成"读余额→加余额→写流水"。
- 数据库结构变更（`sql/sqlite.sql`）：
  - `users` 新增 `password`（bcrypt 哈希）、`updated_at` 列。**存量库需手动迁移**：
    `ALTER TABLE users ADD COLUMN password TEXT NOT NULL DEFAULT '';`
    `ALTER TABLE users ADD COLUMN updated_at DATETIME DEFAULT (datetime('now','localtime'));`
  - 新增表：`roles`、`user_roles`、`role_permission`、`user_sessions`。
- 依赖：`golang.org/x/crypto` 提为直接依赖（bcrypt）。

### 2026-07-19

- 新增项目变更日志（CHANGELOG.md）。
- 重写 `AGENTS.md` 与 `README.md`：
  - `AGENTS.md` 改为大方向开发规则，明确分层边界、Pipeline 设计、扩展规则与文档维护职责。
  - `README.md` 明确项目当前仅代理 OpenAI 兼容格式请求，修正架构描述与使用示例。

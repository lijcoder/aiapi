# CHANGELOG

> 记录项目的版本变更、功能新增、破坏性变更以及对用户/开发者有影响的改动。
> 按时间倒序排列，当前尚未发布版本的内容放在 `[Unreleased]` 下。

## [Unreleased]

### 2026-07-24

- 新增双 token 登录机制（access JWT + refresh 轮换 + 重用检测），替换原单 token 方案，适配公网部署。
  - **access JWT**：HS256 自实现，15min 有效期，无状态不落库，经 `Authorization: Bearer` 头传递，前端存内存（不放 localStorage 防 XSS）。
  - **refresh token**：随机串哈希后存 `user_sessions` 表（明文不落库），HttpOnly Secure cookie `refresh_token` 传递（Path=`/manager`，SameSite=Lax），滑动 7 天 / 绝对 30 天。
  - **轮换**：每次 `/manager/refresh` 删旧发新（同 family），滑动续期顺延但不超绝对上限。
  - **重用检测**：旧 refresh token 再次被使用时吊销整个 family，强制重登（业务码 1017）。
  - 改密 / 禁用用户 / 重置密码后吊销该用户所有会话，即时失效。
  - 账号不存在 / 禁用 / 密码错统一返回相同文案，防账号枚举。
  - 新增 `manager/service` 包封装登录业务（`SessionService`），handler/middleware 仅做 HTTP 适配。
  - 新增接口 `POST /manager/refresh`（不挂 Auth 中间件，靠 refresh cookie）；`/manager/login` 返回体改为 `{access_token, expires_in}`。
  - access JWT 签名密钥走环境变量 `AIAPI_JWT_SECRET`（≥32 字节），启动时由 `base.LoadJWTSecret` 校验，缺失或过短启动失败。
  - 业务码新增 `CodeTokenExpired`(1016) / `CodeSessionReuse`(1017)。
  - 数据库变更：`user_sessions` 表新增 `family_id` / `absolute_expires_at` / `ua` / `ip` 列，新增 `idx_user_sessions_family` 索引。**存量库迁移**：执行 `ALTER TABLE user_sessions ADD COLUMN ...` 四条 + 建索引 + `DELETE FROM user_sessions;`（旧 token 格式不兼容，用户需重登一次）。
- 前端适配双 token 登录机制：access JWT 存内存；401 且业务码为 1016 时自动调 `/manager/refresh` 续期并重试原请求；并发请求合并（多个 401 只触发一次刷新）；页面刷新后靠 refresh cookie 静默恢复会话；`/login` 已登录则跳首页，登出 / 改密后清 access token。

### 2026-07-23

- 修复统计页 / 全局统计 / 仪表盘时间趋势折线倒序问题：时间维度由 `label DESC` 改为 `ASC`（从旧到新正序），维度分组（模型 / 提供商 / Key / 用户）保持 DESC 不变。

### 2026-07-22

- 个人设置页改用 `n-form` 左对齐布局：`Profile` / `ChangePassword` 表单 label 左对齐，按钮区与输入框左边缘对齐；修改密码页顶部加浅红色提示「修改密码后需要重新登录」。

- 新增个人设置功能（修改姓名/邮箱/密码）。
  - `users` 表新增 `email` 字段（可空、不唯一）。
  - 新增 2 个自助接口：`/manager/profile/update/self`（改姓名+邮箱）、`/manager/profile/password/self`（改密码，需校验旧密码）。
  - 修改密码后前端提示并跳转登录页，强制重新登录。
  - 新增「个人设置」菜单项（id=11，所有角色可见），`/profile` 页面包含基本资料与修改密码两个区块。
  - `/manager/self` 返回的 user 信息补充 `email` 字段。
  - 存量库迁移：`ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT '';` + `INSERT OR IGNORE INTO menus ...` / `role_menus ...`。

- 新增 API Key 级模型访问控制。
  - `api_keys` 表新增 `model_policy` 字段（默认 `'all'`），`all`=全量放行，`whitelist`=按白名单。
  - 新增 `apikey_model_items` 表存储白名单明细（model_policy=`whitelist` 时生效）。
  - Proxy Pipeline 在 Auth handler 内追加模型权限校验（紧跟模型配置校验之后），按 Key 策略判定是否可访问请求模型，不通过时返回 404（对外不暴露权限细节）。
  - 管理端新增 4 个接口：超管版 `/manager/apikeys/models/get`、`/set`；普通用户自助版 `/apikeys/models/get/self`、`/set/self`（自助版校验 Key 归属）。
  - 前端新增可复用 `ModelAccessDialog` 组件，超管与普通用户的 Key 管理页操作列均加「模型权限」按钮，支持全量/白名单切换，白名单按 provider 分组多选。
  - 删除 API Key 时事务内级联清理白名单明细。
  - 存量库迁移：`ALTER TABLE api_keys ADD COLUMN model_policy TEXT NOT NULL DEFAULT 'all';` + 建 `apikey_model_items` 表。

- 修复全局统计「按 Key」分组时表格不显示 Key 名称的问题。admin 版原先因「跨用户 key 名不唯一」而不填充名称，导致主行只显示「-」；现改为查全部 Key 构建 key→name 映射并填充，同时优化前端渲染：有名称时显示「名称 + 脱敏 Key」双行，无名称（已删除/未命名）时直接显示脱敏 Key 单行。普通用户统计页同步优化。

- 超管侧栏菜单「模型定价」改名为「模型管理」，相关页面卡片标题与代码注释同步统一。存量库需执行 `UPDATE menus SET name='模型管理' WHERE id=8;`。

- 模型多模态能力配置（第一阶段：配置 + 展示）。
  - `models` 表新增 `supports_text` / `supports_image` / `supports_video` 三个布尔列（默认 `1/0/0`），分别标记模型支持的文本/图像/视频模态。
  - 模型新增/编辑接口与表单同步支持这三项配置，新建模型默认勾选「文本」。
  - 普通用户与超管的模型列表均新增「能力」列，以彩色 `n-tag` 展示已支持模态。
  - 存量库迁移：执行 `ALTER TABLE models ADD COLUMN ...` 三条语句即可，详见 `sql/sqlite.sql` 内注释。

### 2026-07-21

- 新增超级管理后台（超管视角的全局管理功能）。
  - **动态菜单**：新增 `menus` / `role_menus` 表，`/manager/self` 返回当前用户菜单树，前端 `Home.vue` 动态渲染侧栏（支持分组容器/直接跳转/纯标题三种形态），不再硬编码菜单。
  - **超管特权**：`role_permission` 支持 `value='*'` 通配放行所有接口，admin 角色只需一条权限记录即可访问全部超管接口。
  - **用户管理** `/manager/users/*`：列表（含角色 tag）/ 创建 / 编辑 / 启停 / 重置密码 / 分配角色 / 充值；操作列下拉菜单含「API Key」「充值记录」入口，跳转子页面管理该用户的 Key 和流水。
  - **提供商管理** `/manager/providers/*`：列表 / 新增 / 编辑 / 查看（headers 键值对格式化展示）/ 启停；禁用前二次确认。
  - **模型定价管理** `/manager/models/*`：列表（后端按 provider/model 模糊搜索）/ 新增 / 编辑 / 删除。
  - **全平台充值记录** `/manager/recharge/records/list`：JOIN users 拿用户名和操作人名，支持按用户名/账号/备注模糊搜索。
  - **全局统计** `/manager/usage/stats` `/manager/usage/filters`：不限定 user_id，支持 `group_by=user` 维度；筛选下拉含用户列表；复用现有图表框架。
  - **仪表盘** `/manager/dashboard`：8 个指标卡（用户数/Key数/今日请求/今日费用/今日输入Token/今日输出Token/今日总Token/今日缓存命中率）+ 近 7 天趋势图（6 指标切换：费用/请求数/输入Token/输出Token/总Token/缓存命中率）。
  - **API Key 管理（超管版）** `/manager/apikeys/*`（无 `/self` 后缀）：超管管理指定用户的 Key，不校验归属，额度校验用 Key 归属用户的余额。
  - **角色列表** `/manager/roles/list`：用于分配角色弹窗。
  - 数据库变更：新增 `menus` / `role_menus` 表；`roles` 表去掉 `description` 列、新增 `code` 列；`users` 表去掉 `updated_at` 列；`recharge_records` 查询 JOIN users 返回 `user_name` / `operator_name`。
  - 业务码新增：`CodeAccountExists(1011)` / `CodeProviderExists(1012)` / `CodeProviderNotFound(1013)` / `CodeModelExists(1014)` / `CodeModelNotFound(1015)`。
  - 新增 `sql/init-data.sql` 独立存放初始数据（角色/权限/菜单/角色菜单），用 `INSERT OR IGNORE` 可重复执行。

### 2026-07-20

- Token 用量统计增强。
  - 统计新增字段：缓存命中 Token (`cached_tokens`)、缓存未命中 Token (`cache_miss_tokens`)、推理 Token (`reasoning_tokens`)、缓存命中率 (`cache_hit_rate` = cached_tokens / input_tokens)，表格与顶部指标均已覆盖。
  - 接口响应结构调整：`/usage/stats/self` 改为返回 `{summary, rows}`，顶部汇总由后端计算。
  - 按 api_key 分组时，已删除的 key 名称标记为红色（「已删除」）。
  - 日期查询改为左闭右闭（`00:00:00 ~ 23:59:59`），按天模式含结束日全天数据。
  - 空结果时 `rows` 返回 `[]` 而非 `null`，前端兜底处理。
  - 移除 `/usage/detail/self` 接口相关代码。

- 新增 ECharts 图表功能。
  - 时间趋势：费用折线图 / 总 Token 多线面积图 + 费用双轴 / 缓存命中面积图 / 缓存命中率百分比折线图。
  - 按维度（模型/提供商/Key）：环形图展示占比；比率指标用水平柱状图。
  - 支持 费用/总Token/缓存命中/缓存命中率 四个指标切换。
  - 图表代码重构为可复用模块：`composables/useChart.js`（生命周期）、`charts.js`（option 构建器）。
  - 依赖新增 `echarts`。

- 前端统一优化。
  - 金额格式化统一用 `fix4`（提取到 `utils.js`），时间格式化 `formatTime` 提取到 `utils.js`。
  - 所有页面 `n-data-table` 风格对齐：`bordered=false`、`size=small`、`scroll-x` 防溢出、时间列 `ellipsis` 防换行。
  - 充值流水操作人展示 `users.name`（`ListRechargeRecords` LEFT JOIN users）。
  - 充值/模型/密钥页面空数据 `|| []` 兜底，避免 null 导致页面卡死。
  - 默认首页改为使用统计，默认按月查询当月数据，修复 UTC 时区导致日期偏差。
  - 侧栏菜单顺序：使用统计 → API 密钥 → 模型列表 → 充值中心。

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

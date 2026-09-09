# CHANGELOG

> 记录项目的版本变更、功能新增、破坏性变更以及对用户/开发者有影响的改动。
> 按时间倒序排列，当前尚未发布版本的内容放在 `[Unreleased]` 下。

## [Unreleased]

- 新增 OpenAI Responses API 支持（透传）：新增 `parser/responses.go`（`ResponsesParser`）与格式常量 `FormatResponses = "openai-responses"`，客户端可经 `/proxy/openai-responses/:provider/v1/responses` 以 Responses 格式调用（含 `stream: true` 流式），请求原样透传到上游。解析器适配 responses 专用的 usage 字段（`input_tokens` / `output_tokens` / `input_tokens_details.cached_tokens` / `output_tokens_details.reasoning_tokens`，与 chat 的 `prompt_tokens` / `completion_tokens` 不同）与流式事件（`response.created` / `response.output_text.delta` / `response.completed` 等，无 `[DONE]` 标记），使用量统计与计费对 Responses 请求生效。鉴权仍为 `Authorization: Bearer`，模型名取请求体顶层 `model`。无 DB 结构与路由改动（路由即 `/:format/:provider/*`）。`GET v1/models` 在 `openai-responses` 协议下由 `ResponsesParser.FormatModels` 自行序列化（`{object:"list"}`，形状与 OpenAI List Models 一致；responses 暂无官方模型列表格式，实现独立于 OpenAI 结构体，各协议隔离、可独立演进）。

- 修复用户/管理员 Token 用量统计报错：当分组内 `input_tokens` 合计为 0 时，`cache_hit_rate`（`SUM(cached_tokens) / SUM(input_tokens)`）除零得 NULL，扫描进 `float64` 报 `converting NULL to float64`。统计 SQL（`StatsByUser` / `StatsByAdmin` / `Trend7d`）中 `cache_hit_rate` 统一用 `COALESCE(..., 0)` 兜底为 0，统计接口恢复正常返回。

- 超管模型管理新增「复制」功能：操作列调整为「编辑」按钮 + 「更多」下拉菜单（复制/删除），点击复制后弹出新增模型弹窗并预填该模型的全部配置（provider/model 默认为原值，可修改后再提交新增），提交即调用已有的 `/manager/models/create` 新增模型，名称唯一性由后端校验。纯前端改动，无接口/数据结构变化。

- API Key 支持明文还原查看：`api_keys` 表新增 `key_enc` 列（AES-256-GCM 密文，复用 `service/crypto.go` 加密能力，purpose=`:api-key` 与 TOTP/Provider 隔离）。新建 key 时同时写入哈希（鉴权比对）与密文（可还原）；新增查看接口 `/manager/apikeys/reveal/self`（普通用户）与 `/manager/apikeys/reveal`（超管），解密返回明文。前端 API Key 列表 Key 列后增加复制图标，点击直接复制明文到剪贴板（不弹窗），旧版本创建的 key（`key_enc` 为空）提示无法还原。鉴权链路不变（仍走 `key_hash` 等值查找）。存量库需手动迁移 `ALTER TABLE api_keys ADD COLUMN key_enc TEXT NOT NULL DEFAULT ''`，存量 key 明文已不可还原。

- API Key 模型访问弹窗改为搜索式选择：不再一次性加载全量模型，候选列表默认只展示前 10 条，输入关键词走服务端模糊搜索（300ms 防抖），匹配超过 10 条时提示精确查找；已选模型以可关闭标签常驻顶部展示，勾选状态与搜索结果解耦。配套接口调整：`/manager/apikeys/models/get(/self)` 响应新增 `models` 字段（白名单模型的 id/provider/model 简要信息，供前端常驻展示已选模型）。

- 代理路由组织调整：proxy 的路由注册与 echo 适配（参数提取、响应写入）从 `framework/echo.go` 下沉到 `proxy/router/router.go`（与 `manager/router` 同构），`framework/echo.go` 只保留路由组创建与全局中间件；`EchoProxyDirectResponseWrite` 改为包内未导出的 `echoResponseWrite`。行为不变。

- 新增 `GET v1/models` 模型列表端点：返回当前 Provider 下、且当前 API Key 有权访问的模型列表（数据源为本地 `models` 表，与代理鉴权口径一致，不透传上游、不计费）。框架层注册具体路由（`GET /proxy/:format/:provider/v1/models`，静态段优先于通配符）进入独立入口 `proxy.HandleModels`，链路为 `ParseRequest → AuthKey → ListModels`；非 GET 的同路径请求仍落回转发链路。该端点不写 `request_logs`。响应按路径中的协议格式序列化：OpenAI 为 `{object:"list"}`，Anthropic 为 `{data,first_id,last_id,has_more}`（本地全量返回，不支持分页参数）。配套重构：原 `Auth` handler 拆分为 `AuthKey`（Key/用户校验，两条链路共用）+ `AuthModel`（模型定价 + 白名单校验，仅转发链路），鉴权行为不变。

- 请求头脱敏规则加强：`request_logs.request_headers` 的敏感头识别由精确名单（5 个）改为关键词包含匹配（auth/cookie/key/token/secret/password），覆盖 Azure `api-key`、`X-Auth-Token` 等各类自定义凭证头，防止漏脱。

- API Key 哈希化：`api_keys` 表不再存 key 原文，改存 `key_hash`（SHA-256，鉴权比对）+ `key_show`（展示串 `sk-abc****xyz`，创建时由 `service.ApiKeyShow` 生成）。明文 key 仅创建时返回一次。存量库需手动迁移（RENAME 列 → 逐行回填哈希/展示串 → 重建唯一索引，哈希需外部计算，步骤见 `sql/sqlite.sql` 注释）。鉴权链路（Auth/BudgetCheck/扣费）全部改为哈希或 ID 比对。同步调整：创建 API Key 时名称改为必填（key 不再可查看，名称是识别用途的唯一途径）。

- `request_logs` / `usage_records` 不再存储完整 API Key，改存 `api_key_id`（DB 泄露不再暴露 key 原文与调用记录的关联）。统计接口对外契约不变（筛选传 `api_key_id`、按 key 分组展示脱敏 key + 名称）；已删除的 key 在分组统计中展示为 `#id + 已删除`。存量库需手动迁移（加列 → 回填 → DROP 旧列，SQL 见 `sql/sqlite.sql` 注释，需 SQLite ≥ 3.35）。

- 密钥管理重构：签名密钥（JWT/2FA 票据）与加密密钥（TOTP/Provider 配置落库加密）拆分为两把独立密钥。加载优先级为环境变量（新增 `AIAPI_CRYPTO_SECRET`）> 密钥文件（`<DataDir>/keys/*.key`，0600 权限）> 自动生成并写文件——不设置环境变量也可启动（旧版本缺失环境变量会启动失败，行为变更）。旧部署升级时加密密钥自动从 `AIAPI_JWT_SECRET` 播种，历史密文（2FA 密钥、provider 配置）保持可解，请保持原环境变量至少完成一次启动。

- 安全加固：Provider 配置（含上游 API Key）落库改为 AES-256-GCM 加密（加密密钥独立于签名密钥按用途派生，新增 `service/crypto.go` 通用加密模块，与 TOTP 密钥加密用途隔离）。存量明文配置读取时自动兼容，管理端重新保存后转为密文（渐进迁移，无需 DDL）。
- 修复代理日志泄露完整 API Key：`budget.go` 余额不足错误不再打印 key 原文，改为打印 `apiKeyId`（Auth 阶段已注入 ctx，可定位问题且不落任何 key 材料）。

- 新增 TOTP 两步验证（2FA）：管理台用户可在个人设置页自助绑定 Authenticator（扫码/手动密钥 + 首个验证码确认），开启后登录需密码 + 6 位动态码两步。TOTP 密钥 AES-256-GCM 加密入库（密钥派生自 `AIAPI_JWT_SECRET`）；密码验证通过后仅签发 5 分钟 pending 票据，同一票据验证码连续错 5 次作废；关闭 2FA 需校验密码。新增接口 `/manager/login/2fa`、`/manager/2fa/{setup,confirm,disable}/self`；`users` 表新增 `totp_secret` 列（存量库需 `ALTER TABLE` 迁移，并重跑 `sql/init-data.sql` 补普通用户角色的 2FA 接口权限）；新依赖 `github.com/pquerna/otp`。

- 移除 pprof 调试接口：`--add-pprof` 启动参数与 `/debug/pprof/*` 路由全部删除（此前开启后无任何鉴权，会公开 goroutine/堆等运行时信息）。README 中相关说明同步移除。

### 2026-07-27

- 修复前端日历面板英文显示：`n-config-provider` 补 `:date-locale="dateZhCN"`，naive-ui 的日期/日历组件文案由独立的 date-locale 控制，此前只设了 `locale="zhCN"` 导致月份/星期仍显示英文。

- manager 业务错误码收敛：`manager/base/bizcode.go` 从 19 个码精简到 7 个（`CodeSuccess`/`CodeUnknown`/`CodeBadRequest`/`CodeUnauthorized`/`CodeForbidden`/`CodeNotFound`/`CodeTokenExpired`）。只保留有消费方按 code 分支（前端唯一依赖 `CodeTokenExpired` 1016 触发 `/refresh`）或 HTTP 状态语义不同的码；其余业务错误（已存在/不存在/额度不足/密码错误等）统一用通用码 + 中文错误信息区分，不再为每种业务定义独立码。删除死码 `CodeUserDisabled`。
  - handler 错误信息全部中文化；DB 等内部错误统一返回预置实例 `base.ErrInternal`（"系统繁忙，请稍后重试"），替代约 60 处 `NewBizError(CodeUnknown, ...)` 样板。
  - 新增便捷构造 `base.ErrBadReq(msg)` / `base.ErrNotFound(msg)`。
  - 登录失败（账号错/密码错/禁用）HTTP 状态由 401 调整为 400，防枚举文案不变；前端对 `/login` 本就排除 refresh 重试，无影响。
  - `CodeTokenExpired` 编号 1016 不变，前端零改动。

### 2026-07-26

- 列表接口全量分页：`api_key`、模型、提供商、用户列表均改为分页查询，避免全量数据查询过多导致数据库卡死。
  - `ListApiKeySelf`/`ListApiKeyAdmin` 返回 `*base.PageResult[apiKeyItem]`
  - `ListModelsAdmin` 返回 `*base.PageResult[modelItem]`（`ListModels` 保持全量，用于下拉框填充）
  - `ListProviders` 返回 `*base.PageResult[providerItem]`
  - `ListUsers` 返回 `*base.PageResult[userItem]`，关键字过滤从 Go 内存下推到 SQL `WHERE ... LIKE`
  - 新增 `/users/get` 接口查询单个用户信息（替代全量 `listUsers` 查找）
  - 前端各列表页加远程分页器（`n-data-table` remote 模式）
- store 包重构完成：store 回归纯 SQL 包装层，所有事务编排、跨表组装、业务判断下沉到 `service/`。涉及 A1-A6（事务编排）、B1-B2（跨表组装）、C1-C3（业务判断）共 11 项。
- `manager/service/` 迁移到顶层 `service/` 包，manager 与 proxy 共用业务层，消除 proxy 自建 service 的重复。
- store 按表/领域拆分 Store：`ModelAccessStore` 独立于 `ModelStore`（操作 `api_keys.model_policy` + `apikey_model_access` 白名单）。
- 删除 `inList` 工具函数：QueryBuilder 内置 `sqlx.In` 自动展开 `IN (:ids)`，不手动拼接占位符。
- 更新 `AGENTS.md`/`README.md` 架构说明与开发规则。

### 2026-07-25

- 新增通用分页能力（Session 状态 + 显式控制）：`store/base/page.go` 提供 `PageContext`，`manager/base/page.go` 提供 `PageReq`/`PageResult[T]`。handler 创建 `store.PageContext`，用 `store.C().SetPage(pc).Charge().List(...)` 链式调用，`QueryBuilder.Select` 检测到后自动拦截：先 `SELECT COUNT(*) FROM (<原SQL>) t` 查总数写回 `pc.Total`，再追加 `LIMIT ? OFFSET ?` 查当前页。非事务单次分页不用 `ClearPage`（Session 用完即弃）；事务内用 `SetPage`/`ClearPage` 显式控制。store 方法只需写普通 `Select`，未 `SetPage` 时退化为普通查询。`page` 1-based，`page_size` 默认 20、上限 100。分页类型分层：`manager/base` 定义 API 层 `PageReq`/`PageResult`，`store` 暴露内部 `PageContext`，handler 不直接 import `store/base`。
- 充值流水接口改为分页：`/manager/recharge/records`、`/manager/recharge/records/self`、`/manager/recharge/records/list` 入参加 `{page, page_size}`，返回 `{items, total, page, page_size}`。
- 统一 Session 架构：`store.T` 包级函数改为 `Session.T(fn)` 方法，调用方式 `store.C().T(fn)`。事务在当前 Session 上切换 tx，嵌套调用复用当前 tx（不开新事务，保证原子性）。`store.C()` 无参，Session 保存 `tx`/`page` 状态字段，不再用 `context.Value` 传业务参数（tx、分页），context 回归“请求级跨边界数据”本职。
- 重构充值业务：`store/charge.go` 的 `RechargeWithRecord`（读余额→加余额→写流水事务编排）下沉到 `service/charge.go` 的 `ChargeService`；`manager/handler/recharge.go` 改为调 service，不再自写事务。store 回归纯 SQL 包装。service 新增哨兵错误 `ErrUserNotFound`。
- 修复并发充值流水不准：原「先 SELECT 读余额再 UPDATE」是读后写，快照读拿旧值会丢更新。改为「先 UPDATE 加余额（行锁）→ 同事务内 SELECT 新余额 → before 由 after 反推」，跨 SQLite/MySQL/PostgreSQL 通用。
- 充值接口 self/admin 合并：`RechargeSelf` 设当前用户 ID 后委托 `Recharge`，`RechargeRecordsSelf` 同样委托 `RechargeRecords`，消除重复校验逻辑。
- 明确后端四层职责与事务边界（更新 `AGENTS.md`）：
  - `store/` 为纯 SQL 包装层，只做单表读写，不写跨表编排与业务判断。
  - `service/` 承载跨 handler 复用业务逻辑与多表事务编排（manager 与 proxy 共用）；`manager/handler/` 只做 HTTP 适配，不直接写事务。
  - 事务体用 `store.T(fn)` 包裹，写在 `service/`（manager 与 proxy 共用业务层），不写在 `store/` 内。
  - 新增 §2.4 并发安全规则：金额/计数类写操作必须「先 UPDATE 后事务内 SELECT」，禁止读后写、禁止依赖 `RETURNING`/`FOR UPDATE`（跨库不通用）。
  - 新增 §2.5 注释风格：store 方法注释只说做什么，不描述事务用法/调用时机。
  - 新增 §7.3 self/admin 合并模式约定。
  - 同步补充分层表、handler/service 边界、新增 Manager API / 数据库实体的步骤要求。

### 2026-07-24

- 修复重启服务后登录态丢失问题：refresh cookie 的 `Secure` 标记原硬编码为 `true`，导致 HTTP（非 HTTPS）部署下浏览器拒绝存储/发送该 cookie，重启后 access JWT 失效触发 refresh 时拿不到 cookie，被迫重登。现改为跟随当前请求是否 HTTPS 动态判定（`c.Scheme()=="https"`）：HTTPS 直连或反代透传 `X-Forwarded-Proto` 时开启 `Secure`，HTTP 下关闭，本地开发与纯 HTTP 部署不再丢登录态。
- 新增双 token 登录机制（access JWT + refresh 轮换 + 重用检测），替换原单 token 方案，适配公网部署。
  - **access JWT**：HS256 自实现，15min 有效期，无状态不落库，经 `Authorization: Bearer` 头传递，前端存内存（不放 localStorage 防 XSS）。
  - **refresh token**：随机串哈希后存 `user_sessions` 表（明文不落库），HttpOnly Secure cookie `refresh_token` 传递（Path=`/manager`，SameSite=Lax），滑动 7 天 / 绝对 30 天。
  - **轮换**：每次 `/manager/refresh` 删旧发新（同 family），滑动续期顺延但不超绝对上限。
  - **重用检测**：旧 refresh token 再次被使用时吊销整个 family，强制重登（业务码 1017）。
  - 改密 / 禁用用户 / 重置密码后吊销该用户所有会话，即时失效。
  - 账号不存在 / 禁用 / 密码错统一返回相同文案，防账号枚举。
  - 新增 `service` 包封装登录业务（`SessionService`），handler/middleware 仅做 HTTP 适配。
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

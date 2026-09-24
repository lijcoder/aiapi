# 领域词表

> 一个概念一个规范词。写代码、文档、提交信息时用这里的词；实现细节见词条指向的文件。
> 新增领域词时补进本表，避免同一个东西在代码里出现三四种叫法。

## 路由与协议

**provider** — 上游服务商配置，`providers.type` 是它在 URL 里的标识（`/proxy/<provider>/...`）。配置字段 `domain` + `headers` 见 [README「Provider 配置」](../README.md#provider-配置)。

**format** — 客户端使用的协议格式，决定用哪个解析器：`openai` / `anthropic` / `openai-responses`。**`gemini` 已定义但未实现**（`GetParser` 返回 `nil`）。见 [`../parser/AGENTS.md`](../parser/AGENTS.md)。

**链路**（pipeline） — 一个入口从解析到落库的 handler 序列，组装在 `proxy/direct.go`。转发链路与元数据端点（`v1/models`）是两条独立链路。

**元数据端点** — 不访问上游的端点（目前只有 `GET .../v1/models`），不计费、不写 `request_logs`。

## 模型与计费

**model** — 对用户可见的模型名：客户端填它、`v1/models` 返回它、鉴权/白名单/计费/日志都用它。

**provider_model** — 转发时替换请求体顶层 `model` 的上游模型名；空串表示与 `model` 一致（`service.UpstreamModelName`）。

**pricing_config** — 模型的计费配置（JSON，`version: 2`），含默认价与可组合的分段规则。单位是**元/百万 token**。未配置或配置非法的模型不会被转发。

**pricing_snapshot** — 写进 `usage_records` 的计费快照：命中的规则名、条件、实际单价、完整配置、时区与请求开始时间。用于事后审计，不受后续改价影响。

**cache hit / cache miss** — 输入 token 的两种计价口径：`input_cache_hit`（命中提示缓存）与 `input_cache_miss`（其余输入）。Anthropic 的缓存创建与缓存读取会并入完整输入量后再按这两种价计费。

## 认证与权限

**API Key** — 调用代理接口的凭证（`sk-` + 64 hex）。库里存三份：`key_hash`（SHA-256，鉴权比对）、`key_enc`（AES-256-GCM 密文，可还原查看）、`key_show`（展示串 `sk-abc****xyz`）。**与上游凭证无关**——上游 key 配在 `provider.config.headers` 里。

**上游凭证** — 转发时发给上游的鉴权头，只来自 `providers.config.headers`（客户端请求头不转发）。常见形态是 OpenAI 的 `Authorization`、Anthropic 的 `x-api-key`。

**model_policy** — API Key 的模型访问策略：`all`（放行全部）或 `whitelist`（按 `apikey_model_access` 白名单）。白名单无权时对外统一表现为"模型不存在"。

**access JWT / refresh token** — 双 token 登录态。access JWT 无状态、短时效、走 `Authorization` 头；refresh token 落 `user_sessions`（只存哈希）、走 HttpOnly cookie。见 [`security.md`](security.md)。

**JWTSecret / CryptoSecret** — 两把分离的密钥：前者签名（JWT、2FA 票据），后者派生 AES 密钥用于加密落库字段（TOTP 密钥、provider 配置、API Key 原文）。轮换后果见 [`../README.md`](../README.md#环境变量)。

**role_permission** — 接口级权限：`(role_id, entity='API', action='*', value=<接口路径>)`，`value='*'` 为超管通配。新增非超管接口必须同步 `sql/init-data.sql` 的种子。

**menus / role_menus** — 侧栏菜单数据与角色授权。菜单可见性与接口权限是两套，新增页面要同时补。

## 数据与运行

**schema version** — 数据库结构版本，唯一权威是 `constant.SchemaVersion`，与库里 `schema_meta` 表记录的版本比对，不一致拒绝启动。用元数据表而非 SQLite 专有的 `PRAGMA user_version`，以兼容 MySQL / PostgreSQL。迁移见 [`../sql/migrations/README.md`](../sql/migrations/README.md)。

**usage_records / request_logs** — 两张不同的表：前者是**计费用量**（仅成功请求，`status_code < 300`），后者是**请求日志**（转发链路无论成败都记）。别混用。

**已知偏离** — 文档规则与代码现状不一致、且短期内不打算改的地方，登记在 [`architecture.md`](architecture.md) 的「已知偏离」一节。写代码遇到"规则说 X、代码是 Y"时先查这里。

# aiapi — 大模型 API 网关

轻量级大模型 API 反向代理：用统一入口接入多种客户端协议（OpenAI / Anthropic / OpenAI Responses），按配置路由到上游 Provider，透传响应（含 SSE 流式），并记录 Token 用量、请求日志与计费。

自带 Web 管理台（Provider / 模型 / 计费 / API Key / 用户 / 充值 / 统计），前端编译后嵌入 Go 二进制，**单文件部署**。

## 功能特性

- **多协议接入**：`openai`（Chat Completions）、`anthropic`（Messages）、`openai-responses`（Responses API），各自独立解析与用量口径
- **多上游路由**：URL 中指定 Provider，请求原样转发，响应（含 SSE）透明透传
- **鉴权与额度**：管理台签发 API Key，支持 Key 级启停、额度、模型白名单与用户余额校验
- **模型名解耦**：对用户可见的模型名与发往上游的模型名分离，同一上游模型可按不同定价映射成多个别名
- **用量与计费**：记录 input / output / 缓存命中 / 推理 tokens 与首 token、端到端耗时，按可组合的分段规则计费，每笔用量保存计费快照供审计
- **请求日志**：记录请求参数、响应状态、耗时与错误，管理台可直接查看最近日志
- **管理台**：接口级权限 + 动态菜单，双 token 登录与可选的 TOTP 两步验证

## 架构概览

```
                    ┌────────────────────────────┐
 客户端 ──────────▶ │  /proxy/:provider/:format/* │
 (OpenAI/Anthropic/ │         Echo v4            │
  Responses)        └─────────────┬──────────────┘
                                  ▼
                     Pipeline: ParseRequest → AuthKey → AuthModel
                     → BudgetCheck → LoadConfig → RewriteModel
                     → Forward → ParseUsage → Record (+ Log)
                                  │
                     ┌────────────┴────────────┐
                     ▼                         ▼
             上游 Provider（HTTP/SSE）    SQLite（用量/日志/配置/用户）
```

一个请求的处理过程：解析客户端协议 → 校验 API Key 与用户 → 校验模型可用性与定价 → 校验余额 → 载入 Provider 配置 → 把模型名改写为上游模型名 → 转发并透传响应 → 解析用量 → 记录用量并扣费 → 写请求日志。

### 支持的协议

| `:format` | 说明 | 客户端鉴权头 | 上游端点 |
|-----------|------|--------------|----------|
| `openai` | OpenAI Chat Completions 兼容格式 | `Authorization: Bearer <key>` | 由 URL 通配段决定，如 `v1/chat/completions` |
| `anthropic` | Anthropic Messages 格式 | `x-api-key: <key>`（也接受 `Authorization`） | 如 `v1/messages` |
| `openai-responses` | OpenAI Responses API 格式 | `Authorization: Bearer <key>` | 如 `v1/responses` |
| `gemini` | **未实现**：解析器返回 `nil`，请求最终以 401 `missing api key` 失败 | — | — |

> 分层、边界、事务与并发语义见 [`docs/architecture.md`](docs/architecture.md)；领域词见 [`docs/glossary.md`](docs/glossary.md)；接入新协议见 [`parser/AGENTS.md`](parser/AGENTS.md)。

## 快速开始

### 1. 环境要求

- **Go 1.25+**（以 [`go.mod`](go.mod) 为准）
- **Node.js 18 / 20 / 22+**：仅在需要自行构建管理台前端时要求（Vite 6）
- **SQLite 3**：命令行工具 `sqlite3`，用于初始化数据库
- Linux / macOS / Windows（推荐 Linux）

### 2. 编译

```bash
make install        # go mod tidy
make build-all      # 构建前端 + 编译后端（部署与首次构建用这个）
make build          # 只编译后端，复用本地已有的 frontend/dist
```

产物是当前目录下的 `./aiapi`（管理台前端通过 `go:embed` 嵌入）。

> ⚠️ 仓库只跟踪 `frontend/dist/index.html`，`assets/` 与 `favicon.svg` 都不入库。**全新 clone 后直接 `make build`，二进制里的管理台会因为静态资源缺失而白屏**（请求 JS/CSS 会落到 SPA fallback 返回 HTML）。首次构建与部署请用 `make build-all`。

### 3. 初始化数据库

**应用不会自动建库或迁移**，首次部署需手动初始化：

```bash
mkdir -p ~/.aiapi/db
sqlite3 ~/.aiapi/db/aiapi.db < sql/sqlite.sql      # 建表
sqlite3 ~/.aiapi/db/aiapi.db < sql/init-data.sql   # 角色、权限、菜单等种子数据（可重复执行）
```

创建管理员账号（`password` 需填 bcrypt 哈希）：

```bash
# 生成哈希（任选一种）
htpasswd -bnBC 10 "" '你的密码' | tr -d ':\n'      # 需要 apache2-utils / httpd-tools
python3 -c "import bcrypt;print(bcrypt.hashpw(b'你的密码', bcrypt.gensalt()).decode())"
```

```sql
INSERT INTO users (name, account, password, unlimited, enabled)
VALUES ('管理员', 'admin', '<bcrypt-hash>', 1, 1);
INSERT INTO user_roles (user_id, role_id)
VALUES ((SELECT id FROM users WHERE account='admin'), 1);
```

### 4. 启动

```bash
./aiapi                         # 默认监听 :8888，数据目录 ~/.aiapi
./aiapi --port 9000             # 指定端口
./aiapi --address 127.0.0.1:    # 只监听本机（注意结尾的冒号）
./aiapi --data-dir /var/lib/aiapi
```

> `--address` 是**地址前缀**，最终监听地址 = `--address` + `--port`，所以必须带结尾冒号；写 `--address 127.0.0.1` 会拼出 `127.0.0.18888` 导致监听失败。

启动时会打印监听端口与数据目录；若数据库文件不存在，进程会在打印前直接报错退出（错误信息含缺失路径）。

### 5. 第一个请求

在管理台创建 Provider、模型、API Key 后（见 [管理台](#管理台)），即可调用：

```bash
curl http://localhost:8888/proxy/<provider>/openai/v1/chat/completions \
  -H "Authorization: Bearer <你的 API Key>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"你好"}]}'
```

无 API Key 时任一请求都会返回 401，说明服务已就绪。

### 6. 打开管理台

浏览器访问 `http://localhost:8888/`，用上面创建的管理员账号登录。

## 调用代理 API

### 路由与鉴权

```
ANY /proxy/:provider/:format/*
GET /proxy/:provider/:format/v1/models
```

- `:provider`：Provider 的 `type`（管理台配置的唯一标识，如 `openai`）
- `:format`：客户端协议格式，见[支持的协议](#支持的协议)
- `*`：转发到上游的路径。上游完整 URL = `Provider.config.domain` + `/` + 该路径，因此 `domain` 不要以 `/` 结尾

鉴权头按协议不同（见上表）：`openai` / `openai-responses` 用 `Authorization: Bearer <API Key>`，`anthropic` 用 `x-api-key`。这里传的是 **aiapi 签发的 Key**，不是上游 Key。

**请求头不会透传给上游**：上游请求头只取 Provider 配置里的 `config.headers`（含 `Content-Type` 与上游鉴权头）。客户端请求头仅用于提取 API Key 和写日志。

### 调用示例

```bash
# OpenAI Chat Completions
curl http://localhost:8888/proxy/openai/openai/v1/chat/completions \
  -H "Authorization: Bearer <key>" -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}],"stream":false}'

# Anthropic Messages
curl http://localhost:8888/proxy/anthropic/anthropic/v1/messages \
  -H "x-api-key: <key>" -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-5","max_tokens":1024,"messages":[{"role":"user","content":"hi"}]}'

# OpenAI Responses
curl http://localhost:8888/proxy/openai/openai-responses/v1/responses \
  -H "Authorization: Bearer <key>" -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","input":"hi","stream":false}'
```

### 流式（SSE）

把请求体的 `stream` 置为 `true`，响应即为 SSE 事件流，内容原样透传（Anthropic 的 `message_delta`、Responses 的 `response.*` 等事件不改写）。服务端不缓冲、逐块转发，**不设写超时与上游总超时**，长流不会被切断；客户端断开时通过请求 context 取消上游请求。用量在响应结束后单独解析。

### 获取可用模型列表

`GET .../v1/models` 是本地元数据端点：不访问上游、不计费、不写 `request_logs`，返回当前 API Key 有权访问的模型，响应格式按 `:format` 输出。

```bash
curl http://localhost:8888/proxy/openai/openai/v1/models -H "Authorization: Bearer <key>"
# → {"object":"list","data":[{"id":"gpt-4o-mini","object":"model","created":1715000000,"owned_by":"openai"}]}
```

Anthropic 格式返回 Anthropic 的模型列表结构（`{data:[{type,id,display_name,created_at}], first_id, last_id, has_more}`）；`openai-responses` 沿用 OpenAI 形状。该端点不校验 Provider 是否存在或启用，并忽略 query 参数。

### 模型名与上游模型名

模型配置有两个名字，在管理台「模型管理」中维护：`model`（对用户可见：客户端调用时填、`v1/models` 返回、鉴权/白名单/计费/日志都用它）与 `provider_model`（发往上游：转发时替换请求体顶层的 `model`，留空表示与 `model` 相同）。

同一 `provider_model` 可被多个别名复用（例如按不同定价分成多档）；上游响应里的 `model` 不改写；`request_logs.request_body` 记录的是**实际发往上游**的请求体，便于对照排查。

### 错误响应

代理链路遵循客户端协议返回错误，业务码集中在 [`proxy/types/bizcode.go`](proxy/types/bizcode.go)：

| HTTP | 场景 |
|------|------|
| 401 | 缺少/无效 API Key，或用户被禁用 |
| 402 | 余额或 Key 额度不足 |
| 404 | Provider 不存在、模型不存在、模型未配置或定价非法、Key 无该模型权限（白名单无权对外统一表现为"模型不存在"） |
| 5xx | 上游或内部错误（错误详情写入 `request_logs`） |

## 管理台

浏览器访问 `http://<host>:<port>/`。侧栏菜单由后端按当前用户权限动态下发，普通用户与超管看到不同页面。

- **登录态**：账号密码登录，access JWT（15 分钟，走 `Authorization` 头）+ refresh token（HttpOnly cookie，滑动 7 天 / 绝对 30 天，轮换 + 重用检测）。改密/禁用/重置密码会吊销该用户全部会话。
- **两步验证（2FA）**：在「个人设置」自助开启，扫码绑定后登录需再输验证码；TOTP 密钥加密落库。
- **权限与菜单**：接口级权限（`role_permission` 按路径授权，`value='*'` 为超管通配）+ 菜单表驱动侧栏；`/self` 后缀为普通用户自助接口。

**完整的接口清单（含请求字段与调用示例）见 [`docs/api.md`](docs/api.md)**；认证、密钥与权限模型见 [`docs/security.md`](docs/security.md)。

## 配置参考

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--address` | `:` | 监听地址**前缀**，最终地址 = `--address` + `--port`，必须带结尾冒号（如 `127.0.0.1:`） |
| `--port` | `8888` | 监听端口 |
| `--data-dir` | `~/.aiapi` | 数据根目录 |

### 环境变量

| 变量 | 说明 |
|------|------|
| `AIAPI_JWT_SECRET` | 签名密钥（access JWT / 2FA 票据），≥32 字节 |
| `AIAPI_CRYPTO_SECRET` | 加密密钥（派生 AES 密钥，加密 TOTP 密钥、Provider 配置、API Key 原文），≥32 字节 |

未配置时会自动生成随机密钥写入 `<数据目录>/keys/`（文件 0600、目录 0700）并在后续启动复用。**轮换密钥的后果**与数据目录结构见 [`docs/deployment.md`](docs/deployment.md)。

### Provider 配置

Provider 决定请求转发到哪里。`config.headers` 是**唯一**发往上游的请求头来源，因此必须包含上游鉴权头与 `Content-Type`。

```bash
curl -X POST http://localhost:8888/manager/providers/create \
  -H "Authorization: Bearer <access_token>" \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "openai",
    "domain": "https://api.openai.com",
    "headers": {
      "Authorization": ["Bearer sk-xxxxxxxx"],
      "Content-Type": ["application/json"]
    }
  }'
```

| 字段 | 说明 |
|------|------|
| `type` | Provider 唯一标识，即 URL 中的 `:provider`；创建后不可改 |
| `domain` | 上游基础域名，**不要以 `/` 结尾**（上游 URL = `domain` + `/` + 通配路径） |
| `headers` | 转发到上游的请求头，值为字符串数组 |
| `enabled` | 是否启用（管理台可切换） |

`config` 落库时加密存储（兼容历史明文），接口返回脱敏后的展示值。

## 部署与运维

数据库为单实例 SQLite，数据目录默认 `~/.aiapi`（`db/`、`logs/`、`keys/` 三个子目录）；公网部署需在前置反代终结 TLS，并**必须转发 `X-Forwarded-Proto`**，否则 refresh cookie 不带 `Secure`。

升级不会自动迁移：先备份，再按 [`sql/migrations/README.md`](sql/migrations/README.md) 执行缺失的迁移。数据库结构版本记录在 `schema_meta` 表，启动时与二进制比对，**库比二进制旧或新都会拒绝启动**。

完整的反代配置、生产检查清单、日志轮转、备份、升级流程与数据表说明见 [`docs/deployment.md`](docs/deployment.md)。

## 已知限制

- **单实例部署**：使用本地 SQLite，多实例无法共享数据；TOTP 失败计数存在进程内存中（会话本身落库）
- **登录接口无限流**：密码错误无锁定与速率限制，公网部署建议在反代层做限流与封禁
- **请求/响应体全量入库**：`request_logs` 保存完整 body 且该表无索引，暂无截断与脱敏开关，注意隐私合规与表体积
- **余额并发可透支**：先放行后扣费，高并发下余额可能变为负值
- **Gemini 协议未实现**：`format=gemini` 的请求会被拒绝（表现为 401 `missing api key`）
- **模型的上限与模态字段不参与校验**：`max_context_tokens`、`max_completion_tokens`、`supports_*` 目前只用于配置与展示
- **无 CI**：测试与构建需本地执行 `make check` / `make build`

其余待优化项见 [`TODO.md`](TODO.md)。

## 开发

开发规则（红线、边界、自检）见 [`AGENTS.md`](AGENTS.md)；**进入某个包工作时该目录的 `AGENTS.md` 会自动加载**（`frontend/` / `parser/` / `proxy/` / `manager/` / `store/` / `sql/`）。设计理由见 [`docs/decisions/`](docs/decisions/README.md)，变更记录见 [`CHANGELOG.md`](CHANGELOG.md)。

```bash
make hooks       # 每个 clone 跑一次：启用 .githooks/pre-commit，提交前自动跑 make check
make check       # 提交前必跑：格式 + go vet + go test（含架构、权限种子、文档、schema 版本门禁）
make test        # 只跑测试
make dev-ui      # 前端开发模式（3000 端口，代理 /manager 到后端）
make build-all   # 构建含前端的完整二进制
```

## 许可证

仓库暂未包含 LICENSE 文件。

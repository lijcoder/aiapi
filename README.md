# aiapi - 大模型 API 网关

一个轻量级 OpenAI 兼容格式的大模型 API 反向代理，支持多上游 Provider 路由、Token 用量统计、SSE 流式透传与请求日志。

## 功能特性

- **OpenAI 兼容代理**：客户端使用标准 OpenAI API 格式请求，统一转发给兼容的上游提供商
- **多上游路由**：可将请求转发到任意配置的 OpenAI 兼容 Provider（如 OpenAI、Azure、OneAPI 等）
- **SSE 流式透传**：支持流式聊天补全响应的透明转发
- **Token 用量统计**：自动记录 input / output tokens
- **请求日志**：保存每次请求的参数、响应状态、耗时与错误信息
- **Provider 管理**：内置 CRUD 接口动态管理上游配置
- **API Key 管理**：校验调用方身份，支持启用/禁用；key 原文不落库（SHA-256 哈希存储，展示为 `sk-abc****xyz` 头尾片段），明文仅创建时返回一次
- **管理台**：内置 Web 管理界面，支持双 token 安全登录、TOTP 两步验证（2FA）、充值、流水查询

## 架构概览

```
┌─────────┐      ┌──────────────────────┐      ┌─────────────┐      ┌────────────────────┐
│ Client  │ ──▶  │  /proxy/openai/...   │ ──▶  │  Pipeline   │ ──▶  │  OpenAI 兼容上游   │
│         │      │       Echo v4        │      │  handlers   │      │     Provider       │
└─────────┘      └──────────────────────┘      └─────────────┘      └────────────────────┘
                                                        │
                                                        ▼
                                                 ┌─────────────┐
                                                 │ SQLite 存储  │
                                                 │ usage_logs  │
                                                 │ request_logs│
                                                 └─────────────┘
```

### 支持的请求格式

本项目当前仅代理 **OpenAI 兼容格式** 的请求与响应。`:format` 参数目前固定为 `openai`，`:provider` 为配置的上游 Provider 标识。

| 客户端格式 | 上游 Provider | 说明 |
|------------|-------------|------|
| openai     | openai      | 转发至 OpenAI 兼容上游 |
| openai     | azure-openai| 转发至 Azure OpenAI 兼容接口 |
| openai     | custom      | 转发至任意 OpenAI 兼容的第三方网关 |

未来如需支持其他客户端协议（如 Gemini/Anthropic），需在 `parser/` 层扩展实现转换。

### 分层架构

| 层 | 目录 | 职责 |
|----|------|------|
| 入口层 | `main.go` | 参数解析、日志/数据库初始化、启动服务 |
| 框架适配层 | `framework/echo.go` | HTTP 路由注册、请求参数提取、响应写入 |
| 代理编排层 | `proxy/direct.go` | 组装并执行请求处理 Pipeline |
| 代理处理层 | `proxy/handler/*.go` | 单一职责 handler，完成代理业务逻辑（鉴权、计费、转发等） |
| 协议解析层 | `parser/*.go` | 不同厂商的请求/响应解析与 Key 提取 |
| 后台接口层 | `manager/handler/*.go` | 后端给前端的接口入口，只做 HTTP 适配（参数校验、调 service、组装响应） |
| 业务服务层 | `service/*.go` | 跨 handler 复用的业务逻辑、多表事务编排、业务判断（manager 与 proxy 共用） |
| 后台中间件 | `manager/middleware/*.go` | 登录态校验、接口级权限判定 |
| 数据持久层 | `store/*.go` | 纯 SQL 包装层，单表/单条数据读写，不写事务编排与业务判断 |
| 通用工具层 | `constant/`、`log/`、`proxy/sse/` | 常量、日志格式化、SSE 包装 |

## 快速开始

### 环境要求

- Go 1.22+
- 操作系统：Linux / macOS / Windows（推荐 Linux/macOS）

### 编译

```bash
go build -o aiapi .
```

### 启动

密钥加载优先级：**环境变量 > 密钥文件 > 自动生成**。
不设置任何环境变量也能启动——首次启动自动生成随机密钥写入 `<数据目录>/keys/`（0600 权限），后续启动直接复用：

```bash
# 默认端口 8888，数据目录 ~/.aiapi
./aiapi

# 如需显式管理密钥（生产推荐），通过环境变量提供（≥32 字节）：
export AIAPI_JWT_SECRET="your-jwt-secret-at-least-32-bytes"       # 签名密钥（access JWT / 2FA 票据）
export AIAPI_CRYPTO_SECRET="your-crypto-secret-at-least-32-bytes" # 加密密钥（TOTP / Provider 配置落库加密）
```

密钥说明：
- **两把钥匙职责分离**：签名密钥轮换代价低（用户重登即可）；加密密钥轮换会使历史密文（2FA 密钥、provider 配置）无法解密，需重新绑定/保存
- 密钥文件为 `<数据目录>/keys/jwt.key` 与 `crypto.key`；**备份数据库时必须排除 `keys/` 目录**，且不要将数据目录提交到任何仓库/网盘
- 旧版本升级：首次启动时加密密钥自动从 `AIAPI_JWT_SECRET` 播种（历史密文保持可解），此后两把钥匙独立演进；请保持原环境变量至少完成一次启动

# 指定端口
./aiapi --port 8888
```

### 测试代理

```bash
curl http://localhost:8888/proxy/openai/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### 管理台

`frontend/` 是 Vue 3 + Vite + Naive UI 管理台前端，编译后通过 Go `embed` 嵌进二进制。

#### 两步验证（2FA / TOTP）

管理台支持 TOTP 两步验证（Google Authenticator / 1Password / 微软 Authenticator 等），在个人设置页自助开启：

1. 点击「开启」→ 用 Authenticator App 扫描二维码（或手动输入 Base32 密钥）
2. 输入 App 显示的 6 位验证码完成绑定
3. 之后登录需在密码验证后再输入一次验证码

安全设计：
- TOTP 密钥经 AES-256-GCM 加密后入库（加密密钥独立于签名密钥，见「启动」一节密钥说明），数据库泄露不直接暴露 2FA
- 密码验证通过后只签发 5 分钟有效的 pending 票据，同一票据验证码连续错 5 次即作废
- 关闭 2FA 需重新校验登录密码

存量数据库迁移：
```bash
sqlite3 ~/.aiapi/aiapi.db "ALTER TABLE users ADD COLUMN totp_secret TEXT NOT NULL DEFAULT '';"
sqlite3 ~/.aiapi/aiapi.db < sql/init-data.sql   # 补普通用户角色的 2FA 接口权限
```

```bash
# 开发模式（前端热更新）
make dev-ui     # → http://localhost:3000（自动代理 API 到 8887）

# 生产构建（前端嵌入 Go 二进制）
make build-all  # → ./aiapi 一个文件启动，浏览器打开 http://localhost:8887
```

## 配置 Provider

### 新增 Provider

```bash
curl -X POST http://localhost:8888/manager/providers \
  -H "Content-Type: application/json" \
  -d '{
    "type": "openai",
    "config": {
      "domain": "https://api.openai.com",
      "headers": {
        "Authorization": "Bearer sk-xxxxxxxx"
      }
    },
    "enabled": 1
  }'
```

### 列出所有 Provider

```bash
curl http://localhost:8888/manager/providers
```

### 查询指定 Provider

```bash
curl http://localhost:8888/manager/providers/openai
```

### 删除 Provider

```bash
curl -X DELETE http://localhost:8888/manager/providers/openai
```

### Provider 配置字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | Provider 唯一标识，URL 中的 `:provider` |
| `config` | object | 上游配置 |
| `config.domain` | string | 上游基础域名，如 `https://api.openai.com` |
| `config.headers` | object | 转发到上游时附加的 HTTP 头 |
| `enabled` | int | 是否启用，`1` 启用，`0` 禁用 |

## 反向代理路由

```
/proxy/openai/:provider/*
```

- `:format`：当前固定为 `openai`
- `:provider`：上游 Provider 的 `type` 字段
- `*`：上游路径，例如 `v1/chat/completions`

### 示例

```bash
curl http://localhost:8888/proxy/openai/openai/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "hi"}],
    "stream": false
  }'
```

### 数据存储

默认使用 SQLite，数据库文件位于 `~/.aiapi/aiapi.db`。

> 双 token 登录机制上线后，存量库需迁移 `user_sessions` 表（加 `family_id` / `absolute_expires_at` / `ua` / `ip` 列、建 family 索引、清空旧 session），详见 `sql/sqlite.sql` 内注释。

### 核心数据表

- `providers`：上游提供商配置
- `api_keys`：调用方 API Key（存 `key_hash` + `key_show`，不存原文）
- `usage_records`：Token 用量记录
- `request_logs`：请求日志与错误信息
- `users` / `roles` / `user_roles` / `role_permission` / `user_sessions`：用户/角色/权限/会话
- `menus` / `role_menus`：菜单与角色菜单关联
- `recharge_records`：充值流水
- `models`：模型配置

可通过 `sql/sqlite.sql` 查看完整 DDL，初始数据参考 `sql/init-data.sql`。

## 日志与调试

### 应用日志

默认输出到标准输出与 `~/.aiapi/logs/app.log`，支持按大小/时间轮转。

## 开发扩展

- 新增上游响应解析逻辑：参考 `parser/openai.go` 与 `parser/anthropic.go`
- 新增 Pipeline 处理步骤：参考 `proxy/handler/` 目录
- 新增管理接口：`manager/handler/`（HTTP 适配）+ `service/`（业务逻辑）+ `manager/router/router.go`（路由注册）
- 新增业务逻辑（事务编排、跨表组装、业务判断）：写在 `service/`，用 `store.C().T(fn)` 包裹事务
- 新增数据表：`sql/sqlite.sql`（DDL）+ `store/model/models.go`（模型）+ `store/`（纯 SQL Store）
- 开发规则与架构约束详见 `AGENTS.md`

## 后台管理 API

`/manager` 下提供账号密码登录、权限与充值管理。**所有接口均为 POST，参数放在 JSON body**。

### 双 token 登录机制

为适配公网部署，登录态采用 access JWT + refresh token 双 token 方案：

| token | 载体 | 有效期 | 状态 |
|-------|------|--------|------|
| access JWT | 响应体返回，前端存内存，请求经 `Authorization: Bearer` 头携带 | 15 分钟 | 无状态（HS256 自实现，不落库） |
| refresh token | HttpOnly cookie `refresh_token`（Path=`/manager`，SameSite=Lax，HTTPS 下带 Secure 标记） | 滑动 7 天 / 绝对 30 天 | 落库（仅存 SHA-256 哈希） |

要点：
- access JWT 过期（HTTP 401 + 业务码 1016）后前端自动调 `POST /manager/refresh` 续期并重试原请求，并发请求只触发一次刷新。
- refresh token 轮换：每次刷新删旧发新（同 family）。
- 重用检测：旧 refresh token 再次被使用时吊销整个 family，强制重登。
- 改密 / 禁用用户 / 重置密码后吊销该用户所有会话，即时失效。
- 账号不存在 / 禁用 / 密码错统一返回相同文案，防账号枚举。
- 签名密钥（JWT/票据）与加密密钥（落库敏感字段）分离，支持环境变量或 `<数据目录>/keys/` 密钥文件（0600）提供，未配置时自动生成。
- **HTTPS 反代部署**：refresh cookie 的 `Secure` 标记由后端根据请求 scheme 动态判定。若用 nginx 终止 TLS、HTTP 反代到后端，需转发 `X-Forwarded-Proto` 头（或 `X-Forwarded-Protocol` / `X-Forwarded-Ssl: on` / `X-Url-Scheme` 任一），后端据此识别为 HTTPS 并设置 `Secure`，否则会被误判为 HTTP：
  ```nginx
  location / {
      proxy_pass http://127.0.0.1:8887;
      proxy_set_header Host $host;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;   # 必须转发，否则 Secure cookie 识别失败
  }
  ```

### 权限模型

采用接口级权限 `(role_id, entity, action, value)`，存于 `role_permission` 表：
- `entity`：资源类型，如 `API`
- `action`：操作，如 `*`（通配，暂未启用细粒度，保留字段）
- `value`：资源标识，即接口路径，如 `/manager/self`；特殊值 `*` 为超管通配，放行所有接口
- 用户经 `user_roles` 关联多个角色，权限取并集
- 判定规则：当前请求路径 `c.Path()` 命中某条 `(entity=API, value=path)` 即通过；`value='*'` 直接放行所有接口；否则 403

即"某角色被授权某接口路径，就能访问该接口"。admin 角色配一条 `('API', '*', '*')` 即可访问全部超管接口；普通用户按接口路径精确授权（最小权限原则）。`/self` 后缀表示用户自助接口（只操作自己的数据），不加 `/self` 为超管接口。

### 菜单模型

侧栏菜单由后端驱动，存于 `menus` 表，通过 `role_menus` 与角色关联：
- `/manager/self` 接口返回当前用户的菜单树（按 `parent_id` 组装）
- 前端 `Home.vue` 动态渲染，不硬编码
- 菜单形态：有 `children` 为分组容器（点击展开）；无 `children` 且 `path` 非空为直接跳转；`path` 为空为纯标题
- 登录后默认跳转第一个可导航菜单

### 接口一览

> **分页接口**：标注「分页」的接口接受 `page`（页码，1-based，<1 当 1）和 `page_size`（每页条数，默认 20，上限 100）参数，返回 `{items, total, page, page_size}` 结构。

| 路径 | 说明 | 需登录 |
|------|------|--------|
| `POST /manager/login` | 登录，body `{account, password}`，返回 `{access_token, expires_in}` 并写 refresh cookie | 否 |
| `POST /manager/refresh` | 刷新 access JWT + 轮换 refresh token（靠 cookie，不走 access JWT） | 否 |
| `POST /manager/logout` | 登出，吊销当前登录链并清 cookie | 是 |
| `POST /manager/self` | 当前用户基本信息 | 是 |
| `POST /manager/recharge/self` | 给自己充值，body `{amount, remark}`（`userId` 由服务端注入当前用户，传入会被覆盖） | 是 |
| `POST /manager/recharge` | 管理员充值，body `{userId, amount, remark}` | 是 |
| `POST /manager/recharge/records` | 管理员查指定用户充值流水（分页），body `{userId, page, page_size}` | 是 |
| `POST /manager/recharge/records/self` | 查自己的充值流水（分页），body `{page, page_size}`（`userId` 由服务端注入当前用户） | 是 |
| `POST /manager/models` | 模型列表 | 是 |
| `POST /manager/apikeys/list/self` | 查自己的 API Key 列表（key 脱敏） | 是 |
| `POST /manager/apikeys/create/self` | 创建 API Key，body `{name, budget, unlimited}`，明文 key 仅本次返回 | 是 |
| `POST /manager/apikeys/toggle/self` | 启用/禁用 API Key，body `{id}` | 是 |
| `POST /manager/apikeys/delete/self` | 删除 API Key，body `{id}` | 是 |
| `POST /manager/apikeys/rename/self` | 重命名 API Key，body `{id, name}` | 是 |
| `POST /manager/apikeys/budget/self` | 修改 API Key 额度/限额模式，body `{id, budget, unlimited}`，有限额 key 总和不能超过账户余额 | 是 |
| `POST /manager/apikeys/models/get/self` | 查自己的 Key 的模型访问策略，body `{api_key_id}`，返回 `{model_policy, model_ids}` | 是 |
| `POST /manager/apikeys/models/set/self` | 设置自己的 Key 的模型访问策略，body `{api_key_id, model_policy(all\|whitelist), model_ids}` | 是 |
| `POST /manager/profile/update/self` | 修改个人资料，body `{name, email}` | 是 |
| `POST /manager/profile/password/self` | 修改密码，body `{old_password, new_password}`，需校验旧密码 | 是 |
| `POST /manager/usage/stats/self` | Token 用量统计，body `{mode, start_date, end_date, api_key_id(可选), model(可选), provider(可选), group_by(可选: model/provider/api_key)}`，返回 `{summary:{request_count, input_tokens, output_tokens, cached_tokens, cache_miss_tokens, reasoning_tokens, total_tokens, cache_hit_rate, total_cost, avg_cost}, rows:[{label, ...}]}` | 是 |
| `POST /manager/usage/filters/self` | 获取筛选选项（用过的 api_key / model / provider 列表）| 是 |

#### 超管接口（需 admin 角色）

| 路径 | 说明 |
|------|------|
| `POST /manager/users/list` | 用户列表（含角色），body `{keyword}` 按姓名/账号搜索 |
| `POST /manager/users/create` | 创建用户，body `{name, account, password, budget, unlimited}` |
| `POST /manager/users/update` | 编辑用户（账号不可改），body `{id, name, budget, unlimited}` |
| `POST /manager/users/toggle` | 启停用户，body `{id}` |
| `POST /manager/users/reset-password` | 重置密码，body `{id, password}` |
| `POST /manager/users/assign-roles` | 分配角色，body `{id, role_ids}` |
| `POST /manager/roles/list` | 角色列表 |
| `POST /manager/providers/list` | 提供商列表 |
| `POST /manager/providers/create` | 新增提供商，body `{type, domain, headers}` |
| `POST /manager/providers/update` | 编辑提供商（type 不可改），body `{type, domain, headers}` |
| `POST /manager/providers/toggle` | 启停提供商，body `{type}` |
| `POST /manager/models/list` | 模型列表，body `{provider, model}` 模糊搜索，返回项含 `supports_text/supports_image/supports_video` |
| `POST /manager/models/create` | 新增模型，body `{provider, model, input_cache_hit_price, input_cache_miss_price, output_price, max_context_tokens, max_completion_tokens, supports_text, supports_image, supports_video}` |
| `POST /manager/models/update` | 编辑模型（provider+model 不可改），字段同 create（无 provider/model，多 id）|
| `POST /manager/models/delete` | 删除模型 |
| `POST /manager/apikeys/list` | 查指定用户的 API Key，body `{user_id}` |
| `POST /manager/apikeys/toggle` | 启停指定用户的 Key，body `{id}` |
| `POST /manager/apikeys/delete` | 删除指定用户的 Key，body `{id}` |
| `POST /manager/apikeys/rename` | 重命名指定用户的 Key，body `{id, name}` |
| `POST /manager/apikeys/budget` | 修改指定用户 Key 额度，body `{id, budget, unlimited}` |
| `POST /manager/apikeys/models/get` | 查指定 Key 的模型访问策略，body `{api_key_id}`，返回 `{model_policy, model_ids}` |
| `POST /manager/apikeys/models/set` | 设置指定 Key 的模型访问策略，body `{api_key_id, model_policy(all\|whitelist), model_ids}` |
| `POST /manager/recharge/records/list` | 全平台充值流水（分页），body `{keyword, page, page_size}` 按用户名/账号/备注搜索 |
| `POST /manager/usage/stats` | 全局统计，body `{mode, start_date, end_date, user_id(可选), api_key_id(可选), model(可选), provider(可选), group_by(可选: user/model/provider/api_key)}` |
| `POST /manager/usage/filters` | 全局筛选选项（含用户列表） |
| `POST /manager/dashboard` | 仪表盘：汇总指标 + 近 7 天趋势 |

> admin 角色配一条 `role_permission('API', '*', '*')` 即可访问全部超管接口。

### 初始化管理员

执行 `sql/sqlite.sql` 建表后，再执行 `sql/init-data.sql` 初始化角色、权限、菜单等种子数据（用 `INSERT OR IGNORE` 可重复执行）：

```bash
sqlite3 ~/.aiapi/aiapi.db < sql/sqlite.sql
sqlite3 ~/.aiapi/aiapi.db < sql/init-data.sql
```

`init-data.sql` 包含：
- 2 个角色：`admin`（管理员）/ `user`（普通用户）
- admin 角色一条通配权限 `('API', '*', '*')`，放行所有超管接口
- user 角色按接口路径精确授权（自助接口）
- 10 条菜单及角色菜单关联

最后手动创建管理员账号（`password` 列写 bcrypt 哈希）：

```sql
INSERT INTO users (name, account, password, unlimited, enabled) VALUES ('管理员', 'admin', '<bcrypt-hash>', 1, 1);
INSERT INTO user_roles (user_id, role_id) VALUES ((SELECT id FROM users WHERE account='admin'), 1);
```

### 调用示例

```bash
# 登录（access token 返回体，refresh token 写 cookie.jar）
curl -c cookie.jar -X POST http://localhost:8888/manager/login \
  -H 'Content-Type: application/json' \
  -d '{"account":"admin","password":"你的密码"}'
# 返回示例：{"code":0,"data":{"access_token":"eyJ...","expires_in":900}}

# 刷新 access（带 refresh cookie，无需 access token）
curl -b cookie.jar -X POST http://localhost:8888/manager/refresh

# 后续业务请求需带 access token（cookie 仅用于 refresh）
curl -b cookie.jar -H "Authorization: Bearer <access_token>" \
  -X POST http://localhost:8888/manager/recharge \
  -H 'Content-Type: application/json' \
  -d '{"userId":2,"amount":10,"remark":"月度充值"}'
```

> 注意：当前项目仅代理 OpenAI 兼容格式的请求。若未来需要支持非 OpenAI 的客户端协议，需新增 `parser/` 实现并在 `framework/echo.go` 或 `proxy/direct.go` 中做必要适配。

详细的开发规范与约定请查看 [`AGENTS.md`](./AGENTS.md)。

## 贡献指南

1. Fork 本仓库
2. 在 `main` 之外创建功能分支
3. 保持 `AGENTS.md` 与代码同步
4. 提交 PR 并描述变更点

## 许可证

[MIT](LICENSE)（待补充）

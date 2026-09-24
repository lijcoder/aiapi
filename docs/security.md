# 安全与隐私

> 按需阅读：改动认证、鉴权、密钥、加密、日志脱敏前读本文件。
> 硬约束（红线）在 [`AGENTS.md`](../AGENTS.md)；**具体参数以代码为准**，本文件只说明机制与边界。

## 1. 敏感信息红线

- API Key 只在认证阶段使用，**禁止**出现在日志、错误信息或响应中（完整 Key 尤其禁止）。
- 保存请求头前必须对敏感头脱敏（`proxy/handler/log.go` 的关键词匹配：auth / cookie / key / token / secret / password）。
- 请求体目前原样入库，脱敏与截断开关尚未实现（`TODO.md`）；新增记录字段时按"默认不记敏感内容"设计。
- 密钥文件与数据库分离存放：备份 DB 时必须排除 `<DataDir>/keys/` 目录。

## 2. 后台接口的访问控制

- `manager/` 下所有需登录态的接口必须经过 `manager/middleware.Auth`：access JWT 校验 + 接口级权限判定（`role_permission(entity=API, value=path)`）+ 注入登录态；业务函数由 `base.Wrap` 做参数包装与响应输出。
- **不挂 Auth 的接口只有三个**：`/manager/login`（无需登录态）、`/manager/login/2fa`（凭 5 分钟 pending 票据）、`/manager/refresh`（靠 refresh cookie 续期）。`logout` 挂 Auth 但不挂 Require。
- 权限种子：超管角色 `role_permission` 中 `value='*'` 为通配权限，放行所有接口；普通用户按接口路径精确授权（最小权限原则）。**新增非超管接口必须同步 `sql/init-data.sql` 的 `role_permission`**（路径须与路由完整路径一致，含 `/manager` 前缀），否则用户调用直接 403。
- 该义务有门禁兜底：`gate_arch_test.go` 校验所有 `/self` 路由都已授权、且种子里没有野路径。**非 `/self` 但面向普通用户的路由推导不出来**（如 `/manager/models`），需人工判断——历史事故见 [`postmortem/2026-09-24-user-role-permission-seed-gap.md`](postmortem/2026-09-24-user-role-permission-seed-gap.md)。
- 菜单可见性由 `menus` + `role_menus` 决定，与接口权限是两套：新增页面需同时补 `frontend/src/router/index.js` children、`menus` 数据与 `role_menus` 授权，否则页面不可达。

## 3. 登录态：双 token 机制

参数（TTL、cookie 名、token 字节数）定义在 `manager/base/constant.go`，会话业务封装在 `service` 的 `SessionService`：

- **access JWT**：HS256 自实现，无状态短时效，经 `Authorization: Bearer` 头传递，前端存内存（不落 localStorage）。
- **refresh token**：随机串哈希存 `user_sessions` 表，经 HttpOnly + Secure cookie（`refresh_token`）传递；带滑动窗口 + 绝对上限、轮换与重用检测。
- 登录态**不复用** proxy 的 header 鉴权字段。
- 改密 / 禁用用户 / 重置密码后必须吊销该用户的所有会话（走 `SessionService`，不要在 handler 里直接删表）。
- 账号不存在 / 被禁用 / 密码错误统一返回相同文案，防账号枚举。

## 4. 两步验证（2FA / TOTP）

- 可选开启：`users.totp_secret` 存 AES-256-GCM 加密的 TOTP 密钥，空串=未开启。
- 登录分两步：密码通过 → 签发 pending 票据（HS256 JWT，短时效，`purpose=2fa_pending`）→ `/manager/login/2fa` 验票 + 验证码后才发 token 对；同一票据验证码连续错误达上限即作废（内存计数）。
- 绑定用 setup 票据（`purpose=2fa_setup`，内含密钥），确认首个验证码后才落库。
- 业务封装在 `service/totp.go`（`TOTPService`），handler 不自行实现。

## 5. 密钥管理

`manager/base/secret.go` 把两类密钥分离：

| 密钥 | 用途 | 环境变量 |
|------|------|----------|
| `JWTSecret` | JWT / 2FA 票据签名 | `AIAPI_JWT_SECRET` |
| `CryptoSecret` | 派生 AES 密钥（`service/crypto.go`），加密 TOTP 密钥、Provider 配置等落库敏感字段 | `AIAPI_CRYPTO_SECRET` |

加载优先级：环境变量 > 密钥文件（`<DataDir>/keys/*.key`，0600）> 自动生成并写文件。`crypto.key` 缺失且签名密钥来自环境变量时从其播种（旧部署兼容）。业务层只做编排，不各自实现加载逻辑。

## 6. 其他

- Provider 配置的敏感字段落库前加密（兼容历史明文），读取统一走 `service.GetProviderConfig` / `ParseProviderConfig`。
- 管理接口（Provider、模型、用户、充值）必须受权限体系约束，避免任意用户修改配置或额度。
- 唯一索引冲突等可预期错误要转成明确的业务错误（中文），不要把数据库错误直接暴露给前端。

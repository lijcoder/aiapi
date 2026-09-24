# 部署与运维

> 首次安装与调用示例见 [`../README.md`](../README.md)；数据库结构与迁移见 [`../sql/AGENTS.md`](../sql/AGENTS.md) 与 [`../sql/migrations/README.md`](../sql/migrations/README.md)。

## 数据目录

```
<数据目录>/                     # 默认 ~/.aiapi
├── db/aiapi.db                # SQLite（WAL，busy_timeout=5000）；应用不自动创建，须手工初始化
├── logs/app.log               # 应用日志（同时输出 stdout）
└── keys/
    ├── jwt.key                # 签名密钥（0600）
    └── crypto.key             # 加密密钥（0600）
```

密钥加载优先级：**环境变量 > 密钥文件 > 自动生成**。目录权限 0700、密钥文件 0600，由应用创建。

> ⚠️ **轮换密钥的后果**：签名密钥轮换 → 所有登录会话失效（用户重登）；加密密钥轮换 → 历史密文（2FA 密钥、Provider 配置、API Key 原文）无法解密，需重新绑定/保存。务必把 `keys/` 与数据库分开存放，**备份数据库时排除 `keys/`**。

## HTTPS 反向代理

服务本身是明文 HTTP，公网部署必须在前面加 TLS 终结（nginx / Caddy）。**必须转发 `X-Forwarded-Proto`**，否则后端会把 HTTPS 请求当作 HTTP，refresh cookie 不会带 `Secure` 标记：

```nginx
server {
    listen 443 ssl;
    server_name aiapi.example.com;

    location / {
        proxy_pass http://127.0.0.1:8888;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;   # 必须，Secure cookie 依赖它
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;                          # SSE 流式必须关闭缓冲
        proxy_read_timeout 3600s;                     # 长流响应
    }
}
```

## 生产检查清单

- [ ] 数据库已初始化并备份，`keys/` 目录已排除在备份之外
- [ ] 通过环境变量提供 `AIAPI_JWT_SECRET` / `AIAPI_CRYPTO_SECRET`（≥32 字节、每环境独立）
- [ ] TLS 终结已配置，且转发 `X-Forwarded-Proto`
- [ ] 反代已关闭缓冲（SSE）并放宽读超时
- [ ] 修改了初始管理员密码，按需开启 2FA
- [ ] 按需在反代层做登录接口限流（见 [`../README.md`](../README.md#已知限制)）
- [ ] 服务进程由 systemd / supervisor 守护，数据目录权限收紧

## 日志

同时输出到 stdout 与 `<数据目录>/logs/app.log`，按大小轮转：单文件 100MB、保留 10 个备份、最长 30 天。级别固定为 Info（不可配置）。管理台「日志」页可查看最近 100 行。

## 备份

```bash
sqlite3 ~/.aiapi/db/aiapi.db ".backup '/backup/aiapi-$(date +%F).db'"
```

备份后建议验证一次可读（`PRAGMA integrity_check;`）。**不要把 `keys/` 一起备份**：密钥与数据分离存放，丢失密钥会导致密文字段永久无法解密。

## 升级

应用不会自动迁移。升级前：① 备份数据库；② 阅读 [`../CHANGELOG.md`](../CHANGELOG.md) 的破坏性变更；③ 按 [`../sql/migrations/README.md`](../sql/migrations/README.md) 执行缺失的迁移。

**schema 版本校验**：数据库结构版本记录在 `schema_meta` 表（单行，跨数据库通用），启动时与二进制内置的 `constant.SchemaVersion` 比对：

| 库里的版本 | 行为 |
|------------|------|
| 与二进制一致 | 正常启动 |
| 比二进制旧 | **拒绝启动**，提示要执行哪些迁移 |
| 比二进制新 | **拒绝启动**（二进制过旧，降级会丢列丢数据） |
| 没有版本标记（老库缺表） | 告警但允许启动 |

```bash
# 查看当前版本（报 "no such table" 表示还是引入版本标记之前的老库）
sqlite3 ~/.aiapi/db/aiapi.db "SELECT version FROM schema_meta WHERE id = 1;"

# 补标记为当前版本
sqlite3 ~/.aiapi/db/aiapi.db "DELETE FROM schema_meta; INSERT INTO schema_meta (id, version, applied_at) VALUES (1, 1, datetime('now','localtime'));"
```

## 数据表

| 表 | 用途 |
|----|------|
| `providers` | 上游 Provider 配置（`config` 加密存储） |
| `api_keys` / `apikey_model_access` | 调用方 Key（哈希 + 密文 + 展示串）与模型白名单 |
| `models` | 模型、上游模型名、计费配置 |
| `usage_records` / `request_logs` | 计费用量（仅成功请求） / 请求日志（成败都记） |
| `recharge_records` | 充值流水（财务凭证，不建议清理） |
| `users` / `roles` / `user_roles` / `role_permission` | 用户、角色、接口级权限 |
| `menus` / `role_menus` | 菜单与角色菜单 |
| `user_sessions` | refresh token 会话（仅存哈希） |
| `schema_meta` | 数据库结构版本（单行） |

完整 DDL 见 [`../sql/sqlite.sql`](../sql/sqlite.sql)，种子数据见 [`../sql/init-data.sql`](../sql/init-data.sql)。

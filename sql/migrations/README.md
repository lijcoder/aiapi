# 数据库迁移台账

> 应用**不会**自动建库或迁移。新建库直接用 [`sql/sqlite.sql`](../sqlite.sql)（当前完整 schema）；
> 存量库按本文件执行对应迁移，然后把数据库标记到当前版本。
>
> 版本权威只有一个：`constant.SchemaVersion`（Go 常量）。应用启动时读取库里 `schema_meta` 表记录的版本
> 与之比对：一致则正常启动；库比二进制旧会拒绝启动并提示要跑哪些迁移；库比二进制新同样拒绝启动；
> 还没有版本标记（老库没有 `schema_meta` 表）会告警但允许启动。
> `sql/sqlite.sql` 写入的版本必须与常量一致，由 `store` 包的测试强制。

## 版本表

| 版本 | 对应变更 | 迁移方式 |
|------|----------|----------|
| **v1** | 当前基线：`sql/sqlite.sql` 的完整 schema（含 API Key 哈希化、模型分段计费、双 token 会话、2FA、`provider_model` 等全部现状） | 新建库直接执行 `sql/sqlite.sql`；未标记的历史库按下面「v0 → v1」逐项补齐后标记 v1 |

## 升级流程

```bash
# 1. 备份（务必排除 keys/ 目录，密钥与数据分离存放）
sqlite3 <DataDir>/db/aiapi.db ".backup '/backup/aiapi-$(date +%F).db'"

# 2. 查看当前版本（报 "no such table" 表示还是引入版本标记之前的老库）
sqlite3 <DataDir>/db/aiapi.db "SELECT version FROM schema_meta WHERE id = 1;"

# 3. 按下面「v0 → v1」执行缺失的迁移（每项都注明适用条件，已执行过的跳过）

# 4. 建元数据表并标记版本（幂等：先清后写，单行表）
sqlite3 <DataDir>/db/aiapi.db "
CREATE TABLE IF NOT EXISTS schema_meta (
  id INTEGER PRIMARY KEY, version INTEGER NOT NULL, applied_at DATETIME NOT NULL);
DELETE FROM schema_meta;
INSERT INTO schema_meta (id, version, applied_at) VALUES (1, 1, datetime('now','localtime'));"
```

### 多数据库

版本记录用的是普通元数据表，**不是 SQLite 专有的 `PRAGMA user_version`**，因为本项目要兼容
MySQL / PostgreSQL（`user_version` 只有 SQLite 有，MySQL 没有等价物）。Go 侧读取统一是：

```sql
SELECT version FROM schema_meta WHERE id = 1
```

各数据库只有建表/写入语句不同，迁移到新库时按下面的写法落到该库的 DDL 文件里：

| 数据库 | 建表与写入版本行 |
|--------|------------------|
| SQLite / PostgreSQL | `INSERT INTO schema_meta (id, version, applied_at) SELECT 1, 1, <now()> WHERE NOT EXISTS (SELECT 1 FROM schema_meta WHERE id = 1);`（SQLite 用 `datetime('now','localtime')`，PG 用 `NOW()`） |
| MySQL | `INSERT INTO schema_meta (id, version, applied_at) SELECT 1, 1, NOW() FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM schema_meta WHERE id = 1);` |

判断某项是否已执行：用 `sqlite3 <DataDir>/db/aiapi.db ".schema <表名>"` 看列是否存在。
**迁移语句不可重复执行**（`ADD COLUMN` 重复会报 `duplicate column name`），执行前先确认。

## v0 → v1（历史迁移）

这些是按时间累积的变更，未标记版本的旧库按需执行。全部完成后即等价于当前 `sql/sqlite.sql`。

### 1. `users` 增列

```sql
ALTER TABLE users ADD COLUMN password TEXT NOT NULL DEFAULT '';
-- 然后逐个设置密码哈希（bcrypt）：
-- UPDATE users SET password = '<bcrypt-hash>' WHERE account = 'admin';

ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN totp_secret TEXT NOT NULL DEFAULT '';  -- 2FA，AES-GCM 密文，空=未开启
```

执行完 2FA 迁移后需重跑 `sql/init-data.sql`，为普通用户角色补 2FA 接口权限。

### 2. `api_keys` 明文 key → `key_hash` + `key_show`

SQLite 无 SHA-256 函数，哈希必须在外部计算（Python `hashlib` / `shasum -a 256`）。

```sql
ALTER TABLE api_keys RENAME COLUMN key TO key_hash;
ALTER TABLE api_keys ADD COLUMN key_show TEXT NOT NULL DEFAULT '';
-- 3. 逐行回填：明文 key 的 sha256 hex 与展示串（sk- + 前 3 位 hex + '****' + 后 3 位 hex）
--    UPDATE api_keys SET key_hash = '<sha256hex>', key_show = 'sk-abc****xyz' WHERE id = <id>;
DROP INDEX IF EXISTS uq_api_keys_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_api_keys_key_hash ON api_keys(key_hash);
```

迁移后鉴权按 `key_hash` 比对，重启应用生效。

### 3. `api_keys` 增列：模型策略与可还原密文

```sql
ALTER TABLE api_keys ADD COLUMN model_policy TEXT NOT NULL DEFAULT 'all';
ALTER TABLE api_keys ADD COLUMN key_enc TEXT NOT NULL DEFAULT '';
```

存量 key 的明文已不可还原（当时只存 SHA-256），`key_enc` 留空；查看接口对 `key_enc=''` 的记录会提示无法还原。

### 4. `usage_records` / `request_logs`：明文 key → `api_key_id`

需要 SQLite ≥ 3.35（`DROP COLUMN` 支持）。

```sql
ALTER TABLE usage_records ADD COLUMN api_key_id INTEGER NOT NULL DEFAULT 0;
UPDATE usage_records SET api_key_id = COALESCE((SELECT id FROM api_keys WHERE api_keys.key = usage_records.api_key), 0);
ALTER TABLE usage_records DROP COLUMN api_key;
CREATE INDEX IF NOT EXISTS idx_usage_records_api_key_id ON usage_records(api_key_id);

ALTER TABLE request_logs ADD COLUMN api_key_id INTEGER NOT NULL DEFAULT 0;
UPDATE request_logs SET api_key_id = COALESCE((SELECT id FROM api_keys WHERE api_keys.key = request_logs.api_key), 0);
ALTER TABLE request_logs DROP COLUMN api_key;
```

> 第 2 步依赖 `api_keys.key` 列仍然存在——**先做第 2 节（哈希化）之前的库才能这样回填**。
> 若 `api_keys.key` 已被重命名，改为从 `api_keys.key_hash` 反查，或接受 `api_key_id = 0`。

### 5. 耗时字段

```sql
ALTER TABLE usage_records ADD COLUMN first_token_ms INTEGER DEFAULT 0;
ALTER TABLE usage_records ADD COLUMN latency_ms   INTEGER DEFAULT 0;
ALTER TABLE request_logs  ADD COLUMN first_token_ms INTEGER DEFAULT 0;
ALTER TABLE request_logs  ADD COLUMN latency_ms   INTEGER DEFAULT 0;
```

### 6. 模型分段计费（v2 `pricing_config`）

原三项固定单价不迁移，**升级后需在管理台重新配置每个模型的计费**；未配置或配置非法的模型不会被转发。

```sql
ALTER TABLE models ADD COLUMN pricing_config TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_records ADD COLUMN pricing_snapshot TEXT NOT NULL DEFAULT '';
ALTER TABLE models DROP COLUMN input_cache_hit_price;
ALTER TABLE models DROP COLUMN input_cache_miss_price;
ALTER TABLE models DROP COLUMN output_price;
```

### 7. `models` 增列：多模态能力与上游模型名

```sql
ALTER TABLE models ADD COLUMN supports_text  INTEGER NOT NULL DEFAULT 1;
ALTER TABLE models ADD COLUMN supports_image INTEGER NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN supports_video INTEGER NOT NULL DEFAULT 0;

ALTER TABLE models ADD COLUMN provider_model TEXT NOT NULL DEFAULT '';
-- 回填，使旧数据行为与迁移前一致（转发时按用户可见模型名发上游）：
UPDATE models SET provider_model = model WHERE provider_model = '';
```

### 8. `role_permission` 重建

权限是种子数据，可丢弃重建：

```sql
DROP TABLE IF EXISTS role_permission;
```

然后重跑 `sql/init-data.sql` 重建角色权限与菜单种子。

### 9. `user_sessions` 双 token 会话

老 token 格式不兼容，需清空让用户重登一次：

```sql
ALTER TABLE user_sessions ADD COLUMN family_id TEXT NOT NULL DEFAULT '';
ALTER TABLE user_sessions ADD COLUMN absolute_expires_at DATETIME;
ALTER TABLE user_sessions ADD COLUMN ua TEXT NOT NULL DEFAULT '';
ALTER TABLE user_sessions ADD COLUMN ip TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_user_sessions_family ON user_sessions(family_id);
DELETE FROM user_sessions;
```

### 10. 时区修复（可选）

早期数据若按 UTC 写入而库改用本地时间，按需整体平移：

```sql
UPDATE <表名> SET created_at = datetime(created_at, '+8 hours');
```

## 新增一个版本时

1. 改 `sql/sqlite.sql`：把新列/新表直接写进当前 schema（新库一步到位）。
2. 在 `sql/migrations/` 下新建 `v<N>-<主题>.sql`，只写**从上一版到本版**的增量语句。
3. 在 `constant.SchemaVersion` 加一，并同步 `sql/sqlite.sql` 里 `schema_meta` 的版本行（测试会校验两者一致）。
4. 在本文件「版本表」加一行，写清适用条件、是否破坏性、以及需要人工介入的部分（如外部计算哈希、重新配置计费）。
5. `CHANGELOG.md` 记录影响面。

-- SQLite DDL — 手动执行: sqlite3 ~/.aiapi/aiapi.db < sql/sqlite.sql
-- 如果迁移到 MySQL, 参考 sql/mysql.sql
--
-- 规范:
--   PRIMARY KEY         → 放在 CREATE TABLE 内
--   UNIQUE / INDEX      → 统一用 CREATE INDEX / CREATE UNIQUE INDEX
--   FOREIGN KEY         → 不使用（业务层保证引用完整性）
--
-- 时区说明:
--   datetime('now', 'localtime') 使用系统本地时间（非 UTC）
--   存量数据修复: UPDATE 表名 SET created_at = datetime(created_at, '+8 hours');

CREATE TABLE IF NOT EXISTS providers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    type       TEXT NOT NULL,
    config     TEXT NOT NULL DEFAULT '{}',
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now', 'localtime')),
    updated_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_providers_type ON providers(type);

-- users 扩展：增加 password 列
-- 新建库直接用上方定义；存量库需手动迁移：
--   ALTER TABLE users ADD COLUMN password TEXT NOT NULL DEFAULT '';
--   -- 然后逐个设置密码哈希：
--   -- UPDATE users SET password = '<bcrypt-hash>' WHERE account = 'admin';
CREATE TABLE IF NOT EXISTS users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    account    TEXT NOT NULL,
    password   TEXT NOT NULL,           -- 存 bcrypt 哈希
    email      TEXT NOT NULL DEFAULT '', -- 邮箱（可空，不唯一）
    budget     REAL NOT NULL DEFAULT 0,
    unlimited  INTEGER NOT NULL DEFAULT 0,
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_account ON users(account);

-- users 扩展：增加 email 列
-- 新建库直接用上方定义；存量库需手动迁移：
--   ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS api_keys (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    key        TEXT NOT NULL,
    name       TEXT DEFAULT '',
    budget     REAL NOT NULL DEFAULT 0,
    unlimited  INTEGER NOT NULL DEFAULT 0,
    enabled    INTEGER DEFAULT 1,
    model_policy TEXT NOT NULL DEFAULT 'all',  -- 'all'=全量放行 | 'whitelist'=按 apikey_model_access 白名单
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_api_keys_key ON api_keys(key);

-- api_keys 扩展：增加 model_policy 列（模型访问策略）
-- 新建库直接用上方定义；存量库需手动迁移：
--   ALTER TABLE api_keys ADD COLUMN model_policy TEXT NOT NULL DEFAULT 'all';

-- API Key 模型白名单明细（model_policy='whitelist' 时生效）
CREATE TABLE IF NOT EXISTS apikey_model_access (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key_id  INTEGER NOT NULL,
    model_id    INTEGER NOT NULL,
    created_at  DATETIME DEFAULT (datetime('now', 'localtime')),
    UNIQUE(api_key_id, model_id)
);

CREATE TABLE IF NOT EXISTS usage_records (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL,
    api_key        TEXT DEFAULT '',
    provider       TEXT NOT NULL,
    model          TEXT NOT NULL,
    input_tokens   INTEGER NOT NULL DEFAULT 0,
    output_tokens  INTEGER NOT NULL DEFAULT 0,
    total_tokens   INTEGER NOT NULL DEFAULT 0,
    request_id     TEXT DEFAULT '',
    stream         INTEGER DEFAULT 0,
    cached_tokens  INTEGER DEFAULT 0,
    reasoning_tokens INTEGER DEFAULT 0,
    cost             REAL DEFAULT 0,
    unlimited        INTEGER DEFAULT 0,
    created_at     DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE INDEX IF NOT EXISTS idx_usage_records_user  ON usage_records(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_date  ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_model ON usage_records(model);

CREATE TABLE IF NOT EXISTS request_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key         TEXT DEFAULT '',
    format          TEXT NOT NULL,
    provider        TEXT NOT NULL DEFAULT '',
    path            TEXT NOT NULL,
    status_code     INTEGER DEFAULT 0,
    request_headers TEXT DEFAULT '',
    request_body    TEXT DEFAULT '',
    response_body   TEXT DEFAULT '',
    model           TEXT DEFAULT '',
    input_tokens    INTEGER DEFAULT 0,
    output_tokens   INTEGER DEFAULT 0,
    total_tokens    INTEGER DEFAULT 0,
    error           TEXT DEFAULT '',
    latency_ms      INTEGER DEFAULT 0,
    created_at      DATETIME DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE IF NOT EXISTS models (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    provider              TEXT NOT NULL,
    model                 TEXT NOT NULL,
    input_cache_hit_price  REAL NOT NULL DEFAULT 0,
    input_cache_miss_price REAL NOT NULL DEFAULT 0,
    output_price           REAL NOT NULL DEFAULT 0,
    max_context_tokens    INTEGER DEFAULT 0,
    max_completion_tokens INTEGER DEFAULT 0,
    supports_text         INTEGER NOT NULL DEFAULT 1,
    supports_image        INTEGER NOT NULL DEFAULT 0,
    supports_video        INTEGER NOT NULL DEFAULT 0,
    created_at            DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_models_provider_model ON models(provider, model);

-- models 扩展：增加多模态能力列（文本/图像/视频）
-- 新建库直接用上方定义；存量库需手动迁移：
--   ALTER TABLE models ADD COLUMN supports_text  INTEGER NOT NULL DEFAULT 1;
--   ALTER TABLE models ADD COLUMN supports_image INTEGER NOT NULL DEFAULT 0;
--   ALTER TABLE models ADD COLUMN supports_video INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS recharge_records (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL,
    amount         REAL NOT NULL,
    balance_before REAL NOT NULL,
    balance_after  REAL NOT NULL,
    operator       TEXT DEFAULT '',
    remark         TEXT DEFAULT '',
    created_at     DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE INDEX IF NOT EXISTS idx_recharge_records_user ON recharge_records(user_id);
CREATE INDEX IF NOT EXISTS idx_recharge_records_date ON recharge_records(created_at);

-- ===== 后台管理：登录 / 角色 / 权限 / 会话 =====

CREATE TABLE IF NOT EXISTS roles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    code        TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL DEFAULT '',
    created_at  DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_name ON roles(name);

CREATE TABLE IF NOT EXISTS user_roles (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    role_id    INTEGER NOT NULL,
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_pair ON user_roles(user_id, role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);

-- 接口级权限 (subject=role_id, entity, action, value)
--   entity: 资源类型，如 'API'
--   action: 操作，如 '*'（通配，暂未启用细粒度，保留字段）
--   value : 资源标识，如 '/manager/self'
-- 存量库迁移（role_permission 为种子数据，可丢弃重建）：
--   DROP TABLE IF EXISTS role_permission;
CREATE TABLE IF NOT EXISTS role_permission (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    role_id    INTEGER NOT NULL,
    entity     TEXT NOT NULL,
    action     TEXT NOT NULL,
    value      TEXT NOT NULL,
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_role_permission ON role_permission(role_id, entity, action, value);

-- user_sessions 存储登录会话（refresh token），access JWT 不落库。
-- 字段说明：
--   token               refresh token 的 SHA-256 哈希（hex），明文不落库
--   family_id           登录链标识，重用检测用（同一次登录的多次刷新共享同一 family）
--   expires_at          滑动过期时间，每次刷新顺延
--   absolute_expires_at 绝对过期上限，登录时设定不随刷新顺延
--   ua/ip               User-Agent 摘要与登录 IP，仅展示用
--
-- 存量库迁移（双 token 机制上线时执行）：
--   ALTER TABLE user_sessions ADD COLUMN family_id TEXT NOT NULL DEFAULT '';
--   ALTER TABLE user_sessions ADD COLUMN absolute_expires_at DATETIME;
--   ALTER TABLE user_sessions ADD COLUMN ua TEXT NOT NULL DEFAULT '';
--   ALTER TABLE user_sessions ADD COLUMN ip TEXT NOT NULL DEFAULT '';
--   CREATE INDEX IF NOT EXISTS idx_user_sessions_family ON user_sessions(family_id);
--   DELETE FROM user_sessions;  -- 清空旧 session，老 token 格式不兼容，用户需重登一次
CREATE TABLE IF NOT EXISTS user_sessions (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    token               TEXT NOT NULL,
    family_id           TEXT NOT NULL DEFAULT '',
    user_id             INTEGER NOT NULL,
    expires_at          DATETIME NOT NULL,
    absolute_expires_at DATETIME NOT NULL,
    ua                  TEXT NOT NULL DEFAULT '',
    ip                  TEXT NOT NULL DEFAULT '',
    created_at          DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_sessions_token ON user_sessions(token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_family ON user_sessions(family_id);

CREATE TABLE IF NOT EXISTS menus (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id  INTEGER DEFAULT 0,
    name       TEXT NOT NULL,
    path       TEXT NOT NULL,
    icon       TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE IF NOT EXISTS role_menus (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    role_id    INTEGER NOT NULL,
    menu_id    INTEGER NOT NULL,
    created_at DATETIME DEFAULT (datetime('now', 'localtime')),
    UNIQUE(role_id, menu_id)
);

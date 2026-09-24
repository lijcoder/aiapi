-- SQLite 当前 schema DDL
--
-- 新建库（应用不会自动建库，DB 文件必须预先存在）：
--   mkdir -p ~/.aiapi/db
--   sqlite3 ~/.aiapi/db/aiapi.db < sql/sqlite.sql
--   sqlite3 ~/.aiapi/db/aiapi.db < sql/init-data.sql
--
-- 本文件是**当前版本的完整 schema**（等价于所有历史迁移执行后的结果），不承载迁移步骤。
-- 存量库升级步骤见 sql/migrations/README.md；schema 版本记录在 schema_meta 表（单行），
-- 应用启动时会与 constant.SchemaVersion 比对，不一致会拒绝启动。
--
-- 规范:
--   PRIMARY KEY         → 放在 CREATE TABLE 内
--   UNIQUE / INDEX      → 统一用 CREATE INDEX / CREATE UNIQUE INDEX
--   FOREIGN KEY         → 不使用（业务层保证引用完整性）
--
-- 时区说明:
--   datetime('now', 'localtime') 使用系统本地时间（非 UTC）

-- schema_meta：数据库结构版本（单行表，id 固定为 1）。
-- 刻意不用 SQLite 专有的 PRAGMA user_version：本项目要兼容 MySQL / PostgreSQL，
-- 元数据表是三者的公共做法，Go 侧用同一条 SELECT 读取。
CREATE TABLE IF NOT EXISTS schema_meta (
    id         INTEGER PRIMARY KEY,   -- 固定 1，保证单行
    version    INTEGER NOT NULL,      -- schema 版本号，与 constant.SchemaVersion 对应
    applied_at DATETIME NOT NULL
);
-- 幂等写入版本行（重复执行整个 DDL 不会报错）
INSERT INTO schema_meta (id, version, applied_at)
SELECT 1, 1, datetime('now', 'localtime')
WHERE NOT EXISTS (SELECT 1 FROM schema_meta WHERE id = 1);

CREATE TABLE IF NOT EXISTS providers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    type       TEXT NOT NULL,
    config     TEXT NOT NULL DEFAULT '{}',
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now', 'localtime')),
    updated_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_providers_type ON providers(type);

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    account    TEXT NOT NULL,
    password   TEXT NOT NULL,           -- 存 bcrypt 哈希
    totp_secret TEXT NOT NULL DEFAULT '', -- TOTP 密钥（AES-GCM 加密存储，空=未开启 2FA）
    email      TEXT NOT NULL DEFAULT '', -- 邮箱（可空，不唯一）
    budget     REAL NOT NULL DEFAULT 0,
    unlimited  INTEGER NOT NULL DEFAULT 0,
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_account ON users(account);

CREATE TABLE IF NOT EXISTS api_keys (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    key_hash   TEXT NOT NULL,              -- key 原文不落库：SHA-256(hex)，鉴权比对用
    key_enc    TEXT NOT NULL DEFAULT '',   -- key 原文 AES-256-GCM 密文（base64），空串=旧版本不可还原
    key_show   TEXT NOT NULL DEFAULT '',   -- 展示串（sk-abc****xyz），创建后仅存此片段
    name       TEXT DEFAULT '',
    budget     REAL NOT NULL DEFAULT 0,
    unlimited  INTEGER NOT NULL DEFAULT 0,
    enabled    INTEGER DEFAULT 1,
    model_policy TEXT NOT NULL DEFAULT 'all',  -- 'all'=全量放行 | 'whitelist'=按 apikey_model_access 白名单
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_api_keys_key_hash ON api_keys(key_hash);

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
    api_key_id     INTEGER NOT NULL DEFAULT 0,  -- 关联 api_keys.id，不存 key 原文（防 DB 泄露丢 key）
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
    pricing_snapshot TEXT NOT NULL DEFAULT '',
    first_token_ms   INTEGER DEFAULT 0,
    latency_ms       INTEGER DEFAULT 0,
    created_at     DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE INDEX IF NOT EXISTS idx_usage_records_user  ON usage_records(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_date  ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_model ON usage_records(model);
CREATE INDEX IF NOT EXISTS idx_usage_records_api_key_id ON usage_records(api_key_id);

CREATE TABLE IF NOT EXISTS request_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key_id      INTEGER NOT NULL DEFAULT 0,  -- 关联 api_keys.id，不存 key 原文
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
    first_token_ms  INTEGER DEFAULT 0,
    latency_ms      INTEGER DEFAULT 0,
    created_at      DATETIME DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE IF NOT EXISTS models (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    provider              TEXT NOT NULL,
    model                 TEXT NOT NULL,              -- 对用户可见的模型名，鉴权/计价/白名单按它匹配
    provider_model        TEXT NOT NULL DEFAULT '',   -- 转发时替换请求体 model 的上游模型名，空串=跟随 model
    pricing_config        TEXT NOT NULL DEFAULT '',
    max_context_tokens    INTEGER DEFAULT 0,
    max_completion_tokens INTEGER DEFAULT 0,
    supports_text         INTEGER NOT NULL DEFAULT 1,
    supports_image        INTEGER NOT NULL DEFAULT 0,
    supports_video        INTEGER NOT NULL DEFAULT 0,
    created_at            DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_models_provider_model ON models(provider, model);

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
--   value : 资源标识，如 '/manager/self'；'*' 为超管通配
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

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

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    account    TEXT NOT NULL,
    budget     REAL NOT NULL DEFAULT 0,
    unlimited  INTEGER NOT NULL DEFAULT 0,
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_account ON users(account);

CREATE TABLE IF NOT EXISTS api_keys (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    key        TEXT NOT NULL,
    name       TEXT DEFAULT '',
    budget     REAL NOT NULL DEFAULT 0,
    unlimited  INTEGER NOT NULL DEFAULT 0,
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_api_keys_key ON api_keys(key);

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
    created_at            DATETIME DEFAULT (datetime('now', 'localtime'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_models_provider_model ON models(provider, model);

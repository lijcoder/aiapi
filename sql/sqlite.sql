-- SQLite DDL — 手动执行: sqlite3 ~/.aiapi/aiapi.db < sql/sqlite.sql
-- 如果迁移到 MySQL, 参考 sql/mysql.sql
--
-- 规范:
--   PRIMARY KEY         → 放在 CREATE TABLE 内
--   UNIQUE / INDEX      → 统一用 CREATE INDEX / CREATE UNIQUE INDEX
--   FOREIGN KEY         → 不使用（业务层保证引用完整性）

CREATE TABLE IF NOT EXISTS providers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    type       TEXT NOT NULL,
    config     TEXT NOT NULL DEFAULT '{}',
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_providers_type ON providers(type);

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    budget     REAL DEFAULT 0,
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS api_keys (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    key        TEXT NOT NULL,
    name       TEXT DEFAULT '',
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now'))
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
    created_at     DATETIME DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_usage_records_user  ON usage_records(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_date  ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_model ON usage_records(model);

CREATE TABLE IF NOT EXISTS request_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key         TEXT DEFAULT '',
    format          TEXT NOT NULL,
    provider        TEXT NOT NULL DEFAULT '',
    method          TEXT NOT NULL,
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
    created_at      DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS model_pricing (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    provider              TEXT NOT NULL,
    model                 TEXT NOT NULL,
    input_cache_hit_price  REAL NOT NULL DEFAULT 0,
    input_cache_miss_price REAL NOT NULL DEFAULT 0,
    output_price           REAL NOT NULL DEFAULT 0,
    created_at            DATETIME DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_model_pricing ON model_pricing(provider, model);

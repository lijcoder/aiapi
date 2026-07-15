package db

const schema = `
CREATE TABLE IF NOT EXISTS providers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    type       TEXT NOT NULL,
    config     TEXT NOT NULL DEFAULT '{}',
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now')),
    UNIQUE(type)
);

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
    key        TEXT NOT NULL UNIQUE,
    name       TEXT DEFAULT '',
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id)
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
    created_at     DATETIME DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_usage_records_user   ON usage_records(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_date   ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_model  ON usage_records(model);
CREATE INDEX IF NOT EXISTS idx_api_keys_key         ON api_keys(key);

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
`

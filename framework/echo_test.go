package framework

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/store"
)

// setupEchoTest 初始化内存 SQLite（含用户/Key/模型种子数据）与 Echo 实例。
// Key 明文为 sk-test，鉴权按其 SHA-256 哈希比对。
func setupEchoTest(t *testing.T) *echo.Echo {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.MustExec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, account TEXT NOT NULL,
		password TEXT NOT NULL, totp_secret TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '',
		budget REAL NOT NULL DEFAULT 0, unlimited INTEGER NOT NULL DEFAULT 0,
		enabled INTEGER DEFAULT 1, created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	db.MustExec(`CREATE TABLE api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
		key_hash TEXT NOT NULL, key_show TEXT NOT NULL DEFAULT '', name TEXT DEFAULT '',
		budget REAL NOT NULL DEFAULT 0, unlimited INTEGER NOT NULL DEFAULT 0,
		enabled INTEGER DEFAULT 1, model_policy TEXT NOT NULL DEFAULT 'all',
		created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	db.MustExec(`CREATE TABLE apikey_model_access (
		id INTEGER PRIMARY KEY AUTOINCREMENT, api_key_id INTEGER NOT NULL, model_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT (datetime('now', 'localtime')), UNIQUE(api_key_id, model_id)
	)`)
	db.MustExec(`CREATE TABLE models (
		id INTEGER PRIMARY KEY AUTOINCREMENT, provider TEXT NOT NULL, model TEXT NOT NULL,
		input_cache_hit_price REAL NOT NULL DEFAULT 0, input_cache_miss_price REAL NOT NULL DEFAULT 0,
		output_price REAL NOT NULL DEFAULT 0, max_context_tokens INTEGER DEFAULT 0,
		max_completion_tokens INTEGER DEFAULT 0, supports_text INTEGER NOT NULL DEFAULT 1,
		supports_image INTEGER NOT NULL DEFAULT 0, supports_video INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	db.MustExec(`CREATE TABLE request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT, api_key_id INTEGER NOT NULL DEFAULT 0,
		format TEXT NOT NULL, provider TEXT NOT NULL DEFAULT '', path TEXT NOT NULL,
		status_code INTEGER DEFAULT 0, request_headers TEXT DEFAULT '', request_body TEXT DEFAULT '',
		response_body TEXT DEFAULT '', model TEXT DEFAULT '', input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0, total_tokens INTEGER DEFAULT 0, error TEXT DEFAULT '',
		latency_ms INTEGER DEFAULT 0, created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	sum := sha256.Sum256([]byte("sk-test"))
	db.MustExec(`INSERT INTO users (name, account, password, unlimited) VALUES ('t', 't', 'x', 1)`)
	db.MustExec(`INSERT INTO api_keys (user_id, key_hash) VALUES (1, ?)`, hex.EncodeToString(sum[:]))
	db.MustExec(`INSERT INTO models (provider, model) VALUES ('default', 'gpt-4o')`)
	if err := store.Init(db); err != nil {
		t.Fatalf("store init: %v", err)
	}

	e := echo.New()
	EchoInit(e)
	return e
}

func serve(e *echo.Echo, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRouteModelsEndpointAnthropic(t *testing.T) {
	e := setupEchoTest(t)

	// anthropic 格式同路径 → 模型列表入口，响应为 Anthropic 格式；鉴权走 x-api-key 头
	req := httptest.NewRequest(http.MethodGet, "/proxy/anthropic/default/v1/models", nil)
	req.Header.Set("x-api-key", "sk-test")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expect 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"data"`
		FirstID *string `json:"first_id"`
		HasMore bool    `json:"has_more"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Type != "model" || resp.Data[0].ID != "gpt-4o" {
		t.Fatalf("unexpected data: %s", rec.Body.String())
	}
	if resp.FirstID == nil || *resp.FirstID != "gpt-4o" || resp.HasMore {
		t.Fatalf("unexpected envelope: %s", rec.Body.String())
	}
}

func TestRouteModelsEndpoint(t *testing.T) {
	e := setupEchoTest(t)

	// GET v1/models → 模型列表入口，返回本地模型列表
	rec := serve(e, http.MethodGet, "/proxy/openai/default/v1/models", "Bearer sk-test")
	if rec.Code != 200 {
		t.Fatalf("expect 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Object != "list" || len(resp.Data) != 1 || resp.Data[0].ID != "gpt-4o" {
		t.Fatalf("unexpected models response: %s", rec.Body.String())
	}

	// 模型列表端点不写请求日志
	var cnt int
	if err := store.C().Query(`SELECT COUNT(*) FROM request_logs`, nil).Get(&cnt); err != nil {
		t.Fatalf("count request logs: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("models endpoint should not write request_logs, got %d rows", cnt)
	}
}

func TestRouteModelsEndpointFallback(t *testing.T) {
	e := setupEchoTest(t)

	// POST v1/models 不匹配具体路由 → 落回转发管道，body 无 model → 404 模型不存在
	rec := serve(e, http.MethodPost, "/proxy/openai/default/v1/models", "Bearer sk-test")
	if rec.Code != 404 {
		t.Fatalf("expect 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "model not found") {
		t.Fatalf("expect forward pipeline error, got %s", rec.Body.String())
	}

	// GET v1/models 无 Key → 401
	rec = serve(e, http.MethodGet, "/proxy/openai/default/v1/models", "")
	if rec.Code != 401 {
		t.Fatalf("expect 401, got %d", rec.Code)
	}

	// 转发链路仍写请求日志：上述 3 个请求中仅 POST（转发链路）落一条
	var cnt int
	if err := store.C().Query(`SELECT COUNT(*) FROM request_logs`, nil).Get(&cnt); err != nil {
		t.Fatalf("count request logs: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expect 1 request log from forward pipeline, got %d", cnt)
	}
}

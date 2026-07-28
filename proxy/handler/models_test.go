package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
)

// capWriter 实现 types.ProxyResponseWrite，捕获状态码与响应体
type capWriter struct {
	header http.Header
	status int
	body   []byte
}

func (w *capWriter) Header() http.Header            { return w.header }
func (w *capWriter) WriteStatusCode(statusCode int) { w.status = statusCode }
func (w *capWriter) Write(body []byte) (int, error) { w.body = append(w.body, body...); return len(body), nil }

// setupModelsTestDB 初始化内存 SQLite：default 下 2 个模型，other 下 1 个；key 7 为 whitelist
func setupModelsTestDB(t *testing.T) {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.MustExec(`CREATE TABLE models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		input_cache_hit_price REAL NOT NULL DEFAULT 0,
		input_cache_miss_price REAL NOT NULL DEFAULT 0,
		output_price REAL NOT NULL DEFAULT 0,
		max_context_tokens INTEGER DEFAULT 0,
		max_completion_tokens INTEGER DEFAULT 0,
		supports_text INTEGER NOT NULL DEFAULT 1,
		supports_image INTEGER NOT NULL DEFAULT 0,
		supports_video INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	db.MustExec(`CREATE TABLE api_keys (id INTEGER PRIMARY KEY, model_policy TEXT)`)
	db.MustExec(`CREATE TABLE apikey_model_access (api_key_id INTEGER, model_id INTEGER)`)
	db.MustExec(`INSERT INTO models (provider, model) VALUES
		('default', 'gpt-4o'), ('default', 'gpt-4o-mini'), ('other', 'claude-sonnet')`)
	db.MustExec(`INSERT INTO api_keys (id, model_policy) VALUES (7, 'whitelist')`)
	db.MustExec(`INSERT INTO apikey_model_access (api_key_id, model_id) VALUES (7, 2)`)
	if err := store.Init(db); err != nil {
		t.Fatalf("store init: %v", err)
	}
}

func TestListModels_WhitelistFiltered(t *testing.T) {
	setupModelsTestDB(t)

	w := &capWriter{header: http.Header{}}
	ctx := &types.Context{
		P:            parser.OpenAI,
		Writer:       w,
		ProviderType: "default",
		ApiKeyID:     7, // whitelist 仅授权 id=2（default/gpt-4o-mini）
	}
	ListModels(ctx)

	if ctx.Err != nil {
		t.Fatalf("unexpected err: %v", ctx.Err)
	}
	if w.status != 200 {
		t.Fatalf("expect status 200, got %d", w.status)
	}

	var resp struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.body, &resp); err != nil {
		t.Fatalf("invalid json: %v, body: %s", err, w.body)
	}
	if resp.Object != "list" {
		t.Fatalf("expect object=list, got %q", resp.Object)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != "gpt-4o-mini" || resp.Data[0].Object != "model" || resp.Data[0].OwnedBy != "default" {
		t.Fatalf("unexpected data: %+v", resp.Data)
	}
}

func TestListModels_EmptyResult(t *testing.T) {
	setupModelsTestDB(t)

	w := &capWriter{header: http.Header{}}
	ctx := &types.Context{
		P:            parser.OpenAI,
		Writer:       w,
		ProviderType: "nonexist", // 无模型的 provider
		ApiKeyID:     7,
	}
	ListModels(ctx)

	if ctx.Err != nil {
		t.Fatalf("unexpected err: %v", ctx.Err)
	}
	if string(w.body) != `{"object":"list","data":[]}` {
		t.Fatalf("expect empty list, got %s", w.body)
	}
}

func TestListModels_AnthropicFormat(t *testing.T) {
	setupModelsTestDB(t)

	w := &capWriter{header: http.Header{}}
	ctx := &types.Context{
		P:            parser.Anthropic,
		Writer:       w,
		ProviderType: "default",
		ApiKeyID:     7, // whitelist 仅授权 id=2（default/gpt-4o-mini）
	}
	ListModels(ctx)

	if ctx.Err != nil {
		t.Fatalf("unexpected err: %v", ctx.Err)
	}
	var resp struct {
		Data []struct {
			Type        string `json:"type"`
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			CreatedAt   string `json:"created_at"`
		} `json:"data"`
		FirstID *string `json:"first_id"`
		LastID  *string `json:"last_id"`
		HasMore bool    `json:"has_more"`
	}
	if err := json.Unmarshal(w.body, &resp); err != nil {
		t.Fatalf("invalid json: %v, body: %s", err, w.body)
	}
	if len(resp.Data) != 1 || resp.Data[0].Type != "model" || resp.Data[0].ID != "gpt-4o-mini" || resp.Data[0].DisplayName != "gpt-4o-mini" {
		t.Fatalf("unexpected data: %+v", resp.Data)
	}
	if resp.FirstID == nil || *resp.FirstID != "gpt-4o-mini" || resp.LastID == nil || *resp.LastID != "gpt-4o-mini" {
		t.Fatalf("first_id/last_id mismatch: %+v", resp)
	}
	if resp.HasMore {
		t.Fatal("has_more should be false")
	}
}

func TestListModels_AnthropicEmptyIDs(t *testing.T) {
	setupModelsTestDB(t)

	// 空列表时 first_id/last_id 应为 null
	w := &capWriter{header: http.Header{}}
	ctx := &types.Context{
		P:            parser.Anthropic,
		Writer:       w,
		ProviderType: "nonexist",
		ApiKeyID:     7,
	}
	ListModels(ctx)

	if ctx.Err != nil {
		t.Fatalf("unexpected err: %v", ctx.Err)
	}
	if string(w.body) != `{"data":[],"first_id":null,"last_id":null,"has_more":false}` {
		t.Fatalf("unexpected empty response: %s", w.body)
	}
}

func TestListModels_NilParserFallback(t *testing.T) {
	setupModelsTestDB(t)

	// 未知 format 时 P 为 nil，回退 OpenAI 格式（实际流程中会在 AuthKey 阶段 401，此处仅验证不 panic）
	w := &capWriter{header: http.Header{}}
	ctx := &types.Context{
		Writer:       w,
		ProviderType: "default",
		ApiKeyID:     7,
	}
	ListModels(ctx)

	if ctx.Err != nil {
		t.Fatalf("unexpected err: %v", ctx.Err)
	}
	var resp struct {
		Object string `json:"object"`
	}
	if err := json.Unmarshal(w.body, &resp); err != nil || resp.Object != "list" {
		t.Fatalf("expect openai shape, got %s err=%v", w.body, err)
	}
}

package service

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/store"
)

// setupModelTestDB 初始化内存 SQLite 及测试数据：3 个模型（default 下 2 个、other 下 1 个）
func setupModelTestDB(t *testing.T) {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 内存库多连接会各自建库，限制单连接
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
	// key 行必须存在，UpdateKeyPolicy 才能生效（生产流程 key 先于策略存在）
	db.MustExec(`INSERT INTO api_keys (id, model_policy) VALUES (1, ''), (2, ''), (3, '')`)
	if err := store.Init(db); err != nil {
		t.Fatalf("store init: %v", err)
	}
}

func TestListAvailableModels_AllPolicy(t *testing.T) {
	setupModelTestDB(t)
	// key 1：all 策略
	if err := store.C().ModelAccess().UpdateKeyPolicy(1, store.ModelPolicyAll); err != nil {
		t.Fatalf("seed policy: %v", err)
	}

	models, err := NewModelService().ListAvailableModels("default", 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expect 2 models, got %d", len(models))
	}
	// 按 model 排序
	if models[0].Model != "gpt-4o" || models[1].Model != "gpt-4o-mini" {
		t.Fatalf("unexpected models: %+v", models)
	}
}

func TestListAvailableModels_Whitelist(t *testing.T) {
	setupModelTestDB(t)
	// key 2：whitelist，授权 default/gpt-4o(id=1) 与 other/claude-sonnet(id=3)
	if err := NewApiKeyService().SetModelAccess(2, store.ModelPolicyWhitelist, []int64{1, 3}); err != nil {
		t.Fatalf("seed whitelist: %v", err)
	}

	// default provider：只返回 gpt-4o（id=3 属于 other，被过滤）
	models, err := NewModelService().ListAvailableModels("default", 2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(models) != 1 || models[0].Model != "gpt-4o" {
		t.Fatalf("expect [gpt-4o], got %+v", models)
	}

	// other provider：只返回 claude-sonnet
	models, err = NewModelService().ListAvailableModels("other", 2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(models) != 1 || models[0].Model != "claude-sonnet" {
		t.Fatalf("expect [claude-sonnet], got %+v", models)
	}
}

func TestListAvailableModels_EmptyWhitelist(t *testing.T) {
	setupModelTestDB(t)
	// key 3：whitelist 但白名单为空
	if err := NewApiKeyService().SetModelAccess(3, store.ModelPolicyWhitelist, nil); err != nil {
		t.Fatalf("seed whitelist: %v", err)
	}

	models, err := NewModelService().ListAvailableModels("default", 3)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("expect empty, got %+v", models)
	}
}

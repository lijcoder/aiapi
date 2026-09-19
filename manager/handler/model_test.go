package handler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/store"
)

// setupModelHandlerDB 初始化内存 SQLite：一条 provider_model 与 model 不同的别名，
// 一条 provider_model 为空（存量数据）
func setupModelHandlerDB(t *testing.T) {
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
		provider_model TEXT NOT NULL DEFAULT '',
		pricing_config TEXT NOT NULL DEFAULT '',
		max_context_tokens INTEGER DEFAULT 0,
		max_completion_tokens INTEGER DEFAULT 0,
		supports_text INTEGER NOT NULL DEFAULT 1,
		supports_image INTEGER NOT NULL DEFAULT 0,
		supports_video INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	db.MustExec(`INSERT INTO models (provider, model, provider_model) VALUES
		('default', 'gpt-4o', 'gpt-4o-2024-08-06'), ('default', 'legacy', '')`)
	if err := store.Init(db); err != nil {
		t.Fatalf("store init: %v", err)
	}
}

// TestListModels_HidesProviderModelFromUsers 普通用户接口不返回 provider_model
func TestListModels_HidesProviderModelFromUsers(t *testing.T) {
	setupModelHandlerDB(t)

	res, bizErr := ListModels(context.Background(), &ListModelsReq{})
	if bizErr != nil {
		t.Fatalf("list: %+v", bizErr)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expect 2 items, got %d", len(res.Items))
	}
	for _, item := range res.Items {
		if item.Model == "" {
			t.Fatalf("model name missing: %+v", item)
		}
	}
	body, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(body), "provider_model") || strings.Contains(string(body), "gpt-4o-2024-08-06") {
		t.Fatalf("provider model leaked to user response: %s", body)
	}
}

// TestListModelsAdmin_ShowsEffectiveProviderModel 超管接口返回生效的上游模型名（存量空值回退为 model）
func TestListModelsAdmin_ShowsEffectiveProviderModel(t *testing.T) {
	setupModelHandlerDB(t)

	res, bizErr := ListModelsAdmin(context.Background(), &ListModelsAdminReq{})
	if bizErr != nil {
		t.Fatalf("list: %+v", bizErr)
	}
	got := map[string]string{}
	for _, item := range res.Items {
		got[item.Model] = item.ProviderModel
	}
	if got["gpt-4o"] != "gpt-4o-2024-08-06" {
		t.Fatalf("gpt-4o provider_model = %q, want gpt-4o-2024-08-06", got["gpt-4o"])
	}
	if got["legacy"] != "legacy" {
		t.Fatalf("legacy provider_model = %q, want legacy (fallback)", got["legacy"])
	}
}

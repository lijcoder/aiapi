package store

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/store/model"
)

// TestSchemaModelsRoundTrip 用 sql/sqlite.sql 的真实 DDL 建库，验证 models 表与数据模型一致。
// store 查询用 SELECT *，DDL 与 store/model 结构体不同步（如新增列漏定义字段）时这里会失败。
func TestSchemaModelsRoundTrip(t *testing.T) {
	ddl, err := os.ReadFile("../sql/sqlite.sql")
	if err != nil {
		t.Fatalf("read ddl: %v", err)
	}
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(string(ddl)); err != nil {
		t.Fatalf("apply ddl: %v", err)
	}
	if err := Init(db); err != nil {
		t.Fatalf("store init: %v", err)
	}

	m := &model.Model{Provider: "default", Model: "gpt-4o", ProviderModel: "gpt-4o-2024-08-06", PricingConfig: "{}"}
	if err := C().Model().Create(m); err != nil {
		t.Fatalf("create: %v", err)
	}
	stored, err := C().Model().Get("default", "gpt-4o")
	if err != nil || stored == nil {
		t.Fatalf("get: %v %v", stored, err)
	}
	if stored.ID != m.ID || stored.ProviderModel != "gpt-4o-2024-08-06" {
		t.Fatalf("stored = %+v", stored)
	}
}

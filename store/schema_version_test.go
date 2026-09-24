package store

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/constant"
)

// TestSchemaVersionMatchesConstant 校验 sql/sqlite.sql 写入 schema_meta 的版本
// 与 constant.SchemaVersion 一致——这是 schema 版本的唯一权威与 DDL 之间的机械约束，
// 改 schema 时漏改任何一处都会在这里失败。
func TestSchemaVersionMatchesConstant(t *testing.T) {
	ddl, err := os.ReadFile("../sql/sqlite.sql")
	if err != nil {
		t.Fatalf("读取 DDL 失败: %v", err)
	}
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(string(ddl)); err != nil {
		t.Fatalf("执行 DDL 失败: %v", err)
	}

	var got int
	if err := db.Get(&got, "SELECT version FROM schema_meta WHERE id = 1"); err != nil {
		t.Fatalf("读取 schema_meta 失败: %v", err)
	}
	if got != constant.SchemaVersion {
		t.Fatalf("sql/sqlite.sql 写入的版本 = %d，constant.SchemaVersion = %d，两者必须一致", got, constant.SchemaVersion)
	}
}

// TestSchemaDDLIsIdempotent 校验整份 DDL 可以重复执行（建表用 IF NOT EXISTS、版本行幂等写入）。
// 运维在存量库上重跑 sql/sqlite.sql 是允许的操作，不能因此报错或写入重复版本行。
func TestSchemaDDLIsIdempotent(t *testing.T) {
	ddl, err := os.ReadFile("../sql/sqlite.sql")
	if err != nil {
		t.Fatalf("读取 DDL 失败: %v", err)
	}
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	for i := 1; i <= 2; i++ {
		if _, err := db.Exec(string(ddl)); err != nil {
			t.Fatalf("第 %d 次执行 DDL 失败: %v", i, err)
		}
	}

	var rows int
	if err := db.Get(&rows, "SELECT COUNT(*) FROM schema_meta"); err != nil {
		t.Fatalf("统计版本行失败: %v", err)
	}
	if rows != 1 {
		t.Fatalf("schema_meta 有 %d 行，期望单行", rows)
	}
}

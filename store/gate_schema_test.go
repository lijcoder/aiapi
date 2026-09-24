// 门禁文件（不是行为测试）：断言**仓库自身**的 schema 定义与版本常量一致，不验证生产代码的行为。
//
//   - sql/sqlite.sql 写入 schema_meta 的版本 == constant.SchemaVersion
//   - 整份 DDL 可重复执行，且 schema_meta 保持单行
//
// 运行方式：`make gate`（只跑门禁，等价 go test -run '^TestGate' ./...）。
// 它同时留在 `go test ./...` 里——本仓库没有 CI，放进默认测试是让门禁不会被绕过的唯一保证。
//
// 约定：门禁文件以 gate_ 开头、测试函数以 TestGate 开头；store/driver/schema_test.go 是行为测试。

package store

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/constant"
)

// TestGateSchemaVersionMatchesConstant 校验 sql/sqlite.sql 写入 schema_meta 的版本
// 与 constant.SchemaVersion 一致——这是 schema 版本的唯一权威与 DDL 之间的机械约束，
// 改 schema 时漏改任何一处都会在这里失败。
func TestGateSchemaVersionMatchesConstant(t *testing.T) {
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

// TestGateSchemaDDLIsIdempotent 校验整份 DDL 可以重复执行（建表用 IF NOT EXISTS、版本行幂等写入）。
// 运维在存量库上重跑 sql/sqlite.sql 是允许的操作，不能因此报错或写入重复版本行。
func TestGateSchemaDDLIsIdempotent(t *testing.T) {
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

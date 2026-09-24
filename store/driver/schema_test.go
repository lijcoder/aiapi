package driver

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// TestClassifySchemaVersion 覆盖版本判定的全部分支。
func TestClassifySchemaVersion(t *testing.T) {
	cases := []struct {
		name     string
		current  int
		expected int
		want     SchemaStatus
		wantErr  string // 非空表示期望报错，值为错误信息应包含的片段
	}{
		{name: "版本一致", current: 3, expected: 3, want: SchemaMatch},
		{name: "版本为 0 视为未标记", current: 0, expected: 3, want: SchemaUnversioned},
		{name: "库比二进制旧", current: 2, expected: 3, wantErr: "older than"},
		{name: "库比二进制新", current: 4, expected: 3, wantErr: "newer than"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := classifySchemaVersion(tc.current, tc.expected)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("期望无错误，实际: %v", err)
				}
				if got != tc.want {
					t.Fatalf("状态 = %v，期望 %v", got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望报错，实际无错误（状态 %v）", got)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("错误信息 %q 未包含 %q", err.Error(), tc.wantErr)
			}
			// 错误信息必须给出可执行指引，否则运维无从下手
			if !strings.Contains(err.Error(), "sql/migrations/README.md") && !strings.Contains(err.Error(), "upgrade the binary") {
				t.Fatalf("错误信息缺少可执行指引: %q", err.Error())
			}
		})
	}
}

// newTestDB 返回一个内存库；withMeta 为 true 时先建 schema_meta 表。
func newTestDB(t *testing.T, withMeta bool) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	if withMeta {
		if _, err := db.Exec(`CREATE TABLE schema_meta (
			id INTEGER PRIMARY KEY, version INTEGER NOT NULL, applied_at DATETIME NOT NULL)`); err != nil {
			t.Fatalf("建 schema_meta 失败: %v", err)
		}
	}
	return db
}

// TestCheckSchemaVersionReadsMetadataTable 验证从 schema_meta 表读取版本号。
func TestCheckSchemaVersionReadsMetadataTable(t *testing.T) {
	db := newTestDB(t, true)
	if _, err := db.Exec("INSERT INTO schema_meta (id, version, applied_at) VALUES (1, 7, datetime('now'))"); err != nil {
		t.Fatalf("写入版本行失败: %v", err)
	}

	status, err := CheckSchemaVersion(db, 7)
	if err != nil {
		t.Fatalf("同版本应通过: %v", err)
	}
	if status != SchemaMatch {
		t.Fatalf("状态 = %v，期望 SchemaMatch", status)
	}

	if _, err := CheckSchemaVersion(db, 8); err == nil {
		t.Fatal("库比二进制旧时应报错")
	}
	if _, err := CheckSchemaVersion(db, 6); err == nil {
		t.Fatal("库比二进制新时应报错")
	}
}

// TestCheckSchemaVersionWithoutMetaTable 验证历史库（还没有 schema_meta 表）被判为未标记而不是硬失败。
// 这是引入版本标记之前建的库的升级路径：必须放行启动，只告警。
func TestCheckSchemaVersionWithoutMetaTable(t *testing.T) {
	db := newTestDB(t, false)
	status, err := CheckSchemaVersion(db, 1)
	if err != nil {
		t.Fatalf("缺元数据表时不应硬失败（历史库要能启动）: %v", err)
	}
	if status != SchemaUnversioned {
		t.Fatalf("状态 = %v，期望 SchemaUnversioned", status)
	}
}

// TestCheckSchemaVersionWithoutVersionRow 验证表在但没有版本行的情况。
func TestCheckSchemaVersionWithoutVersionRow(t *testing.T) {
	db := newTestDB(t, true)
	status, err := CheckSchemaVersion(db, 1)
	if err != nil {
		t.Fatalf("无版本行时不应报错: %v", err)
	}
	if status != SchemaUnversioned {
		t.Fatalf("状态 = %v，期望 SchemaUnversioned", status)
	}
}

// TestCheckSchemaVersionReportsConnectionFailure 验证真实故障（连探活都失败）仍然硬失败，
// 不会被当成"未标记"而放行。
func TestCheckSchemaVersionReportsConnectionFailure(t *testing.T) {
	db := newTestDB(t, false)
	db.Close() // 关闭连接：SELECT 1 也会失败

	if _, err := CheckSchemaVersion(db, 1); err == nil {
		t.Fatal("连接故障时必须硬失败")
	}
}

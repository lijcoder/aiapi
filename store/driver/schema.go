package driver

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// SchemaStatus 描述数据库标记的 schema 版本与当前二进制的兼容关系。
type SchemaStatus int

const (
	// SchemaMatch 库与二进制版本一致，可直接启动。
	SchemaMatch SchemaStatus = iota
	// SchemaUnversioned 库里没有版本标记（没有 schema_meta 表，或表里没有版本行）。
	// 兼容启动并告警：可能是刚建的新库，也可能是本项目引入版本标记之前的历史库，
	// 无法自动判定，由运维核对 sql/migrations/README.md 后补标记。
	SchemaUnversioned
)

// schemaVersionQuery 读取 schema 版本的查询。
//
// 版本存在 schema_meta 单行表里，而不是 SQLite 专有的 PRAGMA user_version——
// 本项目要兼容 MySQL / PostgreSQL（两者都没有 user_version 等价物）。
// 这条 SELECT 与探活用的 SELECT 1 都是三家公共语法，读取路径不需要按驱动分支。
const schemaVersionQuery = "SELECT version FROM schema_meta WHERE id = 1"

// CheckSchemaVersion 读取数据库记录的 schema 版本并与期望版本比对。
//
// 返回错误表示不可启动：库比二进制旧（缺少迁移）或比二进制新（二进制过旧，降级有风险）。
// 没有版本标记时返回 SchemaUnversioned 且不报错，由调用方决定如何提示。
func CheckSchemaVersion(db *sqlx.DB, expected int) (SchemaStatus, error) {
	var current int
	err := db.Get(&current, schemaVersionQuery)
	if err == nil {
		return classifySchemaVersion(current, expected)
	}
	if errors.Is(err, sql.ErrNoRows) {
		// 表在但没有 id=1 的版本行
		return SchemaUnversioned, nil
	}

	// 查询失败要区分两种情况：元数据表不存在（历史库，应放行并告警）vs 连接/权限故障（必须拦截）。
	// 不解析各驱动的错误码（SQLite "no such table" / MySQL 1146 / PostgreSQL 42P01 各不相同），
	// 改用一次可移植的探活查询：连 SELECT 1 都失败说明是真实故障。
	var probe int
	if probeErr := db.Get(&probe, "SELECT 1"); probeErr != nil {
		return SchemaMatch, fmt.Errorf("read schema version: %w", err)
	}
	return SchemaUnversioned, nil
}

// classifySchemaVersion 是版本判定的纯函数部分（便于单测）。
func classifySchemaVersion(current, expected int) (SchemaStatus, error) {
	switch {
	case current == expected:
		return SchemaMatch, nil
	case current <= 0:
		return SchemaUnversioned, nil
	case current < expected:
		return SchemaMatch, fmt.Errorf(
			"database schema v%d is older than this binary's v%d: apply the missing migrations from sql/migrations/README.md, then update schema_meta to v%d",
			current, expected, expected)
	default:
		return SchemaMatch, fmt.Errorf(
			"database schema v%d is newer than this binary's v%d: upgrade the binary instead of downgrading it",
			current, expected)
	}
}

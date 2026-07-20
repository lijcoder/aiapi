package base

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// QueryBuilder 通用查询构造器
// 统一封装 Named + In + Rebind 三步管道，支持 DB 和 Tx
type QueryBuilder struct {
	ext     sqlx.Ext        // 底层执行器：*sqlx.DB 或 *sqlx.Tx
	bindTyp int             // 占位符类型：sqlx.QUESTION / sqlx.DOLLAR / sqlx.AMPERAND
	query   string          // 处理后的最终 SQL（已展开 IN、已转换占位符）
	args    []interface{}   // 处理后的最终参数列表
	err     error           // 预处理阶段的累积错误（执行时优先返回）
}

// NewQuery 从 sqlx.Ext 创建 QueryBuilder
// driverName 用于确定占位符类型；DB 场景可传 db.DriverName()，Tx 场景需调用方手动传入
func NewQuery(ext sqlx.Ext, driverName string, query string, params any) *QueryBuilder {
	qb := &QueryBuilder{ext: ext}
	if params == nil {
		params = map[string]any{}
	}

	// 第 1 步：Named — 将 :name 替换为 ?，并按参数顺序排列参数值
	bound, args, err := sqlx.Named(query, params)
	if err != nil {
		qb.err = fmt.Errorf("sqlx.Named failed: %w", err)
		return qb
	}

	// 第 2 步：In — 将 IN (?) 展开为 IN (?, ?, ?)
	expanded, args, err := sqlx.In(bound, args...)
	if err != nil {
		qb.err = fmt.Errorf("sqlx.In failed: %w", err)
		return qb
	}

	// 第 3 步：Rebind — 将 ? 转换为目标驱动的占位符格式
	bindTyp := sqlx.BindType(driverName)
	expanded = sqlx.Rebind(bindTyp, expanded)

	qb.bindTyp = bindTyp
	qb.query = expanded
	qb.args = args
	return qb
}

// SQL 返回处理后的最终 SQL 语句（调试用）
func (qb *QueryBuilder) SQL() (string, error) {
	if qb.err != nil {
		return "", qb.err
	}
	return qb.query, nil
}

// Args 返回处理后的最终参数列表（调试用）
func (qb *QueryBuilder) Args() ([]interface{}, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	return qb.args, nil
}

// Err 返回预处理阶段的错误（如果有）
func (qb *QueryBuilder) Err() error {
	return qb.err
}

// BindType 返回当前占位符类型（调试用）
func (qb *QueryBuilder) BindType() int {
	return qb.bindTyp
}

// Get 执行查询并将单行结果绑定到 dest
// dest 可以是 *struct 或 *map[string]interface{}
// 无结果时返回 sql.ErrNoRows
func (qb *QueryBuilder) Get(dest interface{}) error {
	if qb.err != nil {
		return qb.err
	}
	return sqlx.Get(qb.ext, dest, qb.query, qb.args...)
}

// Select 执行查询并将多行结果绑定到 dest
// dest 必须是 *[]struct 或 *[]map[string]interface{}
func (qb *QueryBuilder) Select(dest interface{}) error {
	if qb.err != nil {
		return qb.err
	}
	return sqlx.Select(qb.ext, dest, qb.query, qb.args...)
}

// Exec 执行 INSERT / UPDATE / DELETE
// 返回 sql.Result，包含 LastInsertId 和 RowsAffected
func (qb *QueryBuilder) Exec() (sql.Result, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	return qb.ext.Exec(qb.query, qb.args...)
}

// Query 执行原始查询，返回 *sql.Rows，由调用方自行处理
func (qb *QueryBuilder) Query() (*sql.Rows, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	return qb.ext.Query(qb.query, qb.args...)
}

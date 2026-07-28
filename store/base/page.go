package base

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// PageReq 分页请求参数。handler 入参结构体内嵌它即可获得分页能力。
//   - Page：页码，1-based；<1 按 1 处理
//   - PageSize：每页条数；<=0 用默认 20，>100 截断为 100
type PageReq struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Normalize 规范化分页参数，返回合法的 page/size 与计算好的 offset。
func (p PageReq) Normalize() (page, size, offset int) {
	page = p.Page
	if page < 1 {
		page = 1
	}
	size = p.PageSize
	if size <= 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	offset = (page - 1) * size
	return
}

// PageResult 分页结果。泛型承载列表元素类型，Total 为符合查询条件的总条数。
type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// PageContext 分页上下文，既承载入参（Page/PageSize）又承载输出（Total）。
// 由调用方创建，经 Session.SetPage 设置到 Session，拦截器写回 Total。
type PageContext struct {
	Page     int
	PageSize int
	Total    int64 // 输出，由拦截器在 Select 后写入
}

// countSQLFor 把业务查询包装成 COUNT 查询：SELECT COUNT(*) FROM (<query>) t。
// 子查询里的 ORDER BY 不影响 COUNT（SQLite/MySQL 优化器会消除）。
// 注意：带无关 JOIN 时性能不如手写精简 COUNT，后续可引入 SQL 解析优化。
func countSQLFor(query string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM (%s) t", query)
}

// applyPage 执行分页查询：先 COUNT 总数写回 pc.Total，再追加 LIMIT/OFFSET 查当前页。
// 由 QueryBuilder.Select 在检测到 PageContext 时调用。
// qb.query 是已 Rebind 的最终 SQL（占位符为 ?），qb.args 是位置参数，直接复用。
func applyPage(qb *QueryBuilder, dest interface{}, pc *PageContext) error {
	// 1. COUNT 总数（子查询包装，复用 qb 的 query/args）
	var total int64
	if err := sqlx.Get(qb.ext, &total, countSQLFor(qb.query), qb.args...); err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		total = 0
	}
	pc.Total = total

	// 2. 规范化分页参数并写回 pc（供调用方读取实际生效的 page/size）
	page, size, offset := PageReq{Page: pc.Page, PageSize: pc.PageSize}.Normalize()
	pc.Page = page
	pc.PageSize = size

	// 3. 追加 LIMIT/OFFSET 查当前页（位置参数追加 size/offset）
	pagedSQL := qb.query + " LIMIT ? OFFSET ?"
	return sqlx.Select(qb.ext, dest, pagedSQL, append(qb.args, size, offset)...)
}

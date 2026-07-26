package base

// PageReq 分页请求参数，handler 入参结构体内嵌它即可获得分页能力。
//   - Page：页码，1-based；<1 按 1 处理
//   - PageSize：每页条数；<=0 用默认 20，>100 截断为 100
type PageReq struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PageResult 分页结果，handler 返回值。
type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

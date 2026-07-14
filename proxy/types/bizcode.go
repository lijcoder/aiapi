package types

// BizCode 业务错误码，每个码直接关联其 HTTP 状态码
type BizCode struct {
	ID         int
	httpStatus int
}

// HTTPStatus 返回业务码对应的 HTTP 状态码
func (c BizCode) HTTPStatus() int {
	if c.httpStatus == 0 {
		return 500
	}
	return c.httpStatus
}

var (
	CodeUnknown  = BizCode{0, 500}
	CodeNotFound = BizCode{1001, 404}
)

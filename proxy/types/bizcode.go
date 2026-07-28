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

// IsZero 判断 BizCode 是否未被显式设置（零值）
func (c BizCode) IsZero() bool {
	return c == BizCode{}
}

var (
	CodeUnknown             = BizCode{0, 500}
	CodeNotFound            = BizCode{1001, 404}
	CodeUnauthorized        = BizCode{1002, 401}
	CodeModelNotFound       = BizCode{1003, 404}
	CodeInsufficientBalance = BizCode{1004, 402}
)

const InternalServerError = "internal server error"

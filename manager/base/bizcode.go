package base

// BizCode manager 业务错误码，每个码关联其 HTTP 状态码。
// 与 proxy/types/bizcode.go 解耦，两套独立编号，互不影响。
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
	CodeSuccess        = BizCode{0, 200}
	CodeUnknown        = BizCode{1, 500}
	CodeBadRequest     = BizCode{1001, 400}
	CodeUnauthorized   = BizCode{1002, 401}
	CodeForbidden      = BizCode{1003, 403}
	CodeUserNotFound   = BizCode{1004, 404}
	CodeWrongPassword  = BizCode{1005, 401}
	CodeSessionExpired = BizCode{1006, 401}
	CodeUserDisabled   = BizCode{1007, 403}
	CodeInvalidParams  = BizCode{1008, 400}
	CodeApiKeyNotFound = BizCode{1009, 404}
	CodeBudgetExceeded = BizCode{1010, 400}
)

const InternalServerError = "internal server error"

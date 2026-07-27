package base

// BizCode manager 业务错误码，每个码关联其 HTTP 状态码。
// 与 proxy/types/bizcode.go 解耦，两套独立编号，互不影响。
//
// 只保留有消费方按 code 分支、或 HTTP 状态语义不同的码；
// 业务错误统一用通用码 + 中文错误信息区分，不为每种业务定义独立码。
// 前端唯一按 code 分支的是 CodeTokenExpired(1016)：据此触发 /refresh 续期，编号不可变更。
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
	CodeSuccess      = BizCode{0, 200}
	CodeUnknown      = BizCode{1, 500}    // 内部错误（DB 异常等），统一用 ErrInternal
	CodeBadRequest   = BizCode{1001, 400} // 参数错误 + 业务校验失败（已存在/额度不足/密码错误等）
	CodeUnauthorized = BizCode{1002, 401} // 未登录 / 会话失效 / refresh 失败
	CodeForbidden    = BizCode{1003, 403} // 无接口权限 / 用户被禁用
	CodeNotFound     = BizCode{1004, 404} // 资源不存在（用户/Key/Provider/Model）
	CodeTokenExpired = BizCode{1016, 401} // access JWT 过期，前端据此触发 refresh
)

// ErrInternal 内部错误统一实例，handler 中 DB 等内部失败直接返回。
// BizError 构造后不再修改，可安全共享。
var ErrInternal = NewBizError(CodeUnknown, "系统繁忙，请稍后重试")

// ErrBadReq 构造 400 业务错误：参数错误、业务校验失败等
func ErrBadReq(msg string) *BizError { return NewBizError(CodeBadRequest, msg) }

// ErrNotFound 构造 404 业务错误：资源不存在
func ErrNotFound(msg string) *BizError { return NewBizError(CodeNotFound, msg) }

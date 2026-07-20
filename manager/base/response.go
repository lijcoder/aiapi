package base

import (
	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
)

// BizError 业务错误，携带业务码与消息。业务函数返回 *BizError 表示失败。
type BizError struct {
	Code BizCode
	Msg  string
}

// Error 实现 error 接口
func (e *BizError) Error() string { return e.Msg }

// NewBizError 构造业务错误
func NewBizError(code BizCode, msg string) *BizError {
	return &BizError{Code: code, Msg: msg}
}

// Ok 输出成功响应
func Ok(c echo.Context, data any) error {
	return c.JSON(CodeSuccess.HTTPStatus(), constant.BuildHttpResponseSuccess(data))
}

// Fail 输出失败响应，HTTP 状态由 bizcode 决定
func Fail(c echo.Context, code BizCode, msg string) error {
	return c.JSON(code.HTTPStatus(), constant.HttpGeneralResp{
		Code: code.ID,
		Msg:  msg,
	})
}

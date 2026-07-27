package base

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"

	"github.com/labstack/echo/v4"
)

// BindJSON 从请求体解析 JSON 到 req；空 body 视为零值，不报错。
// 不依赖 Content-Type，统一按 JSON 处理。
func BindJSON(c echo.Context, req any) error {
	r := c.Request()
	if r == nil || r.Body == nil {
		return nil
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil
	}
	return json.Unmarshal(b, req)
}

// ===== 动态参数 Wrap =====
//
// Wrap 只做"参数包装"：把任意业务函数适配成 echo.HandlerFunc。
// 登录态校验与接口级权限校验由 manager/middleware.Auth 中间件负责，Wrap 不再做。
// 业务函数的入参可以是以下任意组合/顺序：
//   - echo.Context       → 注入当前 echo 上下文
//   - context.Context    → 注入携带登录态的请求 context
//   - *Req（指针到结构体）→ new 出零值 + BindJSON 绑定请求体 + 传指针
//   - Req （结构体值）    → new 出零值 + BindJSON 绑定请求体 + 传值
// 返回值固定为 (Resp, *BizError)。
//
// 签名在注册时（Wrap 调用时）一次性校验并预存参数计划，请求时按计划组装实参再 reflect.Call。
// 不支持的参数类型或返回值形状会在启动期 panic，不会拖到运行期。

var (
	echoCtxType = reflect.TypeOf((*echo.Context)(nil)).Elem()
	stdCtxType  = reflect.TypeOf((*context.Context)(nil)).Elem()
	bizErrType  = reflect.TypeOf((*BizError)(nil))
)

type paramKind int

const (
	kindEcho paramKind = iota
	kindCtx
	kindReqPtr
	kindReqValue
)

type paramSpec struct {
	kind paramKind
	typ  reflect.Type // req 参数的结构体类型（指针时为 Elem）
}

type handlerPlan struct {
	fv    reflect.Value
	ins   []paramSpec
	numIn int
}

// Wrap 包装业务函数。biz 必须是函数，返回 (Resp, *BizError)。
func Wrap(biz any) echo.HandlerFunc {
	fv := reflect.ValueOf(biz)
	ft := fv.Type()
	if ft.Kind() != reflect.Func {
		panic("manager/base.Wrap: biz must be a function")
	}
	if ft.NumOut() != 2 || ft.Out(1) != bizErrType {
		panic("manager/base.Wrap: biz must return (Resp, *base.BizError)")
	}
	p := &handlerPlan{fv: fv, numIn: ft.NumIn()}
	reqCount := 0
	for i := 0; i < ft.NumIn(); i++ {
		in := ft.In(i)
		switch {
		case in == echoCtxType:
			p.ins = append(p.ins, paramSpec{kind: kindEcho})
		case in == stdCtxType:
			p.ins = append(p.ins, paramSpec{kind: kindCtx})
		case in.Kind() == reflect.Ptr && in.Elem().Kind() == reflect.Struct:
			p.ins = append(p.ins, paramSpec{kind: kindReqPtr, typ: in.Elem()})
			reqCount++
		case in.Kind() == reflect.Struct:
			p.ins = append(p.ins, paramSpec{kind: kindReqValue, typ: in})
			reqCount++
		default:
			panic("manager/base.Wrap: unsupported param type " + in.String())
		}
	}
	if reqCount > 1 {
		panic("manager/base.Wrap: at most one request-struct parameter allowed")
	}
	return p.serve
}

func (p *handlerPlan) serve(c echo.Context) error {
	ctx := c.Request().Context()
	args := make([]reflect.Value, p.numIn)
	for i, spec := range p.ins {
		switch spec.kind {
		case kindEcho:
			args[i] = reflect.ValueOf(c)
		case kindCtx:
			args[i] = reflect.ValueOf(ctx)
		case kindReqPtr, kindReqValue:
			req := reflect.New(spec.typ)
			if err := BindJSON(c, req.Interface()); err != nil {
				return Fail(c, CodeBadRequest, "请求体格式错误")
			}
			if spec.kind == kindReqPtr {
				args[i] = req
			} else {
				args[i] = req.Elem()
			}
		}
	}
	outs := p.fv.Call(args)
	berr, _ := outs[1].Interface().(*BizError)
	if berr != nil {
		return Fail(c, berr.Code, berr.Msg)
	}
	return Ok(c, outs[0].Interface())
}

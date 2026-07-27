package test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store/model"
)

type testReq struct {
	Name string `json:"name"`
}

func permFor(path string) []model.RolePermission {
	return []model.RolePermission{{Entity: base.EntityAPI, Action: "*", Value: path}}
}

func newUserCtx(path, body string, authed bool) echo.Context {
	e := echo.New()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath(path)
	ctx := req.Context()
	if authed {
		ctx = base.WithAuth(ctx, &model.User{ID: 7, Account: "u"}, permFor(path))
	}
	c.SetRequest(req.WithContext(ctx))
	return c
}

func respBody(c echo.Context) string {
	return c.Response().Writer.(*httptest.ResponseRecorder).Body.String()
}

func decodeData(t *testing.T, body string) any {
	t.Helper()
	var out struct {
		Code int    `json:"code"`
		Data any    `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("unmarshal %q: %v", body, err)
	}
	return out.Data
}

func TestWrap_ReqAndCtx(t *testing.T) {
	biz := func(ctx context.Context, req *testReq) (string, *base.BizError) {
		if base.CurrentUser(ctx).ID != 7 {
			return "", base.NewBizError(base.CodeUnknown, "no user")
		}
		return req.Name + "-ok", nil
	}
	c := newUserCtx("/t", `{"name":"x"}`, true)
	if err := base.Wrap(biz)(c); err != nil {
		t.Fatal(err)
	}
	if d := decodeData(t, respBody(c)); d != "x-ok" {
		t.Fatalf("got %v", d)
	}
}

func TestWrap_NoReq(t *testing.T) {
	biz := func(ctx context.Context) (string, *base.BizError) { return "me", nil }
	c := newUserCtx("/t", "", true)
	if err := base.Wrap(biz)(c); err != nil {
		t.Fatal(err)
	}
	if d := decodeData(t, respBody(c)); d != "me" {
		t.Fatalf("got %v", d)
	}
}

func TestWrap_EchoContextParam(t *testing.T) {
	biz := func(c echo.Context) (string, *base.BizError) { return c.Path(), nil }
	cc := newUserCtx("/t", "", true)
	if err := base.Wrap(biz)(cc); err != nil {
		t.Fatal(err)
	}
	if d := decodeData(t, respBody(cc)); d != "/t" {
		t.Fatalf("got %v", d)
	}
}

func TestWrap_ReqValue(t *testing.T) {
	biz := func(req testReq) (string, *base.BizError) { return req.Name, nil }
	c := newUserCtx("/t", `{"name":"v"}`, true)
	if err := base.Wrap(biz)(c); err != nil {
		t.Fatal(err)
	}
	if d := decodeData(t, respBody(c)); d != "v" {
		t.Fatalf("got %v", d)
	}
}

func TestWrap_BizError(t *testing.T) {
	biz := func(ctx context.Context) (string, *base.BizError) {
		return "", base.ErrBadReq("bad")
	}
	c := newUserCtx("/t", "", true)
	_ = base.Wrap(biz)(c)
	if c.Response().Status != base.CodeBadRequest.HTTPStatus() {
		t.Fatalf("status=%d", c.Response().Status)
	}
}

func TestWrap_BadSignaturePanics(t *testing.T) {
	mustPanic := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s: expected panic", name)
			}
		}()
		f()
	}
	mustPanic("not a func", func() { base.Wrap(123) })
	mustPanic("one return", func() { base.Wrap(func(req *testReq) string { return "" }) })
	mustPanic("wrong err type", func() { base.Wrap(func() (string, error) { return "", nil }) })
	mustPanic("unsupported param", func() { base.Wrap(func(n int) (string, *base.BizError) { return "", nil }) })
	mustPanic("two req params", func() {
		base.Wrap(func(a *testReq, b *testReq) (string, *base.BizError) { return "", nil })
	})
}

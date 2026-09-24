// 门禁辅助函数的单元测试（行为测试，不是门禁本身）。
//
// 门禁靠这些解析函数"看"仓库：解析错了门禁不会报错，只会**静默失效**——该拦的没拦，
// 比没有门禁更危险。所以每个解析函数都要钉住它认的与不认的写法。
// 对照 DSH 的规则：gate scripts "test every admitted/excluded form that changes their detection boundary"。
//
// 门禁本体在 gate_arch_test.go / gate_docs_test.go / store/gate_schema_test.go。

package main

import (
	"strings"
	"testing"
)

func TestParseRouterRoutes(t *testing.T) {
	const src = `package router

func Register(g *echo.Group) {
	g.POST("/login", handler.Login)
	g.POST("/login/2fa", handler.Login2FA) // 登录第二步：g.POST("/fake", nil) 这种行尾注释里的调用不算
	// g.POST("/commented-out", handler.X)
	/* g.POST("/block-commented", handler.X) */
	g.GET("/legacy", handler.Legacy)
	g.POST(
		"/multiline",
		base.Wrap(handler.Multiline),
	)
	g.POST(pathVariable, handler.X)
	g.Use(middleware.Auth)
	g.POST("/last", base.Wrap(handler.Last))
}
`
	got, err := registeredRoutes(src)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	want := []string{"/manager/login", "/manager/login/2fa", "/manager/legacy", "/manager/multiline", "/manager/last"}
	for _, path := range want {
		if !got[path] {
			t.Errorf("应解析出 %s，实际集合 %v", path, got)
		}
	}
	// 注释里的注册不算：字符串匹配实现会在这里误报
	for _, path := range []string{"/manager/fake", "/manager/commented-out", "/manager/block-commented"} {
		if got[path] {
			t.Errorf("注释里的调用不应被当成路由：%s", path)
		}
	}
	// 非字符串首参、非路由方法都不算
	if len(got) != len(want) {
		t.Errorf("解析出 %d 条路由，期望 %d 条：%v", len(got), len(want), got)
	}
}

func TestParseRouterRoutesFailsLoudly(t *testing.T) {
	// 空文件与语法错误都返回 error：调用方（门禁）遇到就 Fatal，不会静默放过。
	for _, src := range []string{"", "package router\nfunc broken("} {
		if _, err := registeredRoutes(src); err == nil {
			t.Errorf("应返回错误而不是空集合: %q", src)
		}
	}
	// 合法但没有路由注册 → 空集合，由门禁自身的"解析结果为空即规则失效"守卫拦截
	got, err := registeredRoutes("package router\n")
	if err != nil {
		t.Fatalf("合法源码不应报错: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("没有注册路由时应返回空集合，实际 %v", got)
	}
}

func TestParseSeedPermissions(t *testing.T) {
	const src = `
INSERT OR IGNORE INTO role_permission (role_id, entity, action, value) VALUES
  (1, 'API', '*', '*');
INSERT OR IGNORE INTO role_permission (role_id, entity, action, value) VALUES
  (2, 'API', '*', '/manager/self'),
  (2, 'API', '*', '/manager/apikeys/list/self'),
  (2, 'API', 'READ', 'not-a-path'),
  (3, 'API', '*', '/manager/other-role');
`
	got := seededPermissions(src)
	if !got["/manager/self"] || !got["/manager/apikeys/list/self"] {
		t.Fatalf("应解析出 user 角色的路径授权，实际 %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("只应收集 user 角色的 /manager 路径，实际 %v", got)
	}
	if len(seededPermissions("")) != 0 {
		t.Fatal("空源码应返回空集合（门禁依赖该行为判断规则失效）")
	}
}

func TestParseDocumentedRoutes(t *testing.T) {
	const src = "| `POST /manager/login` | 登录 |\n" +
		"| `POST /manager/self` | 当前用户 |\n" +
		"| `/manager/not-a-row` | 没有方法前缀，不算 |\n" +
		"| GET /manager/no-backtick | 没有行内代码格式，不算 |\n"
	got := documentedRoutes(src)
	if !got["/manager/login"] || !got["/manager/self"] {
		t.Fatalf("应解析出接口表里的路径，实际 %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("只应收集 `POST /manager/xxx` 形式，实际 %v", got)
	}
	if len(documentedRoutes("")) != 0 {
		t.Fatal("空源码应返回空集合")
	}
}

func TestParseFrontmatterBlock(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
		ok   bool
	}{
		{name: "正常", src: "---\nname: x\n---\n\n# 标题\n", want: "name: x", ok: true},
		{name: "无 frontmatter", src: "# 标题\n", ok: false},
		{name: "未闭合", src: "---\nname: x\n", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := frontmatterBlock(tc.src)
			if ok != tc.ok {
				t.Fatalf("ok = %t，期望 %t", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("block = %q，期望 %q", got, tc.want)
			}
		})
	}
}

func TestMarkdownSlug(t *testing.T) {
	cases := map[string]string{
		"调用代理 API":        "调用代理-api",
		"Provider 配置":     "provider-配置",
		"环境变量":            "环境变量",
		"`base.Wrap` 的用法": "basewrap-的用法",
		"已知限制（补充）":        "已知限制补充",
	}
	for heading, want := range cases {
		if got := slug(heading); got != want {
			t.Errorf("slug(%q) = %q，期望 %q", heading, got, want)
		}
	}
}

func TestStripCode(t *testing.T) {
	src := "正文 [链接](a.md)\n\n```\n[示例](should-be-ignored.md)\n```\n\n行内 `[x](inline.md)` 结束\n"
	got := stripCode(src)
	if strings.Contains(got, "should-be-ignored.md") {
		t.Error("围栏代码块里的链接应被剥离")
	}
	if strings.Contains(got, "inline.md") {
		t.Error("行内代码里的链接应被剥离")
	}
	if !strings.Contains(got, "a.md") {
		t.Error("正文里的链接不应被剥离")
	}
}

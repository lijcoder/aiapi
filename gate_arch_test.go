// 门禁文件（不是行为测试）：断言**仓库自身**的结构与一致性，不验证生产代码的行为。
//
//   - 依赖方向：各层的 import 约束（store/parser/service/handler）
//   - 响应写入：proxy handler 不得直接写响应
//   - 权限种子：/self 路由与 sql/init-data.sql 双向一致
//   - 接口文档：docs/api.md 的接口表与 manager/router 双向一致
//   - 预提交门禁：.githooks/pre-commit 存在、可执行且仍调用 make check
//
// 运行方式：`make gate`（只跑门禁，等价 go test -run '^TestGate' ./...）。
// 它同时留在 `go test ./...` 里——本仓库没有 CI，默认测试（谁跑都全跑）加上
// `make hooks` 启用的 pre-commit 钩子（默认提交路径自动跑）是门禁不会被绕过的两道保证。
//
// 约定：门禁文件以 gate_ 开头、测试函数以 TestGate 开头；其余 *_test.go 是行为测试。

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// 架构门禁：把 AGENTS.md 的红线里**可以机械判定**的部分变成断言。
// 这些规则原先只写在提示词里靠自觉执行，现在跑 `go test ./...` 就会被拦住。
// 规则来源：AGENTS.md RED-03（handler 不写响应）、RED-06（依赖方向单向）。

// bannedImport 描述某目录下禁止出现的 import 前缀。
type bannedImport struct {
	dir        string   // 相对仓库根的目录
	prefixes   []string // 禁止 import 的包路径前缀
	reason     string   // 违反的红线，失败信息里带上
	allowFiles []string // 允许豁免的文件名（如确需例外）
}

var bannedImports = []bannedImport{
	{
		dir:      "store",
		prefixes: []string{"github.com/labstack/echo", "github.com/lijcoder/aiapi/manager", "github.com/lijcoder/aiapi/parser", "github.com/lijcoder/aiapi/proxy", "github.com/lijcoder/aiapi/service"},
		reason:   "store 是纯 SQL 包装层（AGENTS.md RED-06）：不得依赖框架、协议层与业务层",
	},
	{
		dir:      "parser",
		prefixes: []string{"github.com/labstack/echo", "github.com/lijcoder/aiapi/manager", "github.com/lijcoder/aiapi/proxy", "github.com/lijcoder/aiapi/service", "github.com/lijcoder/aiapi/store"},
		reason:   "parser 只做协议解析（AGENTS.md RED-06）：不得操作数据库、写响应或依赖业务层",
	},
	{
		dir:      "service",
		prefixes: []string{"github.com/labstack/echo", "github.com/lijcoder/aiapi/proxy"},
		reason:   "service 是 manager 与 proxy 共用的业务层（AGENTS.md RED-06）：不得依赖 echo 或 proxy",
	},
	{
		dir:      "proxy/handler",
		prefixes: []string{"github.com/labstack/echo"},
		reason:   "proxy handler 不得依赖 echo（AGENTS.md RED-03）：参数来自 types.Context，错误经 ctx.Err 上报",
	},
	{
		dir:        "manager/handler",
		prefixes:   []string{"github.com/labstack/echo"},
		reason:     "manager handler 只做 HTTP 适配，框架细节由 base.Wrap 承担；仅登录相关接口需要直接读写 cookie",
		allowFiles: []string{"login.go"},
	},
}

// TestGatePackageDependencies 断言各层的 import 方向。
func TestGatePackageDependencies(t *testing.T) {
	for _, rule := range bannedImports {
		rule := rule
		t.Run(rule.dir, func(t *testing.T) {
			files := goFilesIn(t, rule.dir)
			if len(files) == 0 {
				t.Fatalf("目录 %s 下没有 Go 文件，规则失效（目录被改名或删除？）", rule.dir)
			}
			for _, file := range files {
				name := filepath.Base(file)
				if contains(rule.allowFiles, name) {
					continue
				}
				for _, imp := range importPaths(t, file) {
					for _, banned := range rule.prefixes {
						if imp == banned || strings.HasPrefix(imp, banned+"/") {
							t.Errorf("%s 不应 import %s\n  %s", file, imp, rule.reason)
						}
					}
				}
			}
		})
	}
}

// TestGateProxyHandlerDoesNotWriteResponse 断言 proxy handler 不直接写响应。
// 失败时只允许设置 ctx.Err / ctx.Code，由 Pipeline 统一输出（AGENTS.md RED-03）。
func TestGateProxyHandlerDoesNotWriteResponse(t *testing.T) {
	for _, file := range goFilesIn(t, "proxy/handler") {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", file, err)
		}
		for _, forbidden := range []string{"c.JSON(", "c.String("} {
			if strings.Contains(string(src), forbidden) {
				t.Errorf("%s 出现 %s：proxy handler 不得直接写响应（AGENTS.md RED-03）", file, forbidden)
			}
		}
	}
}

// TestGateRoutesHavePermissionSeed 断言后台路由与权限种子一致：
//   - 所有 /self 路由必须在 sql/init-data.sql 里授予 user 角色，否则普通用户调用直接 403
//   - 种子里出现的路径必须是已注册路由，避免改名后留下死权限
//
// 覆盖范围说明：非 /self 但面向普通用户的路由（如 /manager/models）不在本规则的推导范围内，
// 新增这类路由时需人工在 init-data.sql 授权，见 manager/AGENTS.md 的「新增接口收尾清单」。
func TestGateRoutesHavePermissionSeed(t *testing.T) {
	routerSrc, err := os.ReadFile("manager/router/router.go")
	if err != nil {
		t.Fatalf("读取 router 失败: %v", err)
	}
	seedSrc, err := os.ReadFile("sql/init-data.sql")
	if err != nil {
		t.Fatalf("读取权限种子失败: %v", err)
	}

	routes, err := registeredRoutes(string(routerSrc))
	if err != nil {
		t.Fatalf("解析 router 失败: %v", err)
	}
	seeded := seededPermissions(string(seedSrc))
	if len(routes) == 0 || len(seeded) == 0 {
		t.Fatalf("解析结果为空（routes=%d seeded=%d），规则失效", len(routes), len(seeded))
	}

	for route := range routes {
		if !strings.HasSuffix(route, "/self") {
			continue
		}
		if !seeded[route] {
			t.Errorf("%s 是自助接口，但 sql/init-data.sql 未给 user 角色授权（普通用户会 403）", route)
		}
	}
	for path := range seeded {
		if !routes[path] {
			t.Errorf("sql/init-data.sql 授权了 %s，但 manager/router 里没有这个路由（改名后遗留的死权限？）", path)
		}
	}
}

// docRoutePattern 匹配 docs/api.md 接口表里的行内代码写法：`POST /manager/xxx`。
var docRoutePattern = regexp.MustCompile("`POST (/manager/[^`]+)`")

// httpVerbNames 是路由注册用的方法名（echo 风格）。
var httpVerbNames = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true, "Any": true,
}

// registeredRoutes 解析 router 源码里注册的路由，返回带 /manager 前缀的路径集合。
//
// 用 AST 而不是字符串匹配：注释里的调用、换行、空格都不影响结果——字符串匹配会把
// "注释掉的一行"当成真实路由，门禁于是静默失效。解析辅助函数本身有单测（repoparse_test.go）。
func registeredRoutes(src string) (map[string]bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "router.go", src, 0)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !httpVerbNames[sel.Sel.Name] || len(call.Args) == 0 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		path, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		out["/manager"+path] = true
		return true
	})
	return out, nil
}

// documentedRoutes 解析 docs/api.md 接口表里的路径（`POST /manager/xxx` 形式的行内代码）。
func documentedRoutes(src string) map[string]bool {
	out := map[string]bool{}
	for _, m := range docRoutePattern.FindAllStringSubmatch(src, -1) {
		out[m[1]] = true
	}
	return out
}

// TestGateAPIDocCoversAllRoutes 断言 docs/api.md 的接口表与 manager/router 的路由集合**双向一致**：
// 漏写的新接口、改名后没跟着改的旧路径都会在这里失败。
//
// docs/api.md 是目录型文档（长度随接口数增长），所以它不设字节预算，靠这条一致性门禁兜底——
// 目录型文档真正的风险是"腐化"，不是"啰嗦"。
func TestGateAPIDocCoversAllRoutes(t *testing.T) {
	routerSrc, err := os.ReadFile("manager/router/router.go")
	if err != nil {
		t.Fatalf("读取 router 失败: %v", err)
	}
	docSrc, err := os.ReadFile("docs/api.md")
	if err != nil {
		t.Fatalf("读取 docs/api.md 失败: %v", err)
	}

	routes, err := registeredRoutes(string(routerSrc))
	if err != nil {
		t.Fatalf("解析 router 失败: %v", err)
	}
	documented := documentedRoutes(string(docSrc))
	if len(routes) == 0 || len(documented) == 0 {
		t.Fatalf("解析结果为空（routes=%d documented=%d），门禁失效", len(routes), len(documented))
	}

	for route := range routes {
		if !documented[route] {
			t.Errorf("路由 %s 没有写进 docs/api.md 的接口表", route)
		}
	}
	for route := range documented {
		if !routes[route] {
			t.Errorf("docs/api.md 记了 %s，但 manager/router 里没有这个路由（改名后没同步？）", route)
		}
	}
}

// seededPermissions 从 init-data.sql 里提取 user 角色（role_id=2）的接口路径授权。
func seededPermissions(src string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "(2, 'API'") {
			continue
		}
		start := strings.LastIndex(line, "'")
		if start <= 0 {
			continue
		}
		value := line[:start]
		value = value[strings.LastIndex(value, "'")+1:]
		if strings.HasPrefix(value, "/manager") {
			out[value] = true
		}
	}
	return out
}

// TestGateGitHooksIsWired 断言预提交门禁仍挂在默认提交路径上（AGENTS.md RED-01）。
//
// 仓库没有 CI，`go test` 里的门禁只有"有人跑"才生效；`.githooks/pre-commit` +
// `core.hooksPath` 是唯一不依赖自觉的触发点。本门禁防止它被删除、被去掉可执行位
// （git 会静默跳过）、被改成空操作，或安装路径与钩子目录脱钩：
//   - `.githooks/pre-commit` 存在、可执行、仍调用 `make check`
//   - `Makefile` 的 hooks 目标把 core.hooksPath 指向 `.githooks`
//
// 覆盖边界：本门禁只看仓库内的接线，**不看本机是否执行过 `make hooks`**——core.hooksPath
// 存在本地 .git/config（不入库），判它会让全新 clone 与将来的 CI 误报。本地未启用时
// `make check` 末尾会提示。
func TestGateGitHooksIsWired(t *testing.T) {
	const hookPath = ".githooks/pre-commit"

	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("%s 不存在：预提交门禁被删除或改名（AGENTS.md RED-01）", hookPath)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("%s 没有可执行位（mode %v）：git 会静默跳过它，门禁等于没装", hookPath, info.Mode().Perm())
	}
	src, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", hookPath, err)
	}
	if !strings.Contains(string(src), "make check") {
		t.Errorf("%s 不再调用 make check（只留注释也算）：钩子被架空，提交时什么都不会拦", hookPath)
	}

	mk, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("读取 Makefile 失败: %v", err)
	}
	if !strings.Contains(string(mk), "core.hooksPath .githooks") {
		t.Errorf("Makefile 的 hooks 目标必须执行 `git config core.hooksPath .githooks`；路径与钩子目录脱钩时，改了钩子也不会生效")
	}
}

// goFilesIn 返回目录下（含子目录）的所有 .go 文件。
func goFilesIn(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s 失败: %v", dir, err)
	}
	return files
}

// importPaths 用 go/parser 提取一个文件的 import 路径。
func importPaths(t *testing.T, file string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", file, err)
	}
	var paths []string
	for _, imp := range f.Imports {
		paths = append(paths, strings.Trim(imp.Path.Value, `"`))
	}
	return paths
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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

// TestPackageDependencies 断言各层的 import 方向。
func TestPackageDependencies(t *testing.T) {
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

// TestProxyHandlerDoesNotWriteResponse 断言 proxy handler 不直接写响应。
// 失败时只允许设置 ctx.Err / ctx.Code，由 Pipeline 统一输出（AGENTS.md RED-03）。
func TestProxyHandlerDoesNotWriteResponse(t *testing.T) {
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

// TestRoutesHavePermissionSeed 断言后台路由与权限种子一致：
//   - 所有 /self 路由必须在 sql/init-data.sql 里授予 user 角色，否则普通用户调用直接 403
//   - 种子里出现的路径必须是已注册路由，避免改名后留下死权限
//
// 覆盖范围说明：非 /self 但面向普通用户的路由（如 /manager/models）不在本规则的推导范围内，
// 新增这类路由时需人工在 init-data.sql 授权，见 docs/howto 与 manager/AGENTS.md 的收尾清单。
func TestRoutesHavePermissionSeed(t *testing.T) {
	routerSrc, err := os.ReadFile("manager/router/router.go")
	if err != nil {
		t.Fatalf("读取 router 失败: %v", err)
	}
	seedSrc, err := os.ReadFile("sql/init-data.sql")
	if err != nil {
		t.Fatalf("读取权限种子失败: %v", err)
	}

	routes := registeredRoutes(string(routerSrc))
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

// registeredRoutes 从 router 源码里提取 g.POST("...") 的路径，返回带 /manager 前缀的集合。
func registeredRoutes(src string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		idx := strings.Index(line, `g.POST("`)
		if idx < 0 {
			continue
		}
		rest := line[idx+len(`g.POST("`):]
		end := strings.Index(rest, `"`)
		if end < 0 {
			continue
		}
		out["/manager"+rest[:end]] = true
	}
	return out
}

// docRoutePattern 匹配 docs/api.md 接口表里的行内代码写法：`POST /manager/xxx`。
var docRoutePattern = regexp.MustCompile("`POST (/manager/[^`]+)`")

// TestAPIDocCoversAllRoutes 断言 docs/api.md 的接口表与 manager/router 的路由集合**双向一致**：
// 漏写的新接口、改名后没跟着改的旧路径都会在这里失败。
//
// docs/api.md 是目录型文档（长度随接口数增长），所以它不设字节预算，靠这条一致性门禁兜底——
// 目录型文档真正的风险是"腐化"，不是"啰嗦"。
func TestAPIDocCoversAllRoutes(t *testing.T) {
	routerSrc, err := os.ReadFile("manager/router/router.go")
	if err != nil {
		t.Fatalf("读取 router 失败: %v", err)
	}
	docSrc, err := os.ReadFile("docs/api.md")
	if err != nil {
		t.Fatalf("读取 docs/api.md 失败: %v", err)
	}

	routes := registeredRoutes(string(routerSrc))
	documented := map[string]bool{}
	for _, m := range docRoutePattern.FindAllStringSubmatch(string(docSrc), -1) {
		documented[m[1]] = true
	}
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

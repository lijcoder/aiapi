// 门禁文件（不是行为测试）：断言**仓库自身**的文档规范与一致性，不验证生产代码的行为。
//
//   - 链接与锚点：所有相对链接可达
//   - 体积预算：常驻/规则类文档不超预算（目录型文档不设预算，改由一致性门禁约束）
//   - skill frontmatter：可被 YAML 解析且具备 name / description
//
// 运行方式：`make gate`（只跑门禁，等价 go test -run '^TestGate' ./...）。
// 它同时留在 `go test ./...` 里——本仓库没有 CI，放进默认测试是让门禁不会被绕过的唯一保证。
//
// 约定：门禁文件以 gate_ 开头、测试函数以 TestGate 开头；其余 *_test.go 是行为测试。

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"
)

// 文档门禁：把"文档别腐化"从口头约定变成断言。
//   - 所有相对链接（含锚点）必须可达：改名/删文件后忘记改链接会在这里失败
//   - 常驻文档有字节上限：防止又一次膨胀回 500 行的规则文件（参考 DSH 的 doc-budgets 门禁）

// docBudgets 是**常驻/规则/流程类**文档的体积上限（字节）。
//
// 只给这一类设预算的原因：它们膨胀意味着"写啰嗦了"，超限的正确反应是把细节下沉。
// 而**目录型参考文档**（如 docs/api.md：一行一个接口，长度随接口数线性增长）不设预算——
// 给它设上限等于给接口数量设上限。这类文档改用"与源头一致性"门禁：
//   - docs/api.md ↔ manager/router/router.go 的路由集合（gate_arch_test.go 的 TestGateAPIDocCoversAllRoutes）
var docBudgets = map[string]int{
	"AGENTS.md":                9500,
	"README.md":                17000,
	"TODO.md":                  7000,
	"docs/AGENTS.md":           6000,
	"docs/architecture.md":     21000,
	"docs/pricing.md":          5000,
	"docs/deployment.md":       6500,
	"docs/security.md":         5500,
	"docs/glossary.md":         5000,
	"frontend/AGENTS.md":       5000,
	"parser/AGENTS.md":         4500,
	"proxy/AGENTS.md":          4500,
	"manager/AGENTS.md":        6000,
	"store/AGENTS.md":          4000,
	"sql/AGENTS.md":            4000,
	"sql/migrations/README.md": 9000,
}

// markdownLink 匹配 [文本](目标)；目标不含空白与右括号。
var markdownLink = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)
var fencedBlock = regexp.MustCompile("(?s)```.*?```")
var inlineCode = regexp.MustCompile("`[^`\n]*`")
var headingLine = regexp.MustCompile(`(?m)^#{1,6}\s+(.*)$`)

// TestGateSkillFrontmatterParses 校验 .agents/skills/*/SKILL.md 的 YAML frontmatter：
// 必须具备 name / description，且值不能使用会让 YAML 解析失败的裸标量写法。
//
// 起因：description 里写了 "… AGENTS.md owns: gates, schema version …" 这样含 ": " 的裸值，
// YAML 会把它解析成嵌套 mapping（"Nested mappings are not allowed in compact mappings"），
// 整个 skill 加载失败，而仓库里没有任何检查会发现——只有 GUI 上弹一行报错。
// 仓库不引 YAML 依赖，这里按 YAML 对 plain scalar 的规则做最小校验（值含 ": " 必须加引号）。
func TestGateSkillFrontmatterParses(t *testing.T) {
	files, err := filepath.Glob(".agents/skills/*/SKILL.md")
	if err != nil {
		t.Fatalf("查找 skill 失败: %v", err)
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", file, err)
		}
		block, ok := frontmatterBlock(string(src))
		if !ok {
			t.Errorf("%s: 缺少 --- 包裹的 frontmatter", file)
			continue
		}
		seen := map[string]bool{}
		for i, line := range strings.Split(block, "\n") {
			// 缩进行属于块标量或嵌套结构，不按顶层 key 校验
			if line == "" || strings.HasPrefix(line, "#") || line != strings.TrimLeft(line, " \t") {
				continue
			}
			key, value, found := strings.Cut(line, ":")
			if !found {
				t.Errorf("%s: frontmatter 第 %d 行不是 `key: value` 形式：%q", file, i+2, line)
				continue
			}
			value = strings.TrimSpace(value)
			if value == "" {
				t.Errorf("%s: frontmatter 的 %s 没有值", file, key)
				continue
			}
			seen[strings.TrimSpace(key)] = true
			if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, "'") {
				continue
			}
			if strings.Contains(value, ": ") {
				t.Errorf("%s: frontmatter 的 %s 是未加引号的裸值且含 \": \"，YAML 会把它当嵌套 mapping 而解析失败——请用双引号把整个值包起来", file, key)
			}
		}
		for _, required := range []string{"name", "description"} {
			if !seen[required] {
				t.Errorf("%s: frontmatter 缺少 %s（skill 发现与触发都依赖它）", file, required)
			}
		}
	}
}

// frontmatterBlock 取出 --- 包裹的 YAML frontmatter。
func frontmatterBlock(src string) (string, bool) {
	lines := strings.Split(src, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), true
		}
	}
	return "", false
}

// TestGateMarkdownLinksResolve 校验文档里的相对链接与锚点。
func TestGateMarkdownLinksResolve(t *testing.T) {
	files := markdownFiles(t)
	if len(files) < 5 {
		t.Fatalf("只找到 %d 个 markdown 文件，规则失效", len(files))
	}
	anchors := map[string]map[string]bool{}
	for _, f := range files {
		anchors[f] = headingAnchors(t, f)
	}

	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", file, err)
		}
		body := stripCode(string(src))
		for _, m := range markdownLink.FindAllStringSubmatch(body, -1) {
			target := m[1]
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") ||
				strings.HasPrefix(target, "mailto:") {
				continue
			}
			pathPart, anchor := target, ""
			if i := strings.Index(target, "#"); i >= 0 {
				pathPart, anchor = target[:i], target[i+1:]
			}
			resolved := file
			if pathPart != "" {
				resolved = filepath.Join(filepath.Dir(file), pathPart)
				if _, err := os.Stat(resolved); err != nil {
					t.Errorf("%s: 链接目标不存在 [%s](%s)", file, m[0], target)
					continue
				}
			}
			if anchor == "" {
				continue
			}
			set, ok := anchors[resolved]
			if !ok {
				// 非 markdown 目标（如图片/附件）不做锚点校验
				if strings.HasSuffix(resolved, ".md") {
					t.Errorf("%s: 锚点目标 %s 未被扫描", file, resolved)
				}
				continue
			}
			if !set[anchor] {
				t.Errorf("%s: 锚点不存在 [%s](%s)", file, m[0], target)
			}
		}
	}
}

// TestGateDocBudgets 校验常驻文档不超过预算。
func TestGateDocBudgets(t *testing.T) {
	for path, budget := range docBudgets {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("%s: 预算内的文档不存在（改名或删除后请同步 docBudgets）", path)
			continue
		}
		if info.Size() > int64(budget) {
			t.Errorf("%s: %d 字节超过预算 %d —— 把细节拆到按需文档，或说明理由后上调预算", path, info.Size(), budget)
		}
	}
}

// markdownFiles 返回仓库内（排除依赖与构建产物）的 markdown 文件。
func markdownFiles(t *testing.T) []string {
	t.Helper()
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "dist": true, "temp": true,
		".vscode": true, "testdata": true,
	}
	var files []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".md") && !strings.HasPrefix(path, "frontend/dist/") {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("遍历仓库失败: %v", err)
	}
	return files
}

// headingAnchors 提取一个 markdown 文件所有标题的锚点（GitHub 风格 slug）。
func headingAnchors(t *testing.T, file string) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", file, err)
	}
	set := map[string]bool{}
	for _, m := range headingLine.FindAllStringSubmatch(stripCode(string(src)), -1) {
		set[slug(m[1])] = true
	}
	return set
}

// slug 复刻 GitHub 标题锚点规则：小写、去掉标点、空格转连字符，保留中英文与数字。
func slug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// stripCode 去掉围栏代码块与行内代码，避免把示例里的 [x](y) 当成真链接。
func stripCode(src string) string {
	src = fencedBlock.ReplaceAllString(src, "")
	return inlineCode.ReplaceAllString(src, "")
}

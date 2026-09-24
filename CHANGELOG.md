# CHANGELOG

> 记录项目的版本变更、功能新增、破坏性变更以及对用户/开发者有影响的改动。
> 按时间倒序排列，当前尚未发布版本的内容放在 `[Unreleased]` 下。

## [Unreleased]

- **移除用户自助充值**：普通用户不能再给自己加余额，充值只能由超管在「用户管理 → 充值」发起。删除自助接口 `POST /manager/recharge/self`（含 `RechargeSelf` handler 与路由注册）、`sql/init-data.sql` 里 user 角色对该路径的授权，以及前端「充值中心」页的充值按钮、弹窗与 `rechargeSelf` API 封装；该页改为只读页（余额 + 自己的充值流水），菜单名与页面标题由「充值中心」改为「充值记录」，只读流水接口 `POST /manager/recharge/records/self` 保持不变。**存量库需手工执行一次**（种子是 `INSERT OR IGNORE`，改名与删除不会自动生效）：`UPDATE menus SET name='充值记录' WHERE id=4;` 与 `DELETE FROM role_permission WHERE role_id=2 AND entity='API' AND value='/manager/recharge/self';`。

- **新增预提交门禁（git hook）**：门禁此前只在手动跑 `make check` 时生效，本批把它挂到默认提交路径——新增 `.githooks/pre-commit`（提交前跑 gofmt 检查 + `go vet` + 全量测试，失败即中止提交并打印修复/跳过提示）与 `make hooks`（执行 `git config core.hooksPath .githooks`，幂等，`make install` 已依赖它）。`core.hooksPath` 存在本地 `.git/config`、不入库，**每个 clone 需各执行一次 `make hooks`**；未启用时 `make check` 收尾会打印一行提示。唯一跳过方式是 `git commit --no-verify`（等于本次放弃门禁）。新增门禁 `TestGateGitHooksIsWired` 断言钩子存在、有可执行位、仍调用 `make check`，且 `Makefile` 的 hooks 目标指向同一目录——防的是三类静默失效：钩子被删、被 `chmod -x`（git 会跳过无可执行位的钩子）、被改成空操作或安装路径与钩子目录脱钩。既有触发方式不变：门禁照旧留在 `go test ./...` 里，`make gate` 仍是快路径。

- **门禁文件改名，区分门禁与行为测试**：门禁文件 `arch_test.go` → `gate_arch_test.go`、`docs_test.go` → `gate_docs_test.go`、`store/schema_version_test.go` → `store/gate_schema_test.go`，门禁函数统一加 `TestGate` 前缀（`TestRoutesHavePermissionSeed` → `TestGateRoutesHavePermissionSeed`、`TestAPIDocCoversAllRoutes` → `TestGateAPIDocCoversAllRoutes`）；新增 `make gate` 快路径（`go test -run '^TestGate' ./...`，`make check` 仍跑全量、不重复跑）与门禁解析辅助函数的自测 `repoparse_test.go`（钉住每个解析函数"认哪些写法、不认哪些写法"，并验证解析为空会触发门禁自身的失效守卫）；`registeredRoutes` 由字符串匹配改为 `go/ast` 语法感知提取，修掉"注释里的注册被当成真实路由"的缺陷。**补记（同批修复）**：本次改名后仓库内有 12 个文件、21 行文档仍引用旧文件名（`arch_test.go`、`docs_test.go`、`store/schema_version_test.go`）与旧函数名，而 `gate_docs_test.go` 只校验 markdown 链接、行内代码形式的文件名不在校验范围内，漂移未被门禁发现；已随本批把全部引用改为当前文件名，并修正 `docs/architecture.md` 中"仓库暂无自动化拦截"与门禁表自相矛盾的表述。

- **新增数据库 schema 版本与迁移台账**：`constant.SchemaVersion` 成为 schema 版本的唯一权威，`sql/sqlite.sql` 只描述当前完整 schema 并在开头建 `schema_meta` 单行元数据表写入版本行，历史迁移步骤移入 `sql/migrations/README.md`（含版本表、逐项增量 SQL、升级流程、多数据库写法）。**版本记录用普通元数据表而非 SQLite 专有的 `PRAGMA user_version`**，因为本项目要兼容 MySQL / PostgreSQL（`user_version` 只有 SQLite 有）；Go 侧读取统一是 `SELECT version FROM schema_meta WHERE id = 1`，不按驱动分支。应用启动时读该版本与常量比对：**库比二进制旧会拒绝启动**并提示要执行哪些迁移；**库比二进制新同样拒绝启动**（避免降级丢列丢数据）；老库没有该表（或表里没有版本行）会告警但允许启动，提示核对台账后补标记——用一次可移植的 `SELECT 1` 探活区分「表不存在」与「连接故障」，后者仍然硬失败。存量库升级到本版本需手工建表并写入版本行（台账与 README 有可复制语句）。`store/schema_version_test.go` 机械校验 DDL 写入的版本与常量一致，并校验整份 DDL 可重复执行。决策理由见 `docs/decisions/2026-09-24-schema-version-and-migration-ledger.md`。

- **修复普通用户权限种子缺失**：`user` 角色缺少 `/manager/models`（「模型列表」页在用，菜单已授权但接口 403）与 `/manager/apikeys/reveal/self`（API Key 页「查看明文」按钮在用）两条授权，普通用户调用均被拒。已补齐种子；存量库重跑 `sql/init-data.sql`（`INSERT OR IGNORE`，幂等）即可生效。复盘见 `docs/postmortem/2026-09-24-user-role-permission-seed-gap.md`。

- **新增门禁（`make check`）**：把原先只写在提示词里靠自觉执行的部分规则变成 `go test` 里的断言，新增 `make check`（gofmt 检查 + `go vet` + `go test`）。① 架构依赖：`store` 不得依赖 echo/manager/parser/service，`parser` 不得依赖 store 与业务层，`service` 不得依赖 echo/proxy，`manager/handler` 仅 `login.go` 允许使用 echo，proxy handler 不得出现 `c.JSON`/`c.String`；② 权限种子：所有 `/self` 路由必须在 `sql/init-data.sql` 给 user 角色授权，且种子里不得有已删除路由的死权限（非 `/self` 的普通用户路由推导不出来，仍在文档中标注为人工判断项）；③ 文档：相对链接与锚点必须可达、常驻文档不得超字节预算；④ schema 版本一致性。清理了 `proxy/handler/auth_key.go` 一处历史遗留的格式问题，使 `gofmt` 检查从首次运行起即为绿。

- **新增协议解析 golden file 测试**：三个协议各一套（`parser/golden_{openai,anthropic,responses}_test.go`）与 30 个真实形状的 fixture（`parser/testdata/`），覆盖 `ParseModel` / `ReplaceModel`（含"无需改写时字节零变动"）/ `ParseApiKey`（各协议鉴权头优先级）/ 非流式与流式用量全字段 / `FormatModelList`，以及非法 JSON、usage 缺失、无用量流式事件等失败路径。经 6 个人为变异验证断言有效。测试过程中发现三个待修问题（非流式与流式的用量口径不一致、非流式 usage 缺失时被记为 0 用量、SSE 多行 `data:` 未按规范合并），已登记 `TODO.md` 第 6–8 项。

- 文档体系重构为分层结构，并把"写在哪一层"写成规范：`AGENTS.md` 收敛为**常驻规则**（速查命令、目录地图、16 条红线、边界判定式、自检清单、导航），并新增**包级 `AGENTS.md`**（`frontend/`、`parser/`、`proxy/`、`manager/`、`store/`、`sql/`，进入对应目录工作时由 harness 自动加载，开篇声明补充根规则、不复述红线）；跨包内容留在 `docs/`：`architecture.md`（分层、边界、事务、并发、运行时不变量、门禁、schema 版本、已知偏离）、`security.md`、新增 `glossary.md`（领域词）、`decisions/`（决策记录：为什么这么定、否决了什么）、`postmortem/`（事故复盘与护栏）、`AGENTS.md`（文档标准：一个事实一个家、放置判定、体积预算）。`docs/howto/` 的内容迁入对应包后删除；新增跨包流程技能 `.agents/skills/aiapi-release/SKILL.md`（发布/升级检查清单）；新增 `docs_test.go` 对 skill frontmatter 的校验（`description` 等值含 `": "` 时必须加引号，否则 YAML 会当成嵌套 mapping 导致整个 skill 静默加载失败）；`temp/` 下两份陈旧草稿（均为已完成的清单）删除，其中仍有效的优化项登记进 `TODO.md`。修正原文档三处自相矛盾与三处与代码不符的表述：Gemini 并非已支持协议（`GetParser` 返回 `nil`）；"在 `Home.vue` 的 `<nav>` 加 `<router-link>`"已作废（菜单由后端 `menus` 树驱动）；"单表可直接调 store 但仍建议统一走 service"改为可判定的边界表；`store/` 边界由"只做单表"放宽为"单表读写 + 只读 JOIN/聚合"。补入此前未成文的约定：Makefile 命令入口、提交信息规范、Go 版本以 `go.mod` 为准、构建物与 `frontend/dist/index.html` 不可删、proxy 英文/manager 中文的错误消息分层、`base.Wrap` 的启动期 panic 约束、运行时不变量。

- 门禁分类调整：**目录型文档不再设字节预算，改用与源头的一致性校验**。`docs/api.md` 的接口表与 `manager/router/router.go` 的路由集合双向比对（`arch_test.go` 的 `TestAPIDocCoversAllRoutes`）——漏写新接口、改名后没同步旧路径都会失败；两个方向都做了变异验证。理由：`api.md` 一行一个接口，长度随接口数线性增长，给它设字节上限等于给接口数量设上限，超限时只能删有用信息。`docs/AGENTS.md` 的预算章节据此改写为"预算只覆盖会写啰嗦的文档"，`docs/architecture.md` 的门禁表补上这一条。

- 拆分 `README.md`：原文 569 行里管理接口清单（19%）、计费配置（8%）与部署运维细节占了大头，把这些**参考型内容**抽成三份按需查阅的文档，README 只留上手路径（快速开始 → 调用示例 → 配置参考 → 已知限制），569 行 / 29.8 KB 降到 280 行 / 15 KB。新增 `docs/api.md`（`/manager` 全部接口、请求字段、调用示例）、`docs/pricing.md`（`pricing_config` v2 结构、条件、计费快照、各协议用量口径）、`docs/deployment.md`（数据目录、密钥轮换后果、HTTPS 反代、生产检查清单、日志、备份、升级与 schema 版本校验、数据表清单）。内容均为搬迁而非重写，未丢失信息；`docs/AGENTS.md` 的分层表与「放置判定」补入这三个位置，`docs_test.go` 的 `docBudgets` 同步（README 预算由 32 KB 降到 17 KB）。

- 重写 `README.md` 为用户文档，按「功能特性 / 架构概览 / 快速开始 / 调用代理 API / 管理台 / 管理接口一览 / 模型计费配置 / 配置参考 / 数据与存储 / 部署 / 已知限制 / 开发」重排，删除与开发规则重复的分层架构表。修正多处与实际不符的说明：① 数据库路径是 `<数据目录>/db/aiapi.db`，原文写作 `~/.aiapi/aiapi.db`（缺少 `db/` 一级）；② Provider 管理此前示例为 `GET/POST/DELETE /manager/providers`，实际统一为 `POST /manager/providers/{create,update,toggle,list}`；③ Provider 请求体是平铺的 `{type, domain, headers}`，原文写成嵌套的 `{type, config:{domain, headers}, enabled}`；④ Go 版本要求以 `go.mod` 为准（1.25.4，原文写 1.22+）；⑤ 管理台接口与请求字段按 handler 结构体逐项校正，补齐 `/manager/users/get`、`/manager/2fa/{setup,confirm,disable}/self`、`/manager/login/2fa`、`/manager/logs`；⑥ `--address` 是地址前缀、必须带结尾冒号（写 `127.0.0.1` 会拼出 `127.0.0.18888`）；⑦ 登录开启 2FA 时返回 `{need_2fa, pending_ticket, expires_in}`；⑧ 会话落库、仅 TOTP 失败计数在内存。新增章节：命令行参数与环境变量、数据目录结构、Provider 配置字段、备份与升级（含 schema 版本校验）、HTTPS 反向代理（必须转发 `X-Forwarded-Proto`，否则 refresh cookie 不带 `Secure`）、生产部署检查清单、已知限制。同时说明全新 clone 后直接 `make build` 会因 `frontend/dist` 仅入库 `index.html`、缺 `assets/` 而导致管理台白屏，首次构建与部署须用 `make build-all`。

- **安全修复**：转发不再跟随上游 3xx。`upstreamClient` 此前未配置 `CheckRedirect`，使用标准库默认行为（最多跟随 10 次）；而标准库只在**跨域**重定向时剥离 `Authorization` / `Www-Authenticate` / `Cookie` / `Cookie2`，本项目的上游凭证来自 `providers.config.headers`，常见形态是 Anthropic 的 `x-api-key`、Azure 风格的 `api-key`——这些自定义头会被原样转发到重定向目标，等于把上游密钥交给第三方（已用跨域 302 复现）。现改为 `CheckRedirect` 返回 `http.ErrUseLastResponse`：不发起第二次请求，3xx 与响应体原样透传给客户端。回归测试同时断言"重定向目标一次都没被访问"与状态码原样透传（`proxy/handler/forward_redirect_test.go`），并经变异验证：移除该配置后测试立即失败。设计与取舍见 `docs/decisions/2026-09-24-refuse-upstream-redirects.md`。

- **破坏性变更**：反向代理路由的路径参数顺序由 `/proxy/:format/:provider/*` 调整为 `/proxy/:provider/:format/*`（模型列表端点同步为 `GET /proxy/:provider/:format/v1/models`），即 `:provider` 在前、`:format` 在后。仅调整路由注册模式与适配层取参（`proxy/router/router.go`），`format` / `provider` 的语义、鉴权、转发、计费与日志行为完全不变，`request_logs` 的 `format` / `provider` 字段取值也不变。**既有调用方与前置网关/Nginx 的 URL 改写规则需同步调整**：`format` 与 `provider` 同名的调用字面不变（如 `/proxy/openai/openai/v1/chat/completions` 无需改动），不同名的调用需换序（如 Responses 由 `/proxy/openai-responses/openai/v1/responses` 变为 `/proxy/openai/openai-responses/v1/responses`）。同步更新 `README.md`（路由说明、架构图与调用示例）与 `AGENTS.md` §4.3，`framework/echo_test.go` 的路由用例随之调整。

- 内部结构调整：协议解析器共用的纯工具（SSE 行/事件切分、请求头取 API Key、请求体顶层 model 改写）从 `parser` 迁到新包 `parser/util`（`SSEParseData`、`SplitSSEEvents`、`EachSSEData`、`ExtractBearerToken`、`ReplaceTopLevelModel`、`ErrModelNotString`），由各协议解析器引用，该包不反向依赖 `parser`；`parser` 不再导出 `SSEParseData` 与 `ErrModelNotString`。鉴权头名不写死在工具里：`ExtractBearerToken(headers, 头名...)` 由协议按优先级传字段（标准头常量为 `parser.HeaderAuthorization`，如 Anthropic = `x-api-key` → 它），按顺序取第一个非空头，值带 `Bearer ` 前缀则剥掉、裸 token 原样返回。代理行为、用量统计与计费口径不变。

- 修复 Anthropic 协议的 Token 用量解析：`usage` 中的 `cache_creation_input_tokens`（缓存创建）与 `cache_read_input_tokens`（缓存读取）此前被直接丢弃，现按统一用量（OpenAI 语义：`input_tokens` 为完整输入、`cached_tokens` 为其子集）映射——完整输入量 = `input_tokens` + 缓存创建 + 缓存读取，`cached_tokens` 取缓存读取（按缓存命中价计费；缓存创建并入未命中价，统一用量没有独立的缓存写入价），`total_tokens` = 完整输入 + 输出；流式与非流式口径一致，另外 `output_tokens_details.thinking_tokens` 计入 `reasoning_tokens`（output 的子集，不额外计费）。Anthropic 流式事件分工明确：`message_delta` 只取用量（官方 API 在该事件给出本次请求的**累计** `input_tokens` + 缓存创建 + 缓存读取 + `output_tokens` + `output_tokens_details.thinking_tokens`），`message_start` 只取 identity（`model` 与 `request_id`），其中的 `output_tokens`（首个很小的计数）不再参与用量解析——这正是此前流式 `total_tokens` 记小的根因。流式解析同时收敛到各协议自身：新增 `Parser.ParseStreamUsage(body)`，由每个协议自行遍历 SSE 并决定取值与总量口径——Anthropic 在 `message_delta` 取累计用量（`message_start` 只取 `model` / `request_id`）、OpenAI chat 只在带 `usage` 的块取用量与 `id`/`model`（不再跨块合并）、Responses 只在 `response.completed` 取用量与 identity（其 response 对象为完整对象，`id`/`model` 必填）；总量 Anthropic = 完整输入 + 输出，OpenAI / Responses = 优先上游值，缺省回退 输入 + 输出。删除了跨协议的 `StreamEvent` 类型与 `ParseStreamEvent` 方法（事件级解析与内容是纯透传，无消费方）及通用聚合 `parser/stream.go`；`parser/util` 只保留 `EachSSEData`/`SplitSSEEvents`/`SSEParseData` 这类 SSE 框架级工具。由此修复 Anthropic 流式 `total_tokens` 停留在 `message_start` 旧值（`input + 1`）的问题：累计 `output_tokens` 在 `message_delta` 才给出，此前覆盖输出量后未重算总量；而 `total_tokens` 参与分段计费的 `total_tokens` 条件匹配，可能命中错误的计费规则。**计费口径变化**：使用提示缓存的 Anthropic 请求输入量会大于升级前（含缓存部分），缓存读取部分由按未命中价计费改为按缓存命中价计费；用量统计中 Anthropic 的 `cached_tokens` 与 `cache_hit_rate` 从此有值（此前恒为 0）。未使用提示缓存的 Anthropic 请求输入量不变，但其流式 `total_tokens` 同样由旧值修正为「完整输入 + 最终输出」（此前停留在 `message_start` 的 `input + 1`）；其它协议在标准上游事件顺序下数值不变。

- 模型新增「提供商模型名 `provider_model`」，把对用户可见的模型名与发往上游的模型名解耦：`model` 仍是客户端调用、`GET v1/models` 返回、鉴权/白名单/计费/用量与日志使用的名字；`provider_model` 是转发时替换请求体顶层 `model` 的值。保存时未传或传空（含纯空格）表示与 `model` 一致，由 `service.ModelService` 归一化后落库；代理链路新增 `RewriteModel` handler（`LoadConfig` 之后、`Forward` 之前），仅在生效上游模型名不等于请求模型名时改写请求体（`Parser` 接口新增 `ReplaceModel`，三个解析器复用 `parser/model_rewrite.go` 的顶层替换逻辑，无需改写时原样返回入参、字节零变动），失败按服务端错误中断请求（`request_logs` 照常记录）。上游响应体（含 SSE）中的 `model` 不改写，仍是上游返回的真实模型名；`request_logs.request_body` 记录的是实际发往上游的请求体，便于对照排查。管理台「模型管理」新增「上游模型」列与 `provider_model` 输入框（新增/编辑/复制可填，`provider`/`model` 仍不可改），普通用户 `/manager/models` 响应不返回该字段。`models` 表新增列，存量库需手动执行 `ALTER TABLE models ADD COLUMN provider_model TEXT NOT NULL DEFAULT '';` 与 `UPDATE models SET provider_model = model WHERE provider_model = '';`（未回填时代理按 `model` 回退，行为与升级前一致）。

- 管理台模型计费配置改为图形化表单：模型基础 Token 与多模态配置位于上方，分段计费位于底部；时间段使用起止时间控件，日期使用日历选择和标签管理，Token 比较符使用直观按钮组；用户端模型列表新增只读计费详情弹窗，展示默认价格、时区、规则条件和优先级。

- 管理台模型列表将“编辑”收纳到“更多”菜单，原编辑按钮位置改为直接查看计费详情。

- 模型计费详情中将规则价格置于条件之前，条件改为分层展示 AND/OR、条件类型和值；移除弹窗底部重复的关闭按钮。

- 计费详情条件改为“适用条件（全部满足/满足任一条件）”分层展示，满足方式红色标注，子条件使用条件序号和层级序号，条件类型统一使用冒号分隔。

- 计费详情进一步简化条件展示：移除条件编号，每个叶子条件独立为卡片，嵌套条件组仅保留满足方式提示。

- 收紧条件卡片的上下内边距和行高，移除多余最小高度，避免内容下方出现空白行。

- 计费详情规则支持独立展开/折叠：多条规则默认折叠、单条规则默认展开；折叠时保留规则概要和价格，隐藏适用条件。

- 移除规则价格区域中冗余的“命中价格：”提示文字。

- 统一默认价格和规则价格的展示文案为“输入（缓存命中）”“输入（缓存未命中）”“输出”，规则价格改为紧凑的价格项布局。

- 修复管理员编辑、复制模型时支持模态未正确回显的问题。

- 计费快照补充完整模型 `pricing_config`，并将 `request_started_at` 统一序列化为 `yyyy-MM-dd HH:mm:ss`，同时以 `matched_rule_name` 保存本次命中的规则名称和实际价格。

- 模型计费升级为 v2 可组合的分段规则：`when` 支持嵌套 `and`/`or` 条件树，时间拆为 weekday、time_range、month_day 原子条件，Token 拆为 gt/gte/lt/lte 原子条件；规则仅以唯一 name 标识，命中多个规则按 priority 选择，未命中回退默认价格。读取 v1 配置时自动转换为 v2。
- `usage_records` 新增 `pricing_snapshot`，固化每次请求实际命中的规则名称、条件、价格、时区和开始时间，保证后续修改模型配置后仍可审计历史费用。用量记录与余额扣减收敛为同一事务，避免只记账或只扣费。
- 移除模型旧的三项固定单价字段；存量库升级后需在管理台重新配置模型计费，迁移 SQL 见 `sql/sqlite.sql` 注释。未配置或配置不合法的模型不会被转发。

- 统一应用与 Echo/HTTP 框架日志输出位置：共用 stdout 和 `logs/app.log`（含日志轮转），保留 Echo 原有日志格式。

- 代理请求日志与用量记录新增 `first_token_ms`（流式请求首 token 耗时）和 `latency_ms`（端到端耗时）字段，统一以毫秒保存；流式首 token 按首次收到上游响应块计时，非流式请求的 `first_token_ms` 保持 0，请求收尾时写入两张表。存量数据库需按 `sql/sqlite.sql` 注释手动补充列。

- 代理错误日志优化：统一按服务端错误记录 HTTP 500；`request_logs.error` 与应用日志同时保存错误类型及脱敏后的底层错误详情，便于直接在管理台定位网络异常原因。

- SSE 透传调整为首个读取块直接发送、后续块延后一块发送、EOF 时发送最后暂存块；读取缓冲区设为 4KB，以降低客户端在收到完成事件后立即断开导致上游收尾读取被取消的概率。

- 代理响应链路重构：合并 `Forward` 与 `Response`，由 `Forward` 统一负责上游请求、响应头过滤、流式/非流式透传及原始响应缓存；新增后置 `ParseUsage` handler，完整响应交由 `parser` 统一解析后再进入用量记录与计费。删除 `proxy/sse` 响应体包装，流式 usage 聚合下沉至 `parser.ParseStreamUsage`。新增响应完整性/客户端提交状态，避免半截流计费及错误 JSON 二次写入；请求日志改用实际客户端状态码与错误响应快照。

- 新增 OpenAI Responses API 支持（透传）：新增 `parser/responses.go`（`ResponsesParser`）与格式常量 `FormatResponses = "openai-responses"`，客户端可经 `/proxy/openai-responses/:provider/v1/responses` 以 Responses 格式调用（含 `stream: true` 流式），请求原样透传到上游。解析器适配 responses 专用的 usage 字段（`input_tokens` / `output_tokens` / `input_tokens_details.cached_tokens` / `output_tokens_details.reasoning_tokens`，与 chat 的 `prompt_tokens` / `completion_tokens` 不同）与流式事件（`response.created` / `response.output_text.delta` / `response.completed` 等，无 `[DONE]` 标记），使用量统计与计费对 Responses 请求生效。鉴权仍为 `Authorization: Bearer`，模型名取请求体顶层 `model`。无 DB 结构与路由改动（路由即 `/:format/:provider/*`）。`GET v1/models` 在 `openai-responses` 协议下由 `ResponsesParser.FormatModels` 自行序列化（`{object:"list"}`，形状与 OpenAI List Models 一致；responses 暂无官方模型列表格式，实现独立于 OpenAI 结构体，各协议隔离、可独立演进）。

- 修复用户/管理员 Token 用量统计报错：当分组内 `input_tokens` 合计为 0 时，`cache_hit_rate`（`SUM(cached_tokens) / SUM(input_tokens)`）除零得 NULL，扫描进 `float64` 报 `converting NULL to float64`。统计 SQL（`StatsByUser` / `StatsByAdmin` / `Trend7d`）中 `cache_hit_rate` 统一用 `COALESCE(..., 0)` 兜底为 0，统计接口恢复正常返回。

- 超管模型管理新增「复制」功能：操作列调整为「编辑」按钮 + 「更多」下拉菜单（复制/删除），点击复制后弹出新增模型弹窗并预填该模型的全部配置（provider/model 默认为原值，可修改后再提交新增），提交即调用已有的 `/manager/models/create` 新增模型，名称唯一性由后端校验。纯前端改动，无接口/数据结构变化。

- API Key 支持明文还原查看：`api_keys` 表新增 `key_enc` 列（AES-256-GCM 密文，复用 `service/crypto.go` 加密能力，purpose=`:api-key` 与 TOTP/Provider 隔离）。新建 key 时同时写入哈希（鉴权比对）与密文（可还原）；新增查看接口 `/manager/apikeys/reveal/self`（普通用户）与 `/manager/apikeys/reveal`（超管），解密返回明文。前端 API Key 列表 Key 列后增加复制图标，点击直接复制明文到剪贴板（不弹窗），旧版本创建的 key（`key_enc` 为空）提示无法还原。鉴权链路不变（仍走 `key_hash` 等值查找）。存量库需手动迁移 `ALTER TABLE api_keys ADD COLUMN key_enc TEXT NOT NULL DEFAULT ''`，存量 key 明文已不可还原。

- API Key 模型访问弹窗改为搜索式选择：不再一次性加载全量模型，候选列表默认只展示前 10 条，输入关键词走服务端模糊搜索（300ms 防抖），匹配超过 10 条时提示精确查找；已选模型以可关闭标签常驻顶部展示，勾选状态与搜索结果解耦。配套接口调整：`/manager/apikeys/models/get(/self)` 响应新增 `models` 字段（白名单模型的 id/provider/model 简要信息，供前端常驻展示已选模型）。

- 代理路由组织调整：proxy 的路由注册与 echo 适配（参数提取、响应写入）从 `framework/echo.go` 下沉到 `proxy/router/router.go`（与 `manager/router` 同构），`framework/echo.go` 只保留路由组创建与全局中间件；`EchoProxyDirectResponseWrite` 改为包内未导出的 `echoResponseWrite`。行为不变。

- 新增 `GET v1/models` 模型列表端点：返回当前 Provider 下、且当前 API Key 有权访问的模型列表（数据源为本地 `models` 表，与代理鉴权口径一致，不透传上游、不计费）。框架层注册具体路由（`GET /proxy/:format/:provider/v1/models`，静态段优先于通配符）进入独立入口 `proxy.HandleModels`，链路为 `ParseRequest → AuthKey → ListModels`；非 GET 的同路径请求仍落回转发链路。该端点不写 `request_logs`。响应按路径中的协议格式序列化：OpenAI 为 `{object:"list"}`，Anthropic 为 `{data,first_id,last_id,has_more}`（本地全量返回，不支持分页参数）。配套重构：原 `Auth` handler 拆分为 `AuthKey`（Key/用户校验，两条链路共用）+ `AuthModel`（模型定价 + 白名单校验，仅转发链路），鉴权行为不变。

- 请求头脱敏规则加强：`request_logs.request_headers` 的敏感头识别由精确名单（5 个）改为关键词包含匹配（auth/cookie/key/token/secret/password），覆盖 Azure `api-key`、`X-Auth-Token` 等各类自定义凭证头，防止漏脱。

- API Key 哈希化：`api_keys` 表不再存 key 原文，改存 `key_hash`（SHA-256，鉴权比对）+ `key_show`（展示串 `sk-abc****xyz`，创建时由 `service.ApiKeyShow` 生成）。明文 key 仅创建时返回一次。存量库需手动迁移（RENAME 列 → 逐行回填哈希/展示串 → 重建唯一索引，哈希需外部计算，步骤见 `sql/sqlite.sql` 注释）。鉴权链路（Auth/BudgetCheck/扣费）全部改为哈希或 ID 比对。同步调整：创建 API Key 时名称改为必填（key 不再可查看，名称是识别用途的唯一途径）。

- `request_logs` / `usage_records` 不再存储完整 API Key，改存 `api_key_id`（DB 泄露不再暴露 key 原文与调用记录的关联）。统计接口对外契约不变（筛选传 `api_key_id`、按 key 分组展示脱敏 key + 名称）；已删除的 key 在分组统计中展示为 `#id + 已删除`。存量库需手动迁移（加列 → 回填 → DROP 旧列，SQL 见 `sql/sqlite.sql` 注释，需 SQLite ≥ 3.35）。

- 密钥管理重构：签名密钥（JWT/2FA 票据）与加密密钥（TOTP/Provider 配置落库加密）拆分为两把独立密钥。加载优先级为环境变量（新增 `AIAPI_CRYPTO_SECRET`）> 密钥文件（`<DataDir>/keys/*.key`，0600 权限）> 自动生成并写文件——不设置环境变量也可启动（旧版本缺失环境变量会启动失败，行为变更）。旧部署升级时加密密钥自动从 `AIAPI_JWT_SECRET` 播种，历史密文（2FA 密钥、provider 配置）保持可解，请保持原环境变量至少完成一次启动。

- 安全加固：Provider 配置（含上游 API Key）落库改为 AES-256-GCM 加密（加密密钥独立于签名密钥按用途派生，新增 `service/crypto.go` 通用加密模块，与 TOTP 密钥加密用途隔离）。存量明文配置读取时自动兼容，管理端重新保存后转为密文（渐进迁移，无需 DDL）。
- 修复代理日志泄露完整 API Key：`budget.go` 余额不足错误不再打印 key 原文，改为打印 `apiKeyId`（Auth 阶段已注入 ctx，可定位问题且不落任何 key 材料）。

- 新增 TOTP 两步验证（2FA）：管理台用户可在个人设置页自助绑定 Authenticator（扫码/手动密钥 + 首个验证码确认），开启后登录需密码 + 6 位动态码两步。TOTP 密钥 AES-256-GCM 加密入库（密钥派生自 `AIAPI_JWT_SECRET`）；密码验证通过后仅签发 5 分钟 pending 票据，同一票据验证码连续错 5 次作废；关闭 2FA 需校验密码。新增接口 `/manager/login/2fa`、`/manager/2fa/{setup,confirm,disable}/self`；`users` 表新增 `totp_secret` 列（存量库需 `ALTER TABLE` 迁移，并重跑 `sql/init-data.sql` 补普通用户角色的 2FA 接口权限）；新依赖 `github.com/pquerna/otp`。

- 移除 pprof 调试接口：`--add-pprof` 启动参数与 `/debug/pprof/*` 路由全部删除（此前开启后无任何鉴权，会公开 goroutine/堆等运行时信息）。README 中相关说明同步移除。

### 2026-07-27

- 修复前端日历面板英文显示：`n-config-provider` 补 `:date-locale="dateZhCN"`，naive-ui 的日期/日历组件文案由独立的 date-locale 控制，此前只设了 `locale="zhCN"` 导致月份/星期仍显示英文。

- manager 业务错误码收敛：`manager/base/bizcode.go` 从 19 个码精简到 7 个（`CodeSuccess`/`CodeUnknown`/`CodeBadRequest`/`CodeUnauthorized`/`CodeForbidden`/`CodeNotFound`/`CodeTokenExpired`）。只保留有消费方按 code 分支（前端唯一依赖 `CodeTokenExpired` 1016 触发 `/refresh`）或 HTTP 状态语义不同的码；其余业务错误（已存在/不存在/额度不足/密码错误等）统一用通用码 + 中文错误信息区分，不再为每种业务定义独立码。删除死码 `CodeUserDisabled`。
  - handler 错误信息全部中文化；DB 等内部错误统一返回预置实例 `base.ErrInternal`（"系统繁忙，请稍后重试"），替代约 60 处 `NewBizError(CodeUnknown, ...)` 样板。
  - 新增便捷构造 `base.ErrBadReq(msg)` / `base.ErrNotFound(msg)`。
  - 登录失败（账号错/密码错/禁用）HTTP 状态由 401 调整为 400，防枚举文案不变；前端对 `/login` 本就排除 refresh 重试，无影响。
  - `CodeTokenExpired` 编号 1016 不变，前端零改动。

### 2026-07-26

- 列表接口全量分页：`api_key`、模型、提供商、用户列表均改为分页查询，避免全量数据查询过多导致数据库卡死。
  - `ListApiKeySelf`/`ListApiKeyAdmin` 返回 `*base.PageResult[apiKeyItem]`
  - `ListModelsAdmin` 返回 `*base.PageResult[modelItem]`（`ListModels` 保持全量，用于下拉框填充）
  - `ListProviders` 返回 `*base.PageResult[providerItem]`
  - `ListUsers` 返回 `*base.PageResult[userItem]`，关键字过滤从 Go 内存下推到 SQL `WHERE ... LIKE`
  - 新增 `/users/get` 接口查询单个用户信息（替代全量 `listUsers` 查找）
  - 前端各列表页加远程分页器（`n-data-table` remote 模式）
- store 包重构完成：store 回归纯 SQL 包装层，所有事务编排、跨表组装、业务判断下沉到 `service/`。涉及 A1-A6（事务编排）、B1-B2（跨表组装）、C1-C3（业务判断）共 11 项。
- `manager/service/` 迁移到顶层 `service/` 包，manager 与 proxy 共用业务层，消除 proxy 自建 service 的重复。
- store 按表/领域拆分 Store：`ModelAccessStore` 独立于 `ModelStore`（操作 `api_keys.model_policy` + `apikey_model_access` 白名单）。
- 删除 `inList` 工具函数：QueryBuilder 内置 `sqlx.In` 自动展开 `IN (:ids)`，不手动拼接占位符。
- 更新 `AGENTS.md`/`README.md` 架构说明与开发规则。

### 2026-07-25

- 新增通用分页能力（Session 状态 + 显式控制）：`store/base/page.go` 提供 `PageContext`，`manager/base/page.go` 提供 `PageReq`/`PageResult[T]`。handler 创建 `store.PageContext`，用 `store.C().SetPage(pc).Charge().List(...)` 链式调用，`QueryBuilder.Select` 检测到后自动拦截：先 `SELECT COUNT(*) FROM (<原SQL>) t` 查总数写回 `pc.Total`，再追加 `LIMIT ? OFFSET ?` 查当前页。非事务单次分页不用 `ClearPage`（Session 用完即弃）；事务内用 `SetPage`/`ClearPage` 显式控制。store 方法只需写普通 `Select`，未 `SetPage` 时退化为普通查询。`page` 1-based，`page_size` 默认 20、上限 100。分页类型分层：`manager/base` 定义 API 层 `PageReq`/`PageResult`，`store` 暴露内部 `PageContext`，handler 不直接 import `store/base`。
- 充值流水接口改为分页：`/manager/recharge/records`、`/manager/recharge/records/self`、`/manager/recharge/records/list` 入参加 `{page, page_size}`，返回 `{items, total, page, page_size}`。
- 统一 Session 架构：`store.T` 包级函数改为 `Session.T(fn)` 方法，调用方式 `store.C().T(fn)`。事务在当前 Session 上切换 tx，嵌套调用复用当前 tx（不开新事务，保证原子性）。`store.C()` 无参，Session 保存 `tx`/`page` 状态字段，不再用 `context.Value` 传业务参数（tx、分页），context 回归“请求级跨边界数据”本职。
- 重构充值业务：`store/charge.go` 的 `RechargeWithRecord`（读余额→加余额→写流水事务编排）下沉到 `service/charge.go` 的 `ChargeService`；`manager/handler/recharge.go` 改为调 service，不再自写事务。store 回归纯 SQL 包装。service 新增哨兵错误 `ErrUserNotFound`。
- 修复并发充值流水不准：原「先 SELECT 读余额再 UPDATE」是读后写，快照读拿旧值会丢更新。改为「先 UPDATE 加余额（行锁）→ 同事务内 SELECT 新余额 → before 由 after 反推」，跨 SQLite/MySQL/PostgreSQL 通用。
- 充值接口 self/admin 合并：`RechargeSelf` 设当前用户 ID 后委托 `Recharge`，`RechargeRecordsSelf` 同样委托 `RechargeRecords`，消除重复校验逻辑。
- 明确后端四层职责与事务边界（更新 `AGENTS.md`）：
  - `store/` 为纯 SQL 包装层，只做单表读写，不写跨表编排与业务判断。
  - `service/` 承载跨 handler 复用业务逻辑与多表事务编排（manager 与 proxy 共用）；`manager/handler/` 只做 HTTP 适配，不直接写事务。
  - 事务体用 `store.T(fn)` 包裹，写在 `service/`（manager 与 proxy 共用业务层），不写在 `store/` 内。
  - 新增 §2.4 并发安全规则：金额/计数类写操作必须「先 UPDATE 后事务内 SELECT」，禁止读后写、禁止依赖 `RETURNING`/`FOR UPDATE`（跨库不通用）。
  - 新增 §2.5 注释风格：store 方法注释只说做什么，不描述事务用法/调用时机。
  - 新增 §7.3 self/admin 合并模式约定。
  - 同步补充分层表、handler/service 边界、新增 Manager API / 数据库实体的步骤要求。

### 2026-07-24

- 修复重启服务后登录态丢失问题：refresh cookie 的 `Secure` 标记原硬编码为 `true`，导致 HTTP（非 HTTPS）部署下浏览器拒绝存储/发送该 cookie，重启后 access JWT 失效触发 refresh 时拿不到 cookie，被迫重登。现改为跟随当前请求是否 HTTPS 动态判定（`c.Scheme()=="https"`）：HTTPS 直连或反代透传 `X-Forwarded-Proto` 时开启 `Secure`，HTTP 下关闭，本地开发与纯 HTTP 部署不再丢登录态。
- 新增双 token 登录机制（access JWT + refresh 轮换 + 重用检测），替换原单 token 方案，适配公网部署。
  - **access JWT**：HS256 自实现，15min 有效期，无状态不落库，经 `Authorization: Bearer` 头传递，前端存内存（不放 localStorage 防 XSS）。
  - **refresh token**：随机串哈希后存 `user_sessions` 表（明文不落库），HttpOnly Secure cookie `refresh_token` 传递（Path=`/manager`，SameSite=Lax），滑动 7 天 / 绝对 30 天。
  - **轮换**：每次 `/manager/refresh` 删旧发新（同 family），滑动续期顺延但不超绝对上限。
  - **重用检测**：旧 refresh token 再次被使用时吊销整个 family，强制重登（业务码 1017）。
  - 改密 / 禁用用户 / 重置密码后吊销该用户所有会话，即时失效。
  - 账号不存在 / 禁用 / 密码错统一返回相同文案，防账号枚举。
  - 新增 `service` 包封装登录业务（`SessionService`），handler/middleware 仅做 HTTP 适配。
  - 新增接口 `POST /manager/refresh`（不挂 Auth 中间件，靠 refresh cookie）；`/manager/login` 返回体改为 `{access_token, expires_in}`。
  - access JWT 签名密钥走环境变量 `AIAPI_JWT_SECRET`（≥32 字节），启动时由 `base.LoadJWTSecret` 校验，缺失或过短启动失败。
  - 业务码新增 `CodeTokenExpired`(1016) / `CodeSessionReuse`(1017)。
  - 数据库变更：`user_sessions` 表新增 `family_id` / `absolute_expires_at` / `ua` / `ip` 列，新增 `idx_user_sessions_family` 索引。**存量库迁移**：执行 `ALTER TABLE user_sessions ADD COLUMN ...` 四条 + 建索引 + `DELETE FROM user_sessions;`（旧 token 格式不兼容，用户需重登一次）。
- 前端适配双 token 登录机制：access JWT 存内存；401 且业务码为 1016 时自动调 `/manager/refresh` 续期并重试原请求；并发请求合并（多个 401 只触发一次刷新）；页面刷新后靠 refresh cookie 静默恢复会话；`/login` 已登录则跳首页，登出 / 改密后清 access token。

### 2026-07-23

- 修复统计页 / 全局统计 / 仪表盘时间趋势折线倒序问题：时间维度由 `label DESC` 改为 `ASC`（从旧到新正序），维度分组（模型 / 提供商 / Key / 用户）保持 DESC 不变。

### 2026-07-22

- 个人设置页改用 `n-form` 左对齐布局：`Profile` / `ChangePassword` 表单 label 左对齐，按钮区与输入框左边缘对齐；修改密码页顶部加浅红色提示「修改密码后需要重新登录」。

- 新增个人设置功能（修改姓名/邮箱/密码）。
  - `users` 表新增 `email` 字段（可空、不唯一）。
  - 新增 2 个自助接口：`/manager/profile/update/self`（改姓名+邮箱）、`/manager/profile/password/self`（改密码，需校验旧密码）。
  - 修改密码后前端提示并跳转登录页，强制重新登录。
  - 新增「个人设置」菜单项（id=11，所有角色可见），`/profile` 页面包含基本资料与修改密码两个区块。
  - `/manager/self` 返回的 user 信息补充 `email` 字段。
  - 存量库迁移：`ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT '';` + `INSERT OR IGNORE INTO menus ...` / `role_menus ...`。

- 新增 API Key 级模型访问控制。
  - `api_keys` 表新增 `model_policy` 字段（默认 `'all'`），`all`=全量放行，`whitelist`=按白名单。
  - 新增 `apikey_model_items` 表存储白名单明细（model_policy=`whitelist` 时生效）。
  - Proxy Pipeline 在 Auth handler 内追加模型权限校验（紧跟模型配置校验之后），按 Key 策略判定是否可访问请求模型，不通过时返回 404（对外不暴露权限细节）。
  - 管理端新增 4 个接口：超管版 `/manager/apikeys/models/get`、`/set`；普通用户自助版 `/apikeys/models/get/self`、`/set/self`（自助版校验 Key 归属）。
  - 前端新增可复用 `ModelAccessDialog` 组件，超管与普通用户的 Key 管理页操作列均加「模型权限」按钮，支持全量/白名单切换，白名单按 provider 分组多选。
  - 删除 API Key 时事务内级联清理白名单明细。
  - 存量库迁移：`ALTER TABLE api_keys ADD COLUMN model_policy TEXT NOT NULL DEFAULT 'all';` + 建 `apikey_model_items` 表。

- 修复全局统计「按 Key」分组时表格不显示 Key 名称的问题。admin 版原先因「跨用户 key 名不唯一」而不填充名称，导致主行只显示「-」；现改为查全部 Key 构建 key→name 映射并填充，同时优化前端渲染：有名称时显示「名称 + 脱敏 Key」双行，无名称（已删除/未命名）时直接显示脱敏 Key 单行。普通用户统计页同步优化。

- 超管侧栏菜单「模型定价」改名为「模型管理」，相关页面卡片标题与代码注释同步统一。存量库需执行 `UPDATE menus SET name='模型管理' WHERE id=8;`。

- 模型多模态能力配置（第一阶段：配置 + 展示）。
  - `models` 表新增 `supports_text` / `supports_image` / `supports_video` 三个布尔列（默认 `1/0/0`），分别标记模型支持的文本/图像/视频模态。
  - 模型新增/编辑接口与表单同步支持这三项配置，新建模型默认勾选「文本」。
  - 普通用户与超管的模型列表均新增「能力」列，以彩色 `n-tag` 展示已支持模态。
  - 存量库迁移：执行 `ALTER TABLE models ADD COLUMN ...` 三条语句即可，详见 `sql/sqlite.sql` 内注释。

### 2026-07-21

- 新增超级管理后台（超管视角的全局管理功能）。
  - **动态菜单**：新增 `menus` / `role_menus` 表，`/manager/self` 返回当前用户菜单树，前端 `Home.vue` 动态渲染侧栏（支持分组容器/直接跳转/纯标题三种形态），不再硬编码菜单。
  - **超管特权**：`role_permission` 支持 `value='*'` 通配放行所有接口，admin 角色只需一条权限记录即可访问全部超管接口。
  - **用户管理** `/manager/users/*`：列表（含角色 tag）/ 创建 / 编辑 / 启停 / 重置密码 / 分配角色 / 充值；操作列下拉菜单含「API Key」「充值记录」入口，跳转子页面管理该用户的 Key 和流水。
  - **提供商管理** `/manager/providers/*`：列表 / 新增 / 编辑 / 查看（headers 键值对格式化展示）/ 启停；禁用前二次确认。
  - **模型定价管理** `/manager/models/*`：列表（后端按 provider/model 模糊搜索）/ 新增 / 编辑 / 删除。
  - **全平台充值记录** `/manager/recharge/records/list`：JOIN users 拿用户名和操作人名，支持按用户名/账号/备注模糊搜索。
  - **全局统计** `/manager/usage/stats` `/manager/usage/filters`：不限定 user_id，支持 `group_by=user` 维度；筛选下拉含用户列表；复用现有图表框架。
  - **仪表盘** `/manager/dashboard`：8 个指标卡（用户数/Key数/今日请求/今日费用/今日输入Token/今日输出Token/今日总Token/今日缓存命中率）+ 近 7 天趋势图（6 指标切换：费用/请求数/输入Token/输出Token/总Token/缓存命中率）。
  - **API Key 管理（超管版）** `/manager/apikeys/*`（无 `/self` 后缀）：超管管理指定用户的 Key，不校验归属，额度校验用 Key 归属用户的余额。
  - **角色列表** `/manager/roles/list`：用于分配角色弹窗。
  - 数据库变更：新增 `menus` / `role_menus` 表；`roles` 表去掉 `description` 列、新增 `code` 列；`users` 表去掉 `updated_at` 列；`recharge_records` 查询 JOIN users 返回 `user_name` / `operator_name`。
  - 业务码新增：`CodeAccountExists(1011)` / `CodeProviderExists(1012)` / `CodeProviderNotFound(1013)` / `CodeModelExists(1014)` / `CodeModelNotFound(1015)`。
  - 新增 `sql/init-data.sql` 独立存放初始数据（角色/权限/菜单/角色菜单），用 `INSERT OR IGNORE` 可重复执行。

### 2026-07-20

- Token 用量统计增强。
  - 统计新增字段：缓存命中 Token (`cached_tokens`)、缓存未命中 Token (`cache_miss_tokens`)、推理 Token (`reasoning_tokens`)、缓存命中率 (`cache_hit_rate` = cached_tokens / input_tokens)，表格与顶部指标均已覆盖。
  - 接口响应结构调整：`/usage/stats/self` 改为返回 `{summary, rows}`，顶部汇总由后端计算。
  - 按 api_key 分组时，已删除的 key 名称标记为红色（「已删除」）。
  - 日期查询改为左闭右闭（`00:00:00 ~ 23:59:59`），按天模式含结束日全天数据。
  - 空结果时 `rows` 返回 `[]` 而非 `null`，前端兜底处理。
  - 移除 `/usage/detail/self` 接口相关代码。

- 新增 ECharts 图表功能。
  - 时间趋势：费用折线图 / 总 Token 多线面积图 + 费用双轴 / 缓存命中面积图 / 缓存命中率百分比折线图。
  - 按维度（模型/提供商/Key）：环形图展示占比；比率指标用水平柱状图。
  - 支持 费用/总Token/缓存命中/缓存命中率 四个指标切换。
  - 图表代码重构为可复用模块：`composables/useChart.js`（生命周期）、`charts.js`（option 构建器）。
  - 依赖新增 `echarts`。

- 前端统一优化。
  - 金额格式化统一用 `fix4`（提取到 `utils.js`），时间格式化 `formatTime` 提取到 `utils.js`。
  - 所有页面 `n-data-table` 风格对齐：`bordered=false`、`size=small`、`scroll-x` 防溢出、时间列 `ellipsis` 防换行。
  - 充值流水操作人展示 `users.name`（`ListRechargeRecords` LEFT JOIN users）。
  - 充值/模型/密钥页面空数据 `|| []` 兜底，避免 null 导致页面卡死。
  - 默认首页改为使用统计，默认按月查询当月数据，修复 UTC 时区导致日期偏差。
  - 侧栏菜单顺序：使用统计 → API 密钥 → 模型列表 → 充值中心。

- 新增 Token 用量与费用统计功能。
  - 新增管理接口（均需登录 + 接口授权）：
    `POST /manager/usage/stats/self`、`/usage/filters/self`。
  - 支持按天/月聚合统计（天模式限31天），可按 api_key / 模型 / 提供商组合筛选。
  - api_key 筛选用 id 而非完整 key（防泄露）。
  - `store/usage.go` 新增 `StatsByUser` / `DistinctModelsByUser` / `DistinctProvidersByUser`。
  - 前端新增「使用统计」页面（`frontend/src/views/Usage.vue`）。
  - `sql/sqlite.sql` 种子权限注释补充 2 条 `/manager/usage/*/self` 路径。

- 新增个人 API Key 管理功能（用户自助）：列表 / 创建 / 启停 / 删除 / 重命名 / 修改额度。
  - 新增管理接口（均需登录 + 接口授权）：
    `POST /manager/apikeys/list/self`、`/apikeys/create/self`、`/apikeys/toggle/self`、`/apikeys/delete/self`、`/apikeys/rename/self`、`/apikeys/budget/self`。
  - 所有读写接口先校验 key 归属当前用户，防止越权。
  - API Key 生成采用 `crypto/rand` 32 字节随机 + `sk-` 前缀（256bit 熵），唯一性由 `uq_api_keys_key` 兜底，冲突重试 3 次。
  - 列表接口对 key 做脱敏（`sk-xxxx****xxxx`），明文 key 仅在创建响应中返回一次。
  - 额度约束：所有「有限额」密钥的 budget 之和不能超过账户余额；创建与修改额度接口均做该校验，无限账户跳过。新增 `CodeBudgetExceeded`（1010）。
  - `store/apikey.go` 补充 `GetByID` / `ListByUser` / `Create` / `SetEnabled` / `UpdateName` / `UpdateBudget` / `SumLimitedBudgetByUser`，并暴露 `IsUniqueConstraintErr`。
  - `manager/base/bizcode.go` 新增 `CodeApiKeyNotFound`（1009）、`CodeBudgetExceeded`（1010）。
  - 前端新增「API 密钥」页面（`frontend/src/views/ApiKeys.vue`），含创建弹窗、明文 key 一次性展示与复制、名称列内联改名图标、修改额度弹窗（限制/无限制二选一，限制时填金额且不超账户可用额度）、启停/删除操作。
  - `App.vue` 包裹 `n-message-provider` / `n-config-provider` 以支持 `useMessage`。
  - `sql/sqlite.sql` 种子权限注释同步补充 6 条 `/manager/apikeys/*/self` 路径（admin + user 各一份）。

### 2026-07-19

- 新增后台管理前端（`frontend/`）：Vue 3 + Vite 管理台，Go `embed` 嵌进二进制。
  - 页面：登录 / 充值（弹窗充值 + 流水表格）。
  - 开发期 `make dev-ui`（端口 3000），自动代理 API 到 Go 后端。
  - 生产构建 `make build-all`，一个 `aiapi` 文件启动即带前端页面。
- 新增后台管理功能（`manager/`）：账号密码登录 / 登出、接口级权限 `(role_id, entity, action, value)`、普通用户自充值、管理员给任意用户充值。
  - 登录态基于服务端 `user_sessions` 表 + HttpOnly Cookie，不使用 `Authorization` 头。
  - 多角色（用户-角色多对多）；接口级 ACL，`role_permission` 表用 `entity=API, value=接口路径` 授权，请求路径命中即放行；`action` 保留暂作通配。
  - 所有管理接口一律 POST + JSON body；manager 自有业务码（`manager/base/bizcode.go`），与 proxy 解耦。
  - 复用已有 `recharge_records` 表与 `store.Charge()`，新增 `RechargeWithRecord` 在事务内完成"读余额→加余额→写流水"。
- 数据库结构变更（`sql/sqlite.sql`）：
  - `users` 新增 `password`（bcrypt 哈希）、`updated_at` 列。**存量库需手动迁移**：
    `ALTER TABLE users ADD COLUMN password TEXT NOT NULL DEFAULT '';`
    `ALTER TABLE users ADD COLUMN updated_at DATETIME DEFAULT (datetime('now','localtime'));`
  - 新增表：`roles`、`user_roles`、`role_permission`、`user_sessions`。
- 依赖：`golang.org/x/crypto` 提为直接依赖（bcrypt）。

### 2026-07-19

- 新增项目变更日志（CHANGELOG.md）。
- 重写 `AGENTS.md` 与 `README.md`：
  - `AGENTS.md` 改为大方向开发规则，明确分层边界、Pipeline 设计、扩展规则与文档维护职责。
  - `README.md` 明确项目当前仅代理 OpenAI 兼容格式请求，修正架构描述与使用示例。

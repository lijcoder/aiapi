# TODO - 待优化项

> 记录**已识别但暂不实施**的优化项；实施后从本文档移除，并在 `CHANGELOG.md` 记录。
> 开发规则见 [`AGENTS.md`](AGENTS.md)，架构约定与已知偏离见 [`docs/architecture.md`](docs/architecture.md)，
> 定过的设计见 [`docs/decisions/`](docs/decisions/README.md)。**已接受的现状不在这里**（如余额并发可透支，见 architecture.md §6）。

### 1. request_logs 治理：加索引 + body 截断/脱敏开关

- **现状**：`request_logs` 无任何索引，且 `request_body` / `response_body` 全量入库；表随时间线性膨胀，按时间查询/清理都是全表扫。
- **风险 / 影响**：长期运行后表体积与查询耗时不可控；全量 body 可能含用户隐私内容，对外服务时有合规风险。
- **方案**：① 加 `created_at` 索引（需一次迁移，走 `sql/migrations/`）；② body 落库前截断（如 64KB）并提供脱敏开关；③ 可选：定期清理与异步批量写。
- **前置条件**：对外开放服务前应至少完成 ②。
- **关联**：`proxy/handler/log.go`、`sql/sqlite.sql`、`README.md` 的「已知限制」。

### 2. 热路径配置缓存

- **现状**：每个代理请求都要查 `models` / `providers` / `api_keys.model_policy`，无进程内缓存。
- **风险 / 影响**：SQLite 上量后成为吞吐瓶颈；单机场景目前可接受。
- **方案**：进程内 TTL 缓存 + 管理端变更时主动失效；**多实例部署时需改为共享态或接受 TTL 内不一致**。
- **关联**：`proxy/handler/auth_model.go`、`config.go`、`service/model.go`。

### 3. Auth 与 BudgetCheck 重复查询 Key 与 User

- **现状**：`AuthKey` 与 `BudgetCheck` 各自调用一次 `service.GetKeyAndUser`，同一请求查两遍。
- **方案**：Auth 查到的结果放进 `ctx`，BudgetCheck 复用。
- **关联**：`proxy/handler/auth_key.go`、`budget.go`、`proxy/types/context.go`。

### 4. 登录接口限流与失败锁定

- **现状**：已做统一文案 + 时序对齐防枚举，但无速率限制与失败锁定。
- **风险 / 影响**：公网暴露时密码可被在线爆破（内网可缓，公网必做）。
- **方案**：按 IP + 账号失败 N 次锁定；反向代理层限流可作起步方案（见 README「生产部署检查清单」）。
- **关联**：`manager/handler/login.go`。

### 5. 过期数据清理任务

- **现状**：`user_sessions` 的过期行无清理机制（启动时清理已移除），长期运行缓慢堆积，仅体积问题、无安全风险。
- **方案**：每日定时调用现成的 `UserSession().DeleteExpired`；与第 1 项一起做可共用后台任务骨架。
- **关联**：`store/session.go`、`main.go`。

### 6. OpenAI 非流式用量口径与流式不一致

- **现状**：`parser/openai.go` 的非流式 `ParseUsage` 直接取上游 `total_tokens`，**不做回退**（回退函数 `openaiTotalTokens` 只在流式路径被调用）；而 `parser/AGENTS.md` 的口径表与 CHANGELOG 声明「优先上游值，缺省回退 输入 + 输出」，Responses 非流式确实回退。
- **风险 / 影响**：上游省略 `total_tokens` 时 `usage_records.total_tokens = 0`，而该字段参与分段计费的条件匹配，可能命中错误的计费规则；同一场景流式却会回退。
- **方案**：非流式也走 `openaiTotalTokens` 回退。golden fixture（`parser/testdata/openai_response_total_missing.json`）已就绪，改完直接补断言。
- **关联**：`parser/openai.go`、`parser/golden_openai_test.go`。

### 7. 非流式「上游没给 usage」被记成 0 用量

- **现状**：三个协议的非流式解析在 usage 缺失时返回**非 nil 的零用量**，只有流式路径有零用量守卫；OpenAI 非流式还缺少响应类型校验（Anthropic 校验 `type=="message"`、Responses 校验 `object=="response"`）。
- **风险 / 影响**：上游返回 200 的错误体（或缺少 usage 的正常体）会被 `proxy/handler/record.go` 当成有效用量，写入零 token 的 `usage_records` 并参与计费规则匹配。
- **方案**：非流式补上与流式一致的零用量守卫；OpenAI 补响应类型校验（或明确记录"为何不做"）。
- **关联**：`parser/openai.go`、`parser/anthropic.go`、`parser/responses.go`、`proxy/handler/record.go`。

### 8. SSE 多行 `data:` 未按规范合并

- **现状**：`parser/util/sse.go` 逐行回调，不把同一事件的多行 `data:` 按 SSE 规范用换行拼接。
- **风险 / 影响**：上游把同一个 JSON 拆到多行 `data:` 时，各片段都解析失败，用量**静默丢失**（不计费也不报错，排查困难）。
- **方案**：按事件聚合 data 行后再回调。fixture `anthropic_messages_stream_multiline_data.sse` 已固化当前行为，改后需同步更新断言。
- **关联**：`parser/util/sse.go`、`parser/testdata/`。

## 登记格式

新增时按上面格式追加，编号递增（不复用旧编号）：

```markdown
### <编号>. <一句话标题>
- **现状**：现在是什么样
- **风险 / 影响**：不改会怎样
- **方案**：打算怎么做（可选方案一并列出）
- **前置条件**：什么情况下才需要做
- **关联**：相关文件 / 章节
```

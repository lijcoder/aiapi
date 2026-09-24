# AGENTS.md — 协议解析层

本文件补充仓库级规则 [`../AGENTS.md`](../AGENTS.md)，只写 `parser/` 特有的约定。

`parser/` 只做协议相关的解析与提取：**不操作数据库、不写响应、不依赖 store**（有门禁校验，见根目录 `arch_test.go`）。协议无关的纯工具放 `parser/util/`（无状态纯函数，且不反向依赖 `parser`）。

## 现状

| format | 状态 | 客户端鉴权头 | 上游路径 |
|--------|------|--------------|----------|
| `openai` | ✅ | `Authorization: Bearer` | 由调用方决定，如 `v1/chat/completions` |
| `anthropic` | ✅ | `x-api-key` → 回退 `Authorization` | 如 `v1/messages` |
| `openai-responses` | ✅ | `Authorization: Bearer` | 如 `v1/responses` |
| `gemini` | ❌ **未实现**（`GetParser` 返回 `nil`） | — | — |

`gemini` 未实现的后果：`ctx.P == nil` → `ParseRequest` 不填任何字段 → 请求最终以 401 `missing api key` 失败。补齐它不只是加一个解析器：Gemini 的模型名在 **URL path** 而非请求体，`ReplaceModel` 需要 path 版本，handler 侧要支持改写 path。

## 新增协议

1. 在 `parser/` 下新增解析器，实现 `Parser` 接口（`parser/interface.go`），并加编译期断言（现有三个解析器都有）。
2. 在 `parser/interface.go` 注册，让 `GetParser` 返回它；未实现的 format 显式返回 `nil`，不要返回半成品。
3. 模型名成对实现：`ParseModel`（提取，供鉴权/计价）与 `ReplaceModel`（替换为上游模型名，供转发）。请求体顶层的实现复用 `parser/util.ReplaceTopLevelModel`；模型名在别处的协议自行处理，并保证**无需改写时原样返回入参 body、字节零变动**。
4. 用量解析分 `ParseUsage`（非流式）与 `ParseStreamUsage`（流式），都由本协议实现：流式用 `parser/util.EachSSEData` 取 SSE 的 data 载荷，事件结构、字段合并与 `total_tokens` 口径按本协议定义，**不引入跨协议事件结构**。
5. `ParseApiKey` 按本协议的鉴权头优先级调用 `parser/util.ExtractBearerToken(headers, 头名...)`，按顺序取第一个非空头；值带 `Bearer ` 前缀会剥掉，裸 token 原样返回。
6. 协议相关的响应序列化（如模型列表）也归本协议：放不进 `Parser` 主接口的能力用可选接口 + 类型断言（参考 `ModelsFormatter` / `FormatModelList`），由业务层把数据映射成 parser 中立结构。

## 各协议口径

| 协议 | `total_tokens` | 其他 |
|------|----------------|------|
| Anthropic | 完整输入 + 输出（完整输入 = `input_tokens` + 缓存创建 + 缓存读取） | 流式取 `message_delta` 的累计值；`thinking_tokens` 计入 `reasoning_tokens` |
| OpenAI / Responses | 优先取上游 `usage`，缺省回退 输入 + 输出 | OpenAI 流式需要带 `usage` 的 chunk（`stream_options.include_usage`），否则不记用量 |

## 测试

`parser/` 的 golden file 测试放在 `testdata/`，测试与源码**同包**。改动必须覆盖**流式与非流式**两种场景，以及非法 JSON、usage 缺失等失败路径。

## 自检

- [ ] 流式与非流式用量都有测试覆盖
- [ ] `ReplaceModel` 无需改写时字节零变动
- [ ] 新增 format 时 `README.md` 的协议表同步

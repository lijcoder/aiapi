# AI-Agent 代理系统

## 项目概述

一个轻量级 AI 大模型 API 反向代理，支持多厂商路由、Token 用量统计、SSE 流式透传、请求日志。

## 技术栈

- Go 1.22+
- Echo v4（HTTP 框架）
- SQLite（数据库，`mattn/go-sqlite3`）

## 项目结构

```
aiapi/
├── main.go                  # 入口：解析参数 → 初始化 DB → 启动 HTTP 服务
├── constant/
│   └── constant.go          # 全局常量（端口、内存限制等）
├── db/
│   ├── models.go            # 数据模型（Provider, User, ApiKey, UsageRecord, RequestLog）
│   ├── schema.go            # DDL（参考用，手动建表）
│   └── store.go             # CRUD + Init()
├── parser/
│   ├── interface.go         # Parser 接口 + 格式常量 + ExtractApiKey
│   └── openai.go            # OpenAI 响应解析实现
├── proxy/
│   ├── direct.go            # Handle 入口
│   ├── pipeline.go          # Pipeline 框架（AddLast / AddFinally / Execute）
│   ├── types/               # 共享类型
│   │   ├── context.go       # Context, ProxyRequest, ProxyResponseWrite
│   │   └── bizcode.go       # BizCode 枚举（CodeUnknown, CodeNotFound）
│   ├── handler/             # 业务 handler
│   │   ├── config.go        # LoadConfig — 从 DB 加载 provider 配置
│   │   ├── forward.go       # Forward — 转发请求到上游
│   │   ├── response.go      # Response — 处理响应（流式/非流式）
│   │   ├── record.go        # Record — 异步记录 token 用量
│   │   └── log.go           # Log — 保存请求日志（AddFinally）
│   └── sse/
│       └── body.go          # SSE 流式响应体包装器
├── manager/
│   └── provider.go          # Provider CRUD HTTP API
└── framework/
    └── echo.go              # Echo 初始化 + 路由注册 + 参数解析
```

## 核心路由

| 方法 | 路径 | 说明 |
|------|------|------|
| `ANY` | `/proxy/:format/:provider/*` | 反向代理 |
| `POST` | `/manager/providers` | 新增 Provider |
| `GET` | `/manager/providers` | 列表 |
| `GET` | `/manager/providers/:type` | 详情 |
| `DELETE` | `/manager/providers/:type` | 删除 |

- `:format` = `openai` / `gemini` / `anthropic`（API 协议格式）
- `:provider` = provider 表的 type 字段（后端实际提供商）
- `:*` = 上游路径（如 `v1/chat/completions`）

## 管道流程

```
Handle(req)
  │
  └─ NewPipeline()
       ├── AddLast(handler.LoadConfig)   ← 查 DB 加载 provider
       ├── AddLast(handler.Forward)      ← 转发请求到上游
       ├── AddLast(handler.Response)     ← 流式/非流式分流处理
       ├── AddLast(handler.Record)       ← 记录 token 用量（成功时）
       ├── AddFinally(handler.Log)       ← 始终执行：写请求日志
       └── Execute(ctx)
              │
              ├── for 循环遍历 handlers，遇 ctx.Err != nil 终止
              ├── writeError(ctx)        ← 有错误写 HTTP 响应
              └── finally handlers        ← 日志始终记录
```

各 handler 不检查 ctx.Err，Pipeline 的 for 循环控制流程，出错自动跳过后续 handler。Finally handler 不受中断影响。

## 数据库

路径：`~/.aiapi/aiapi.db`，启动时自动建表。

### providers
```sql
CREATE TABLE providers (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    type  TEXT NOT NULL UNIQUE,    -- 唯一标识，URL 中的 :provider
    config TEXT NOT NULL,          -- JSON: {"domain":"...","headers":{...}}
    enabled INTEGER DEFAULT 1,
    created_at DATETIME,
    updated_at DATETIME
);
```

### api_keys
```sql
CREATE TABLE api_keys (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    key        TEXT NOT NULL UNIQUE,
    name       TEXT DEFAULT '',
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME
);
```

### usage_records
```sql
CREATE TABLE usage_records (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key       TEXT DEFAULT '',
    provider      TEXT NOT NULL,
    model         TEXT NOT NULL,
    input_tokens  INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    total_tokens  INTEGER DEFAULT 0,
    request_id    TEXT DEFAULT '',
    stream        INTEGER DEFAULT 0,
    created_at    DATETIME
);
```

### request_logs
```sql
CREATE TABLE request_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key         TEXT DEFAULT '',
    format          TEXT NOT NULL,
    provider        TEXT NOT NULL DEFAULT '',
    method          TEXT NOT NULL,
    path            TEXT NOT NULL,
    status_code     INTEGER DEFAULT 0,
    request_headers TEXT DEFAULT '',       -- JSON，敏感头脱敏
    request_body    TEXT DEFAULT '',
    model           TEXT DEFAULT '',
    input_tokens    INTEGER DEFAULT 0,
    output_tokens   INTEGER DEFAULT 0,
    total_tokens    INTEGER DEFAULT 0,
    error           TEXT DEFAULT '',
    latency_ms      INTEGER DEFAULT 0,
    created_at      DATETIME DEFAULT (datetime('now'))
);
```

## 扩展规则

### 新增 Parser（如 Gemini、Anthropic）

1. 在 `parser/` 下新建文件（如 `gemini.go`）
2. 实现 `Parser` 接口：
   - `ParseUsage(body []byte) (*Usage, error)` — 非流式
   - `ParseStreamEvent(data []byte) (*StreamEvent, error)` — 流式 SSE
3. 在 `parser/interface.go` 注册：
   - `var Gemini Parser = &GeminiParser{}`
   - `case FormatGemini: return Gemini`

### 新增 Pipeline Handler

1. 在 `proxy/handler/` 下新建文件
2. 写一个函数 `func Xxx(ctx *types.Context)`：
   - 成功时直接 return（`ctx.Err` 保持 nil）
   - 失败时设 `ctx.Err` + `ctx.Code` 后 return
3. 在 `proxy/direct.go` 的 pipeline 里 `.AddLast(handler.Xxx)`
4. 如果是始终执行的收尾操作，用 `.AddFinally(handler.Xxx)`

### 新增 Manager API

1. 在 `manager/` 下新建文件（或扩展 `provider.go`）
2. 在 `manager/provider.go` 的 `RegisterRoutes` 中注册路由

### 新增错误码

在 `proxy/types/bizcode.go` 追加一行：

```go
CodeXxx = BizCode{1002, 404}  // → 返回 HTTP 404
```

## 关键约定

- **echo.go 只做 HTTP 参数解析**，不包含业务逻辑
- **proxy/direct.go Handle 只做管道组装**，不写业务代码
- **proxy/handler/** 是业务核心，每个 handler 独立可测试
- **错误处理**：handler 只设 `ctx.Err` + `ctx.Code`，Pipeline 统一调 `writeError`
- **Token 统计**：非流式走 `interceptResponse`，流式走 `streamResponse`，统一由 `Record` handler 记录
- **失败不记录**：`status_code >= 300` 时跳过 usage_records，但 request_logs 始终记录
- **API Key 提取**：通过 `parser.ExtractApiKey(headers, format)` 统一获取

## 启动

```bash
go build -o aiapi .
./aiapi --port 8888
```

# AI-Agent 代理系统

## 项目概述

一个轻量级 AI 大模型 API 反向代理，支持多厂商路由、Token 用量统计、SSE 流式透传。

## 技术栈

- Go 1.22+
- Echo v4（HTTP 框架）
- SQLite（数据库，`mattn/go-sqlite3`）
- 无其他外部依赖

## 项目结构

```
aiapi/
├── main.go                  # 入口：初始化 DB → 启动 HTTP 服务
├── constant/
│   └── constant.go          # 全局常量（端口、内存限制等）
├── db/
│   ├── models.go            # 数据模型（Provider, User, ApiKey, UsageRecord）
│   ├── schema.go            # DDL（参考用，手动建表）
│   └── store.go             # CRUD 操作
├── parser/
│   ├── interface.go         # Parser 接口 + 格式常量 + ExtractApiKey
│   └── openai.go            # OpenAI 响应解析实现
├── proxy/
│   ├── direct.go            # Handle 入口 + ProxyRequest + ProxyResponseWrite
│   ├── chain.go             # 链式调用核心（LoadConfig → Forward → Intercept/Stream → Record → WriteResponse）
│   └── sse.go               # SSE body 解析包装器 (io.ReadCloser)
├── manager/
│   └── provider.go          # Provider CRUD HTTP API
└── framework/
    └── echo.go              # Echo 初始化 + 路由注册 + 参数解析
```

## 核心路由

| 方法 | 路径 | 说明 |
|------|------|------|
| `ANY` | `/proxy/:format/:provider/*` | 反向代理 |
| `ANY` | `/proxy/debug/:traceid/:format/:provider/*` | 调试模式 |
| `POST` | `/manager/providers` | 新增 Provider |
| `GET` | `/manager/providers` | 列表 |
| `GET` | `/manager/providers/:type` | 详情 |
| `DELETE` | `/manager/providers/:type` | 删除 |

- `:format` = `openai` / `gemini` / `anthropic`（API 协议格式）
- `:provider` = provider 表的 type 字段（后端实际提供商）
- `:*` = 上游路径（如 `v1/chat/completions`）

## 数据库

路径：`~/.aiapi/aiapi.db`，手动建表（参考 `db/schema.go`）

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
    key        TEXT NOT NULL UNIQUE,   -- API Key
    name       TEXT DEFAULT '',        -- 备注名
    budget     REAL DEFAULT 0,        -- 预算上限(USD)，0 无限制
    enabled    INTEGER DEFAULT 1,
    created_at DATETIME
);
```

### usage_records
```sql
CREATE TABLE usage_records (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key       TEXT DEFAULT '',     -- 调用来源 Key
    provider      TEXT NOT NULL,       -- URL 中的 :provider
    model         TEXT NOT NULL,
    input_tokens  INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    total_tokens  INTEGER DEFAULT 0,
    request_id    TEXT DEFAULT '',
    stream        INTEGER DEFAULT 0,
    created_at    DATETIME
);
```

## 插件/扩展规则

### 新增 Parser（如 Gemini、Anthropic）

1. 在 `parser/` 下新建文件（如 `gemini.go`）
2. 实现 `Parser` 接口：
   - `ParseUsage(body []byte) (*Usage, error)` — 非流式
   - `ParseStreamEvent(data []byte) (*StreamEvent, error)` — 流式 SSE
3. 在 `parser/interface.go` 注册：
   - `var Gemini Parser = &GeminiParser{}`
   - `case FormatGemini: return Gemini`

### 新增链步骤

1. 在 `proxy/chain.go` 添加方法，返回 `*Chain`
2. 出错时设置 `c.err`，不要在步骤内写 HTTP 响应
3. 在 `proxy/direct.go` 的 `Handle` 里编排到链中
4. 错误响应统一由 `chain.WriteError()` 处理

### 新增 Manager API

1. 在 `manager/` 下新建文件（或扩展 `provider.go`）
2. 在 `manager/provider.go` 的 `RegisterRoutes` 中注册路由

## 关键约定

- **echo.go 只做 HTTP 参数解析**，不包含业务逻辑
- **proxy/direct.go Handle 只做链编排**，不写业务代码
- **proxy/chain.go** 是业务核心，每个步骤独立可测试
- **错误处理**：链条方法只设 `c.err`，`Handle` 统一检查后调 `chain.WriteError()`
- **Token 统计**：非流式走 `Intercept().Record()`，流式走 `StreamResponse()` 后取 `sseBody.Usage()` 再 `Record()`
- **失败不记录**：`status_code >= 300` 时不写 usage_records
- **API Key 提取**：通过 `parser.ExtractApiKey(headers, format)` 统一获取

## 启动

```bash
go build -o aiapi .
./aiapi --port 8888
```

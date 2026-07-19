# aiapi - 大模型 API 网关

一个轻量级 OpenAI 兼容格式的大模型 API 反向代理，支持多上游 Provider 路由、Token 用量统计、SSE 流式透传与请求日志。

## 功能特性

- **OpenAI 兼容代理**：客户端使用标准 OpenAI API 格式请求，统一转发给兼容的上游提供商
- **多上游路由**：可将请求转发到任意配置的 OpenAI 兼容 Provider（如 OpenAI、Azure、OneAPI 等）
- **SSE 流式透传**：支持流式聊天补全响应的透明转发
- **Token 用量统计**：自动记录 input / output tokens
- **请求日志**：保存每次请求的参数、响应状态、耗时与错误信息
- **Provider 管理**：内置 CRUD 接口动态管理上游配置
- **API Key 管理**：校验调用方身份，支持启用/禁用
- **pprof 调试**：可选启用 Go 性能分析接口

## 架构概览

```
┌─────────┐      ┌──────────────────────┐      ┌─────────────┐      ┌────────────────────┐
│ Client  │ ──▶  │  /proxy/openai/...   │ ──▶  │  Pipeline   │ ──▶  │  OpenAI 兼容上游   │
│         │      │       Echo v4        │      │  handlers   │      │     Provider       │
└─────────┘      └──────────────────────┘      └─────────────┘      └────────────────────┘
                                                        │
                                                        ▼
                                                 ┌─────────────┐
                                                 │ SQLite 存储  │
                                                 │ usage_logs  │
                                                 │ request_logs│
                                                 └─────────────┘
```

### 支持的请求格式

本项目当前仅代理 **OpenAI 兼容格式** 的请求与响应。`:format` 参数目前固定为 `openai`，`:provider` 为配置的上游 Provider 标识。

| 客户端格式 | 上游 Provider | 说明 |
|------------|-------------|------|
| openai     | openai      | 转发至 OpenAI 兼容上游 |
| openai     | azure-openai| 转发至 Azure OpenAI 兼容接口 |
| openai     | custom      | 转发至任意 OpenAI 兼容的第三方网关 |

未来如需支持其他客户端协议（如 Gemini/Anthropic），需在 `parser/` 层扩展实现转换。

## 快速开始

### 环境要求

- Go 1.22+
- 操作系统：Linux / macOS / Windows（推荐 Linux/macOS）

### 编译

```bash
go build -o aiapi .
```

### 启动

```bash
# 默认端口 8888，数据目录 ~/.aiapi
./aiapi

# 指定端口
./aiapi --port 8888

# 启用 pprof 调试接口
./aiapi --pprof
```

### 测试代理

```bash
curl http://localhost:8888/proxy/openai/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 配置 Provider

### 新增 Provider

```bash
curl -X POST http://localhost:8888/manager/providers \
  -H "Content-Type: application/json" \
  -d '{
    "type": "openai",
    "config": {
      "domain": "https://api.openai.com",
      "headers": {
        "Authorization": "Bearer sk-xxxxxxxx"
      }
    },
    "enabled": 1
  }'
```

### 列出所有 Provider

```bash
curl http://localhost:8888/manager/providers
```

### 查询指定 Provider

```bash
curl http://localhost:8888/manager/providers/openai
```

### 删除 Provider

```bash
curl -X DELETE http://localhost:8888/manager/providers/openai
```

### Provider 配置字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | Provider 唯一标识，URL 中的 `:provider` |
| `config` | object | 上游配置 |
| `config.domain` | string | 上游基础域名，如 `https://api.openai.com` |
| `config.headers` | object | 转发到上游时附加的 HTTP 头 |
| `enabled` | int | 是否启用，`1` 启用，`0` 禁用 |

## 反向代理路由

```
/proxy/openai/:provider/*
```

- `:format`：当前固定为 `openai`
- `:provider`：上游 Provider 的 `type` 字段
- `*`：上游路径，例如 `v1/chat/completions`

### 示例

```bash
curl http://localhost:8888/proxy/openai/openai/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "hi"}],
    "stream": false
  }'
```

## 数据存储

默认使用 SQLite，数据库文件位于 `~/.aiapi/aiapi.db`。

### 核心数据表

- `providers`：上游 Provider 配置
- `api_keys`：调用方 API Key
- `usage_records`：Token 用量记录
- `request_logs`：请求日志与错误信息

可通过 `sql/sqlite.sql` 查看完整 DDL。

## 日志与调试

### 应用日志

默认输出到标准输出与 `~/.aiapi/logs/app.log`，支持按大小/时间轮转。

### pprof

启动时添加 `--pprof` 参数，然后访问：

```bash
# 查看性能概览
curl http://localhost:8888/debug/pprof

# 下载 30 秒 CPU profile
go tool pprof http://localhost:8888/debug/pprof/profile

# 查看堆内存
go tool pprof http://localhost:8888/debug/pprof/heap
```

## 开发扩展

- 新增上游响应解析逻辑：参考 `parser/openai.go` 与 `parser/anthropic.go`
- 新增 Pipeline 处理步骤：参考 `proxy/handler/` 目录
- 新增管理接口：参考 `manager/provider.go`

> 注意：当前项目仅代理 OpenAI 兼容格式的请求。若未来需要支持非 OpenAI 的客户端协议，需新增 `parser/` 实现并在 `framework/echo.go` 或 `proxy/direct.go` 中做必要适配。

详细的开发规范与约定请查看 [`AGENTS.md`](./AGENTS.md)。

## 贡献指南

1. Fork 本仓库
2. 在 `main` 之外创建功能分支
3. 保持 `AGENTS.md` 与代码同步
4. 提交 PR 并描述变更点

## 许可证

[MIT](LICENSE)（待补充）

# AGENTS.md — 代理链路

本文件补充仓库级规则 [`../AGENTS.md`](../AGENTS.md)，只写 `proxy/` 特有的约定。

## Pipeline 约定

- 用 Pipeline 编排请求处理流程，**每个 handler 只负责一个步骤**；`proxy/direct.go` 只组装，不实现业务。
- handler 成功时直接返回；失败时只设置 `ctx.Err` 与 `ctx.Code`，由 Pipeline 统一输出错误响应。
- **禁止在 handler 中检查 `ctx.Err` 决定是否执行**（Pipeline 已保证失败后不再执行；全仓只有 `proxy/pipeline.go`、`proxy/direct.go` 读它）。
- `Failed` 时**不写响应体**：副作用留给框架层。
- 不可中断的收尾操作（如写日志）注册为 `AddFinally`。
- handler 应能独立于 Echo 与真实数据库做单元测试。

现有两条链路（`proxy/direct.go`）：

```
转发   ParseRequest → AuthKey → AuthModel → BudgetCheck → LoadConfig → RewriteModel → Forward → ParseUsage → Record  (+ Finally: Log)
元数据 ParseRequest → AuthKey → ListModels
```

`AuthKey`（Key/用户校验，所有链路共用）与 `AuthModel`（模型定价 + 白名单，仅转发链路）是两个独立 handler，新链路按需取用。

## 新增 handler

1. 在 `proxy/handler/` 新增文件，签名统一 `func Xxx(ctx *types.Context)`。
2. 按是否需要收尾执行，在 `proxy/direct.go` 选 `AddLast` / `AddFinally` 注册，顺序放在逻辑合理的位置。
3. 失败路径要有单测（同包 `package handler`，参考 `proxy/handler/*_test.go`）。

## 路由与元数据端点

- 入口区分靠**路由注册**，不在运行时按 path 判定：echo 静态段优先于通配符 `*`，`GET /:provider/:format/v1/models` → `proxy.HandleModels`，其余 → `proxy.Handle`。
- 元数据端点（不依赖上游）：组装自己的 Pipeline，**不挂 `Log`、不经 `Forward`、不计费、不写 `request_logs`**；具体路由没有通配参数，由 router 适配层补写 `req.Path` 供日志使用。
- 错误日志打印收敛在共享的 `logErrors`，不要各写一份。

## 转发的硬约束

- `Forward` 只做 HTTP 传输：构造请求、复制安全响应头、写回客户端、缓存原始响应；**不做协议解析、用量统计或计费**（后两者分别是 `ParseUsage` 与 `Record` 的职责）。
- **只使用 `providers.config.headers` 作为上游请求头**，客户端请求头不转发（含 `Content-Type`）；客户端头仅用于提取 Key 与写日志。
- 上游 URL = `config.domain` + `/` + 通配路径，因此 `domain` 不能以 `/` 结尾。
- **不跟随上游 3xx**（`CheckRedirect` 返回 `http.ErrUseLastResponse`）：标准库只剥离 `Authorization` 等四个头，`x-api-key` 这类自定义凭证头会被转发到重定向目标。理由与代价见 [`../docs/decisions/2026-09-24-refuse-upstream-redirects.md`](../docs/decisions/2026-09-24-refuse-upstream-redirects.md)，有回归测试兜底。
- 不得给上游 client 设总超时，也不得给 HTTP Server 设 `WriteTimeout`——长 SSE 会被误杀（根目录 RED-13）。

## 响应与错误

- 上游响应原样透传（含状态码与 SSE 事件）；proxy 自己产生的错误是 JSON `{"error":{"message":"..."}}` + 对应 HTTP 状态。
- 上游 4xx/5xx 原样透传，**不计费、不写 `usage_records`**，但仍写 `request_logs`。
- proxy 链路对外错误消息用**英文**（网关客户端的错误契约）；错误码集中在 `proxy/types/bizcode.go`，新增码需编号唯一。

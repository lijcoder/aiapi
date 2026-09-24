# 上游 3xx 不跟随，原样透传给客户端

- **状态**：已实施
- **日期**：2026-09-24
- **相关**：[`../../proxy/handler/forward.go`](../../proxy/handler/forward.go)、[`../../proxy/handler/forward_redirect_test.go`](../../proxy/handler/forward_redirect_test.go)

## 背景

转发用的 `upstreamClient` 原先没有配置 `CheckRedirect`，因此使用标准库默认行为：最多跟随 10 次重定向。

标准库只在**跨域**重定向时剥离 `Authorization`、`Www-Authenticate`、`Cookie`、`Cookie2`（`net/http/client.go` 的 `shouldCopyHeaderOnRedirect`）。而本项目的上游凭证来自 `providers.config.headers`，常见形态是 Anthropic 的 `x-api-key`、Azure 风格的 `api-key`——**这些自定义头不在剥离名单里**。实测（跨域 302）：

```
Authorization  ""                 ✅ 已被剥离
x-api-key      "sk-ant-SECRET"    ❌ 泄漏到重定向目标
api-key        "azure-SECRET"     ❌ 泄漏到重定向目标
```

而 Provider 常配第三方中转服务，一次 302 足以交出上游密钥。

## 决定

`upstreamClient.CheckRedirect` 返回 `http.ErrUseLastResponse`：不发起第二次请求，3xx 与响应体原样透传给客户端。回归测试同时断言**重定向目标一次都没被访问**与状态码原样透传；另有配置存在性测试防止后人删掉该字段。

## 否决的方案

- **跟随重定向但清空凭证头**：上游 302 到自身另一路径是合法场景，但"清空后重试"会让上游收到无凭证请求并返回 401，行为难解释；且需要自己实现跨域判定与头筛选，等于重写标准库逻辑。
- **只对自定义头做剥离名单**：名单永远不全（未来接入的协议可能用 `x-goog-api-key` 等），漏一个就等于泄漏。
- **限制重定向目标白名单**：需要配置项与运维理解成本，收益不明确——LLM API 正常不会返回 3xx。

## 代价与后果

- 若某个上游确实靠 3xx 跳转（例如域名迁移）才能工作，代理会把它当作普通响应返回给客户端；这种情况应在 Provider 配置里直接写最终地址。
- 该策略覆盖所有 Provider，是"凭证-bearing 请求"的统一规则；新增上游协议时不需要单独处理。

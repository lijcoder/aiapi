# 模型计费配置

> 模型的 `pricing_config` 是版本化 JSON（当前 `version: 2`），在管理台「模型管理」中图形化编辑。
> 计费的实现与并发语义见 [`architecture.md`](architecture.md) §6；用量记录字段见 [`../sql/AGENTS.md`](../sql/AGENTS.md)。

## 规则结构

单价单位为**元/百万 Token**。判定顺序：命中多条规则时取 `priority` 最大的一条；都没命中则用 `default_price`。未填写 `timezone` 时默认 `Asia/Shanghai`，时间按**请求开始时刻**匹配。

```json
{
  "version": 2,
  "timezone": "Asia/Shanghai",
  "default_price": {"input_cache_hit": 0.5, "input_cache_miss": 2, "output": 8},
  "rules": [
    {
      "name": "工作日高峰大请求",
      "enabled": true,
      "priority": 100,
      "when": {
        "op": "and",
        "children": [
          {"type": "weekday", "values": [1, 2, 3, 4, 5]},
          {"op": "or", "children": [
            {"type": "time_range", "start": "09:00", "end": "12:00"},
            {"type": "time_range", "start": "14:00", "end": "18:00"}
          ]},
          {"type": "total_tokens", "operator": "gte", "value": 10000}
        ]
      },
      "price": {"input_cache_hit": 0.8, "input_cache_miss": 3, "output": 12}
    }
  ]
}
```

## 条件

`when` 是可嵌套的 `and` / `or` 条件树；同一模型的规则 `name` 与 `priority` 必须唯一。

| 叶子条件 | 取值 |
|----------|------|
| `weekday` | ISO 星期：周一为 `1`，周日为 `7`；多值内部为 OR |
| `time_range` | `HH:MM`，左闭右开；不支持跨天 |
| `month_day` | `MM-DD`，每年重复匹配 |
| `total_tokens` | `operator` 为 `gt` / `gte` / `lt` / `lte` |

配置按**严格模式**解析：未知字段、`version` 非 2、规则 `name` 或 `priority` 重复、价格为负或非有限数都会被拒绝，管理台保存时返回具体原因。读取 v1 配置会自动转换为 v2，保存后统一为 v2。

## 计费快照

每笔 `usage_records` 保存 `pricing_snapshot`：命中的规则名、条件、实际单价、**整份 `pricing_config`**、时区与请求开始时间（`yyyy-MM-dd HH:mm:ss`）。因此事后改价不会影响历史费用的审计。

## 用量口径

按 OpenAI 语义统一：`input_cache_hit` 为命中缓存的输入，`input_cache_miss` 为其余输入。

| 协议 | 口径 |
|------|------|
| Anthropic | `cache_creation_input_tokens` 与 `cache_read_input_tokens` 并入完整输入量（缓存读取按命中价、缓存创建按未命中价计费）；`output_tokens_details.thinking_tokens` 计入 `reasoning_tokens`（属输出、不重复计费）；流式用量取 `message_delta` 的累计值 |
| OpenAI / Responses | 优先取上游 `usage`，缺省回退 输入 + 输出；流式需要带 `usage` 的 chunk |

**未配置或配置非法的模型不会被转发**（对外返回 404 类错误，消息 `model pricing is not configured`）。

## 已知待修

非流式与流式的 `total_tokens` 回退口径不一致（`parser/openai.go` 非流式不回退），而该字段参与 `total_tokens` 条件匹配，可能命中错误规则。详见 [`../TODO.md`](../TODO.md) 第 6 项。

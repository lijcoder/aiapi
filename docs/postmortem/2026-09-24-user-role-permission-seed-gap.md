# 普通用户的「模型列表」与「查看 Key 明文」返回 403

- **日期**：2026-09-24
- **影响**：`user` 角色调用 `/manager/models` 与 `/manager/apikeys/reveal/self` 被拒（403）。「模型列表」菜单已分配给该角色，页面可见但列表加载失败；API Key 页的「查看明文」按钮不可用。admin 角色不受影响。
- **状态**：已修复（补种子 + 加门禁）

## 概要

接口写了、路由注册了、菜单也授权了，但 `role_permission` 种子漏了这两条路径。权限判定是**精确路径匹配**，种子缺失即 403。逃逸的根本原因不是漏写，而是"路由注册"与"权限种子"是两份必须人工保持一致的清单，**中间没有任何机械校验**：文档里有这条规则，但规则只是愿望。

## 时间线

1. 重构文档时逐条核对 `manager/router/router.go` 与 `sql/init-data.sql`，发现两边条数对不上。
2. 确认两条路由已在 router 注册（`/manager/models`、`/manager/apikeys/reveal/self`），但种子里的 user 角色权限列表没有它们。
3. 另发现前端确实会调用：`frontend/src/views/ApiKeys.vue` 的「查看」按钮走 `/apikeys/reveal/self`，说明不是"预留但未使用"的接口。
4. 补种子，并加上 `gate_arch_test.go`（当时名为 `arch_test.go`）的双向校验门禁。

## 根因

`manager/middleware.Require` 按当前请求路径在 `role_permission` 里精确匹配，`value='*'` 之外没有任何推导逻辑。因此**新增路由 ≠ 新增权限**，后者要单独维护一份 SQL 种子。而种子只在初始化/迁移时执行一次（`INSERT OR IGNORE`），存量库还需要重跑。

## 为什么没被拦住

| 安全网 | 为什么失效 |
|--------|------------|
| 单元测试 | `manager/test/` 只覆盖了 `base.Wrap` 与权限判定的机制，没有任何测试检查"路由集合 ↔ 种子集合" |
| 文档规则 | 「新增非超管 manager 接口必须同步 `role_permission`」确实写在操作手册里，但没有执行者——规则靠人读、人记 |
| 菜单授权 | 反而增加了误判：菜单已授权给 user 角色，人工检查时容易认为"权限配过了" |
| CI / 门禁 | 仓库当时完全没有 CI 与自动化检查 |
| 使用者反馈 | 内部部署、admin 单账号使用，普通用户路径几乎没人走 |

## 护栏

- `gate_arch_test.go` 的 `TestGateRoutesHavePermissionSeed`（当时分别为 `arch_test.go` 与 `TestRoutesHavePermissionSeed`）：**双向**校验——所有 `/self` 路由必须在 user 角色种子里；种子里出现的路径必须是已注册路由（防改名后留下死权限）。已用变异验证：删掉一行种子，测试立即失败。
- `sql/init-data.sql` 在权限块上方写明义务，并指出门禁位置。
- 收尾清单进入 [`../../manager/AGENTS.md`](../../manager/AGENTS.md)。

**护栏的已知边界**：非 `/self` 但面向普通用户的路由（如 `/manager/models`）无法从命名推导，门禁覆盖不到，仍需人工判断——这条限制写在门禁代码注释与 `manager/AGENTS.md` 里，避免后来者误以为已被全覆盖。

# AGENTS.md — 后台管理接口

本文件补充仓库级规则 [`../AGENTS.md`](../AGENTS.md)，只写 `manager/` 特有的约定。
安全与权限模型见 [`../docs/security.md`](../docs/security.md)；前端页面见 [`../frontend/AGENTS.md`](../frontend/AGENTS.md)。

## 接口形态（硬性）

- **所有接口一律 `POST` + JSON body**，不用路径参数、不用 GET（查询类也用 POST）。
- 响应统一 envelope `{code, msg, data}`：成功 `code=0`；失败 code 非 0，HTTP 状态由 `manager/base/bizcode.go` 的 `BizCode` 决定。handler **不要自己构造响应**，交给 `base.Wrap` / `base.Ok` / `base.Fail`。

## 业务函数签名

```go
func Xxx(ctx context.Context, req *XxxReq) (*XxxResp, *base.BizError)
```

- 参数只能由 `echo.Context` / `context.Context` / **至多一个**请求结构体指针组成；返回值必须是 `(值, *base.BizError)`。
- **违规在启动期 panic**（`manager/base/wrap.go`），报错只出现在启动日志里——服务直接起不来。
- 空 body 按零值处理，不报错；参数校验放在业务函数里（用 `base.ErrBadReq`）。
- 登录态从 `context.Context` 取（`base.CurrentUser`）。

## 分层判定

| 场景 | 放哪 |
|------|------|
| 参数校验、组装响应 | `manager/handler/` |
| 单表读写（`store.C().Xxx().List(...)`） | 允许 handler 直接调 store（现状主流） |
| 跨表校验 / 业务不变量（如"有限额 Key 的额度总和 ≤ 用户余额"） | `service/` |
| 一次写多张表、事务编排 | `service/`（`store.C().T(fn)`） |
| 被多个 handler 复用的逻辑、哈希/加密等业务工具 | `service/` |

## 路由与中间件

1. 在 `manager/handler/` 按业务领域新增文件（如 `user.go`、`charge.go`）；同一领域的普通用户版与超管版合并在同一文件，不单独起 `*_admin.go`。
2. 在 `manager/router/router.go` 的 `Register(g *echo.Group)` 中用 `base.Wrap(handler.Xxx)` 注册；`/manager` 根路径由 `framework/echo.go` 直写。
3. 中间件挂载：`login` / `login/2fa` / `refresh` 不挂 Auth；`logout` 挂 Auth 不挂 Require；其余挂 Auth + Require。

## 命名约定

- 列表查询统一 `/list` 后缀（`/users/list`、`/providers/list`、`/models/list`、`/recharge/records/list`）。
- 普通用户自助接口用 `/self` 后缀，超管接口不加 `/self`。
- 存量反例（新接口不要照抄）：`/manager/models`（用户模型列表）、`/manager/apikeys/list/self`（`list` 成了中缀）、`/manager/recharge/records`（自助流水无 `/self`）。
- self / admin 合并模式：同一业务的自助版与超管版**合并为一个通用函数**，self 入口只设 `req.UserID = cur.ID` 后委托（参考 `RechargeSelf` → `Recharge`）。现状只有 recharge 按此落地，`apikey` 仍是两份近似实现，属待收敛项。

## 分页

- 类型分层：`manager/base` 定义 `PageReq`（内嵌到 Req）与 `PageResult[T]`（出参）；`store` 只暴露 `PageContext`。handler 不直接 import `store/base`。
- 非事务单次分页：链式调用，Session 用完即弃，不用 `ClearPage`：

```go
pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
items, err := store.C().SetPage(pc).Charge().List(...)
return &base.PageResult[T]{Items: items, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
```

- `page` 1-based（`<1` 按 1），`page_size` 默认 20、上限 100（`manager/base/page.go`）。未 `SetPage` 时 `Select` 退化为普通查询。
- 事务内多次查询需要显式 `SetPage` / `ClearPage` 控制作用范围——当前代码**没有任何调用点**，属预留能力，不要为它做额外设计。

## 新增接口收尾清单

- [ ] handler + service/store + router 注册
- [ ] **`sql/init-data.sql` 的 `role_permission`**：非超管接口必须给 user 角色加一条，路径与路由完整路径一致（含 `/manager` 前缀），否则 403
- [ ] 涉及新页面时同步 `menus` + `role_menus` + 前端路由（见 [`../frontend/AGENTS.md`](../frontend/AGENTS.md)）
- [ ] 错误用 `base.ErrBadReq` / `base.ErrNotFound`（中文消息）/ `base.ErrInternal`；**不新增业务码**
- [ ] 失败路径（非法参数、越权、资源不存在）有单测
- [ ] `README.md` 接口说明与 `CHANGELOG.md` 同步

> 权限种子有门禁兜底：`gate_arch_test.go` 的 `TestGateRoutesHavePermissionSeed` 校验所有 `/self` 路由都已授权、且种子里没有已删除路由的死权限。
> **覆盖边界**：非 `/self` 但面向普通用户的路由（如 `/manager/models`）推导不出来，必须人工判断。历史事故见 [`../docs/postmortem/2026-09-24-user-role-permission-seed-gap.md`](../docs/postmortem/2026-09-24-user-role-permission-seed-gap.md)。

# 管理接口参考

> `/manager` 下的全部接口。认证与权限模型见 [`security.md`](security.md)，接口开发规范见 [`../manager/AGENTS.md`](../manager/AGENTS.md)。
> 面向使用者的快速上手在 [`../README.md`](../README.md)。
>
> 本文件是目录型参考：**接口表与 `manager/router/router.go` 双向校验**（`gate_arch_test.go` 的 `TestGateAPIDocCoversAllRoutes`）——
> 新增或改名路由后忘了同步这里会直接测试失败，所以它不设字节预算，只管跟着接口走。

## 约定

- **所有接口都是 `POST`，参数放 JSON body**，不用路径参数、不用 GET（查询类也用 POST）。
- 响应统一为 `{code, msg, data}`：成功 `code=0`；失败 `code` 非 0，HTTP 状态由业务码决定。
- 标注「分页」的接口接受 `page`（1-based）与 `page_size`（默认 20，上限 100），返回 `{items, total, page, page_size}`。
- 除下表标注「免登录」外，其余接口都需要 `Authorization: Bearer <access_token>`。
- 自助接口（`/self` 后缀）只能操作自己的数据；不加 `/self` 的是超管接口，需要 admin 角色。

## 免登录

| 路径 | 说明 |
|------|------|
| `POST /manager/login` | 登录。body `{account, password}`；返回 `{access_token, expires_in}` 并写 refresh cookie；开启 2FA 时改返回 `{need_2fa: true, pending_ticket, expires_in}` |
| `POST /manager/login/2fa` | 登录第二步。body `{pending_ticket, code}` |
| `POST /manager/refresh` | 刷新 access token 并轮换 refresh token（凭 cookie） |

## 自助接口（需登录 + 授权）

| 路径 | 说明 |
|------|------|
| `POST /manager/logout` | 登出（需登录态，不需接口授权） |
| `POST /manager/self` | 当前用户信息 + 菜单树 + 权限列表 |
| `POST /manager/recharge/records/self` | 自己的充值流水（分页） |
| `POST /manager/models` | 可用模型列表（分页，body `{provider, model}` 模糊搜索；不含上游模型名） |
| `POST /manager/apikeys/list/self` | 自己的 API Key 列表（分页） |
| `POST /manager/apikeys/create/self` | 创建 API Key。body `{name, budget, unlimited}`，**明文 key 仅本次返回** |
| `POST /manager/apikeys/toggle/self` | 启用/禁用。body `{id}` |
| `POST /manager/apikeys/delete/self` | 删除。body `{id}` |
| `POST /manager/apikeys/rename/self` | 改名。body `{id, name}` |
| `POST /manager/apikeys/budget/self` | 改额度。body `{id, budget, unlimited}`；有限额 Key 的额度之和不得超过账户余额 |
| `POST /manager/apikeys/reveal/self` | 查看 Key 明文。body `{id}`，返回 `{key, revealable}`（旧数据不可还原） |
| `POST /manager/apikeys/models/get/self` | 查 Key 的模型策略。body `{api_key_id}` |
| `POST /manager/apikeys/models/set/self` | 设置模型策略。body `{api_key_id, model_policy: all\|whitelist, model_ids}` |
| `POST /manager/usage/stats/self` | 用量统计。body `{mode, start_date, end_date, api_key_id?, model?, provider?, group_by?}` |
| `POST /manager/usage/filters/self` | 筛选选项（用过的 Key / 模型 / Provider） |
| `POST /manager/profile/update/self` | 改资料。body `{name, email}` |
| `POST /manager/profile/password/self` | 改密码。body `{old_password, new_password}` |
| `POST /manager/2fa/setup/self` | 生成 TOTP 密钥与二维码（不落库），返回 setup 票据 |
| `POST /manager/2fa/confirm/self` | 校验首个验证码并生效。body `{setup_ticket, code}` |
| `POST /manager/2fa/disable/self` | 关闭 2FA。body `{password}` |

## 超管接口（需 admin 角色）

| 路径 | 说明 |
|------|------|
| `POST /manager/users/list` | 用户列表（分页，body `{keyword}` 按姓名/账号搜索） |
| `POST /manager/users/get` | 用户详情。body `{id}` |
| `POST /manager/users/create` | 创建用户。body `{name, account, password, budget, unlimited}` |
| `POST /manager/users/update` | 编辑用户（账号不可改）。body `{id, name, budget, unlimited}` |
| `POST /manager/users/toggle` | 启停用户。body `{id}` |
| `POST /manager/users/reset-password` | 重置密码。body `{id, password}` |
| `POST /manager/users/assign-roles` | 分配角色。body `{id, role_ids}` |
| `POST /manager/roles/list` | 角色列表 |
| `POST /manager/providers/list` | Provider 列表（分页） |
| `POST /manager/providers/create` | 新增 Provider。body `{type, domain, headers}` |
| `POST /manager/providers/update` | 编辑 Provider（`type` 不可改）。body `{type, domain, headers}` |
| `POST /manager/providers/toggle` | 启停 Provider。body `{type}` |
| `POST /manager/models/list` | 全部模型（分页，含 `provider_model` 与模态支持） |
| `POST /manager/models/create` | 新增模型。body `{provider, model, provider_model, pricing_config, max_context_tokens, max_completion_tokens, supports_text, supports_image, supports_video}` |
| `POST /manager/models/update` | 编辑模型（`provider`/`model` 不可改）。body `{id, provider_model, pricing_config, max_context_tokens, max_completion_tokens, supports_*}` |
| `POST /manager/models/delete` | 删除模型。body `{id}` |
| `POST /manager/apikeys/list` | 指定用户的 Key 列表（分页）。body `{user_id}` |
| `POST /manager/apikeys/toggle` | 启停指定 Key。body `{id}` |
| `POST /manager/apikeys/delete` | 删除指定 Key。body `{id}` |
| `POST /manager/apikeys/rename` | 改名。body `{id, name}` |
| `POST /manager/apikeys/budget` | 改额度。body `{id, budget, unlimited}` |
| `POST /manager/apikeys/reveal` | 查看 Key 明文。body `{id}` |
| `POST /manager/apikeys/models/get` | 查模型策略。body `{api_key_id}` |
| `POST /manager/apikeys/models/set` | 设置模型策略。body `{api_key_id, model_policy, model_ids}` |
| `POST /manager/recharge` | 给指定用户充值。body `{userId, amount, remark}` |
| `POST /manager/recharge/records` | 指定用户充值流水（分页）。body `{userId}` |
| `POST /manager/recharge/records/list` | 全平台充值流水（分页，body `{keyword}`） |
| `POST /manager/usage/stats` | 全局用量统计。body `{mode, start_date, end_date, user_id?, api_key_id?, model?, provider?, group_by?}` |
| `POST /manager/usage/filters` | 全局筛选选项（含用户列表） |
| `POST /manager/dashboard` | 仪表盘：汇总指标（用户数/Key 数/今日请求/今日费用）+ 近 7 天费用趋势 |
| `POST /manager/logs` | 最近 100 行应用日志（从文件尾倒序） |

`admin` 角色只需一条 `role_permission('API', '*', '*')` 即可访问全部超管接口。

## 调用示例

```bash
# 登录（-c 保存 refresh cookie）
curl -c cookie.jar -X POST http://localhost:8888/manager/login \
  -H 'Content-Type: application/json' \
  -d '{"account":"admin","password":"你的密码"}'
# → {"code":0,"msg":"success","data":{"access_token":"eyJ...","expires_in":900}}

# access 过期后刷新
curl -b cookie.jar -X POST http://localhost:8888/manager/refresh

# 业务请求带 access token（cookie 只用于 refresh）
curl -b cookie.jar -X POST http://localhost:8888/manager/recharge \
  -H "Authorization: Bearer <access_token>" \
  -H 'Content-Type: application/json' \
  -d '{"userId":2,"amount":10,"remark":"月度充值"}'
```

## 相关

- 登录态、2FA、密钥、权限模型：[`security.md`](security.md)
- 管理台页面与菜单：[`../frontend/AGENTS.md`](../frontend/AGENTS.md)
- 新增接口的规范与收尾清单：[`../manager/AGENTS.md`](../manager/AGENTS.md)

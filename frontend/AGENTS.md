# AGENTS.md — 管理台前端

本文件补充仓库级规则 [`../AGENTS.md`](../AGENTS.md)，只写 `frontend/` 特有的约定。
跨包的分层与红线不在重复；文档该写在哪见 [`../docs/AGENTS.md`](../docs/AGENTS.md)。

`frontend/` 是 Vue 3 + Vite + Naive UI 管理台，构建产物通过 Go `embed` 嵌入二进制。

## 命令

| 目的 | 命令 |
|------|------|
| 开发 | `make dev-ui`（端口 3000，Vite 只代理 `/manager` 到 8887，**不代理 `/proxy`**） |
| 发布 | `make build-all`（先 `npm run build` 再 `go build`） |

改前端源码后不跑 `make build-all`，二进制里仍是旧 UI。注意仓库只跟踪 `frontend/dist/index.html`，`assets/` 不入库——全新 clone 直接 `make build` 会白屏。

## 新增页面（四件事都要做，缺一个页面就不可达或被 403）

1. **组件**：在 `frontend/src/views/` 下写 Vue 单文件组件（Naive UI 的 `n-card`/`n-button`/`n-data-table`/`n-modal` 等；图标库 `@vicons/ionicons5`）。超管页面放 `frontend/src/views/admin/`（路由在 `/admin/*` 下），import 路径多一级 `../../`。
2. **路由**：在 `frontend/src/router/index.js` 加入 Home 路由的 `children`。
3. **菜单种子**：在 `sql/init-data.sql` 的 `menus` 表加一行，并在 `role_menus` 给相应角色授权。
4. **接口权限**：新页面调用的每个非超管接口，都要在 `sql/init-data.sql` 的 `role_permission` 里给 user 角色加路径授权（见 [`../manager/AGENTS.md`](../manager/AGENTS.md) 的收尾清单；`/self` 路由有门禁兜底，非 `/self` 的需人工判断）。

> ⚠️ 侧栏菜单**不是**在前端硬编码的：`Home.vue` 用 `v-for="menu in menus"` 渲染后端 `/manager/self` 返回的菜单树。往 `<nav>` 里手写 `<router-link>` 不会生效（历史文档中的这条说明已作废）。

菜单形态：有 `children` 为分组容器（点击展开/收起）；无 `children` 且 `path` 非空为直接跳转；`path` 为空为纯标题。

前端路由守卫只校验登录态、**不校验角色**：普通用户直接输 `/admin/*` 能渲染页面，但页面内接口会 403。不要在守卫里补角色判断，页面可见性由后端菜单树决定。

## 接口调用

在 `frontend/src/api/index.js` 封装 `request(PATH, body)`：所有接口都是 `POST` + JSON body，响应统一 `{code, msg, data}`，以 `data.code` 真值判定失败并解包 `data`；业务码 `1016`（access 过期）触发 `/manager/refresh` 后重试。业务代码不要绕过封装直接 `fetch`。

## 公共模块（优先复用，不要重写）

| 模块 | 路径 | 职责 |
|------|------|------|
| 工具函数 | `frontend/src/utils.js` | `fix4`（金额格式化，4 位小数去尾 0）、`formatTime`（时间格式化） |
| 图表选项 | `frontend/src/charts.js` | ECharts option 构建器、`metricConfig` 指标配置 |
| 图表生命周期 | `frontend/src/composables/useChart.js` | ECharts init/resize/dispose，`watch` 数据自动重绘 |
| 分页逻辑 | `frontend/src/composables/usePagination.js` | `n-data-table` 远程分页通用逻辑（`pagination`/`onPage`/`onPageSize`/`resetAndLoad`），含 `showTotal` 与 `showSizePicker` |

## 表格风格统一

所有 `n-data-table` 保持一致：

- `:bordered="false"` `size="small"`——无边框紧凑风格
- 列宽超出容器时加 `:scroll-x="总宽"` 防溢出
- 金额列统一 `fix4()`；时间列统一 `formatTime()` + `ellipsis: { tooltip: true }` 防换行
- 空数据兜底 `value = (await xxx()) || []`
- 删除/禁用等危险操作使用 `useDialog` 二次确认，不用 `window.confirm`

## 超管特权

`role_permission` 里 `value='*'` 为超管通配权限，admin 角色只需 `('API', '*', '*')` 一条即可访问全部超管接口；普通用户按接口路径精确授权（最小权限原则）。

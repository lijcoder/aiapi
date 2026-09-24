---
name: aiapi-release
description: "Use when preparing or reviewing a release, upgrade, or schema change in the aiapi repo (github.com/lijcoder/aiapi). Walks the cross-package checklist that no single package's AGENTS.md owns — gates, schema version, data migrations, docs/CHANGELOG sync, and the frontend embed."
---

# aiapi 发布与升级

**这是跨包流程**：每一步落在不同的目录，任何单个包的 `AGENTS.md` 都不覆盖全流程。
规则本体见根 [`AGENTS.md`](../../../AGENTS.md)；数据库部分见 [`sql/AGENTS.md`](../../../sql/AGENTS.md) 与 [`sql/migrations/README.md`](../../../sql/migrations/README.md)。

## 1. 改完代码后的门禁

```bash
make hooks    # 每个 clone 只跑一次：启用 .githooks/pre-commit，提交前自动跑 make check
make check    # gofmt 检查 + go vet + go test（含架构/权限种子/文档/schema 版本门禁）
make build
```

改了 `frontend/` 还要 `make build-all`——仓库只跟踪 `frontend/dist/index.html`，前端产物不入库，不重新构建的话二进制里的管理台是旧的。

## 2. 涉及数据库结构时

1. 把新结构写进 `sql/sqlite.sql`（不要在里面追加历史迁移说明）。
2. 新建 `sql/migrations/v<N>-<主题>.sql`，只写增量语句。
3. `constant.SchemaVersion` 加一，并同步 `sql/sqlite.sql` 里 `schema_meta` 的版本行（门禁会校验两者一致）。
4. 在 `sql/migrations/README.md` 的版本表登记：适用条件、是否破坏性、需要人工介入的部分（如外部计算哈希、重新配置计费）。
5. 同步 `store/model/models.go`、`store/schema_test.go`。

**升级路径必须在发布说明里写清楚**：老库要跑哪些迁移、跑完用什么语句标记版本。版本只能递增，降级会丢列丢数据。

## 3. 涉及接口或页面时

- 新增非超管管理接口 → 必须补 `sql/init-data.sql` 的 `role_permission`（`/self` 路由有门禁兜底，非 `/self` 的需人工判断）。
- 新增页面 → 同步 `menus` 种子 + `role_menus` 授权 + `frontend/src/router/index.js`。
- 清单在 [`manager/AGENTS.md`](../../../manager/AGENTS.md) 与 [`frontend/AGENTS.md`](../../../frontend/AGENTS.md)。

## 4. 文档同步（按根 AGENTS.md RED-02 判定）

| 变化 | 要更新 |
|------|--------|
| 用户可见的功能、接口、部署方式 | `README.md` |
| 开发规则、架构约定、项目边界 | `AGENTS.md` 或对应包的 `AGENTS.md` |
| 破坏性变更、模块拆分、目录调整 | `CHANGELOG.md`（写清影响面与升级步骤） |
| 只影响内部的移动/重命名/重构 | 不需要单独记 CHANGELOG |
| 定了新设计或推翻了旧决定 | `docs/decisions/` 新写一篇 |
| 出过一次"没被拦住"的故障 | `docs/postmortem/` 写一篇，并把教训落成门禁 |

## 5. 提交

- 提交信息用 Conventional Commits：`<type>(<scope>): <中文描述>`，破坏性变更加 `!`（如 `feat(proxy)!:`）。
- 破坏性变更要同时更新 `README.md` 的升级说明与 `CHANGELOG.md`。
- 提交前再跑一次 `make check`；门禁失败不要用跳过的方式绕过（改了 schema 却忘了同步常量，正是门禁要拦的情况）。装了钩子后这一步会自动发生；`git commit --no-verify` 能跳过它，用一次等于放弃一次门禁。

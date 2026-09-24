# 数据库 schema 用单一版本常量 + 迁移台账，启动时校验

- **状态**：已实施
- **日期**：2026-09-24
- **相关**：[`../../constant/constant.go`](../../constant/constant.go)、[`../../store/driver/schema.go`](../../store/driver/schema.go)、[`../../sql/migrations/README.md`](../../sql/migrations/README.md)、[`../../sql/sqlite.sql`](../../sql/sqlite.sql)

## 背景

应用不自动建库也不自动迁移，所有结构变更以注释形式散落在 `sql/sqlite.sql` 里。由此产生三个问题：

1. **无法判定线上库处于哪一版**。升级时要人工逐条比对注释与表结构，漏执行一条只有在运行到相关功能时才暴露。
2. **DDL 与迁移历史混在一起**。`sql/sqlite.sql` 既是"新库的完整 schema"，又是"旧库的迁移手册"，两件事的读者和生命周期完全不同。
3. **版本没有权威**。没有任何地方能回答"当前是哪一版"。

## 决定

- 版本权威只有一个：`constant.SchemaVersion`（Go 常量）。
- `sql/sqlite.sql` 只描述**当前完整 schema**，开头建 `schema_meta` 单行元数据表并写入版本行；迁移历史移到 `sql/migrations/README.md` 台账。
- 启动时 `driver.CheckSchemaVersion` 读取 `schema_meta` 的版本行与常量比对：库旧 → 拒绝启动并指出要执行哪些迁移；库新 → 拒绝启动（二进制过旧，降级有风险）；**没有版本标记 → 允许启动但告警**。
- 门禁：`store` 包的测试把 `sql/sqlite.sql` 的 pragma 与常量绑定，改一处漏另一处会失败。

## 否决的方案

- **应用自动迁移**：需要一个迁移框架与版本锁，且自动 DDL 在有存量数据时风险高（部分迁移需外部计算哈希、需重新配置计费），与本项目的"部署简单、数据可控"取向不符。
- **用 SQLite 专有的 `PRAGMA user_version` 记录版本**（第一版实现）：改动最小、不新增表，但本项目规划支持 MySQL / PostgreSQL，而 `user_version` 只有 SQLite 有（MySQL 没有等价物，PostgreSQL 也没有）。选它等于把版本机制绑死在当前驱动上，将来迁移数据库时要再改一次协议。改用 `schema_meta` 元数据表：三个目标库都支持，Go 侧只有一条 `SELECT`，读取路径不按驱动分支。
- **靠注释里的迁移清单**（现状）：无法机械判定版本，正是要解决的问题。
- **未标记（0）直接拒绝启动**：会把现有部署全部挡在门外；这类库可能已经是最新结构，只是没标记。选择告警 + 由运维核对台账后手工标记。
- **用文件名/目录排序推导版本**：多一个需要维护的隐式约定，不如一个显式常量。

## 代价与后果

- 改 schema 要动**三处**（`sql/sqlite.sql`、`constant.SchemaVersion`、`sql/migrations/` + 台账），漏一处会被门禁拦住，但确实比只改 DDL 麻烦。
- 首次升级到本版本的老库需要手工建 `schema_meta` 并写入版本行（台账里有可复制的语句）。
- `schema_meta` 表本身也属于 schema：读取失败时无法区分"表不存在（历史库）"与"连接故障"，代码用一次可移植的探活（`SELECT 1`）来区分——探活也失败就硬失败，否则按未标记放行。这是不解析各驱动错误码（SQLite/MySQL/PostgreSQL 各不同）的折中。
- 版本只能单调递增：不支持降级，因为降级意味着丢失列与数据。

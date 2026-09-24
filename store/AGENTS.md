# AGENTS.md — 数据持久层

本文件补充仓库级规则 [`../AGENTS.md`](../AGENTS.md)，只写 `store/` 特有的约定。
DDL 规范与迁移在 [`../sql/AGENTS.md`](../sql/AGENTS.md)。

`store/` 是纯 SQL 包装层：**单表读写 + 只读 JOIN/聚合**，不写跨表写操作、不写事务体、不写业务判断、不放纯函数业务工具。这些有门禁校验（根目录 `gate_arch_test.go`），越界会编译期失败。

## 边界判定

| 场景 | 放哪 |
|------|------|
| 单表 `Get/Select/Exec` | `store/` |
| 只读 JOIN / 聚合查询 | `store/`（现状如此，允许） |
| 一次写多张表 / 事务体 | `service/`（`store.C().T(fn)` 包裹） |
| 业务不变量、分组维度→SQL 表达式映射、哈希/展示串等业务工具 | `service/` |
| 纯 SQL 工具（拼查询、分页拦截） | `store/base/` |

## Store 写法

- 按表/领域拆分 Store，不同表的操作不混在同一 Store（如操作 `api_keys.model_policy` + `apikey_model_access` 的 `ModelAccessStore` 独立于操作 `models` 的 `ModelStore`）。
- 每个 Store 的命名空间入口写在自己文件内：`func (s *Session) Xxx() *XxxStore`；**不要**集中到 `store/base.go`。
- Store 方法默认在调用方传入的 `*Session` 上执行，自身不感知是否在事务里：`Session.T(fn)` 包裹则进事务，`store.C()` 则非事务。
- `IN` 查询直接写 `IN (:ids)` 传 slice 参数，QueryBuilder 内置 `sqlx.In` 自动展开，**不手拼占位符**。
- 不依赖 `UPDATE ... RETURNING`、`SELECT ... FOR UPDATE`（SQLite/MySQL 不通用）。
- 方法注释只说明**做什么**，不写事务用法、调用时机、并发语义——那些属于 `service/`/`handler/` 的编排逻辑，写在 store 里是越界且误导。

## 分页

`store` 只暴露 `PageContext`（`store.PageContext` 是 `store/base.PageContext` 的别名）；分页入参/出参类型在 `manager/base`（`PageReq` / `PageResult`），handler 不直接 import `store/base`。

`QueryBuilder` 检测到 `PageContext` 时自动拦截：先 `SELECT COUNT(*) FROM (<原SQL>) t` 写回 `Total`，再追加 `LIMIT/OFFSET`。未 `SetPage` 时 `Select` 退化为普通查询。

## 测试

数据层测试自建内存 SQLite 并先调 `store.Init(db)`，不要连真实数据目录。改动列结构时同步 [`../sql/sqlite.sql`](../sql/sqlite.sql)（`store/schema_test.go` 与 `store/gate_schema_test.go` 会校验 DDL 与模型/版本号一致）。

# AGENTS.md — 数据库与 SQL

本文件补充仓库级规则 [`../AGENTS.md`](../AGENTS.md)，只写 `sql/` 与数据库结构相关的约定。
Go 侧的数据访问写法见 [`../store/AGENTS.md`](../store/AGENTS.md)。

## 文件分工

| 文件 | 内容 |
|------|------|
| [`sqlite.sql`](sqlite.sql) | **当前完整 schema**（新建库直接执行），开头建 `schema_meta` 表并写入版本行 |
| [`init-data.sql`](init-data.sql) | 种子数据：角色、权限、菜单、角色菜单。用 `INSERT OR IGNORE`，**可重复执行** |
| [`migrations/README.md`](migrations/README.md) | 迁移台账：版本表、每版增量 SQL、升级流程 |

**版本权威是 `constant.SchemaVersion`**，启动时与库里 `schema_meta` 表的版本行比对，不一致拒绝启动。`store/schema_version_test.go` 校验 DDL 写入的版本与常量一致，并校验整份 DDL 可重复执行。

**为什么用元数据表而不是 `PRAGMA user_version`**：本项目要兼容 MySQL / PostgreSQL，而 `user_version` 是 SQLite 专有（MySQL 没有等价物）。`schema_meta` 是三者的公共做法，Go 侧读取只有一条 `SELECT version FROM schema_meta WHERE id = 1`，不按驱动分支。新增其它数据库的 DDL 时，只有建表语句与写入版本行的语法不同（SQLite/PG 用 `INSERT ... SELECT ... WHERE NOT EXISTS`，MySQL 需要 `FROM DUAL`）。

## DDL 规范

- `PRIMARY KEY` 写在 `CREATE TABLE` 内；`UNIQUE` / `INDEX` 统一用 `CREATE INDEX` / `CREATE UNIQUE INDEX`（唯一索引命名 `uq_<表>_<列>`）。
- **不使用 FOREIGN KEY**，引用完整性由业务层保证。
- 时间列统一 `datetime('now', 'localtime')`（系统本地时间，非 UTC）。
- 迁移不在应用内自动执行；需要外部计算（哈希等）的步骤由人工完成并在台账里写明。

## 新增表 / 列的流程

1. 把结构直接写进 `sqlite.sql`（新建库一步到位），不要在 DDL 里罗列历史迁移步骤。
2. 新建 `migrations/v<N>-<主题>.sql`：只写**从上一版到本版**的增量语句。
3. `constant.SchemaVersion` 加一，并同步 `sqlite.sql` 里 `schema_meta` 的版本行。
4. 在 `migrations/README.md` 的版本表加一行，写清适用条件、是否破坏性、需人工介入的部分。
5. 同步 `store/model/models.go`、`store/schema_test.go` 与 `CHANGELOG.md`。

## 种子数据

- 权限种子（`role_permission`）：新增非超管接口时必须授权，否则 403。义务与门禁见 [`../manager/AGENTS.md`](../manager/AGENTS.md)。
- 菜单种子（`menus` / `role_menus`）：新增页面时必须同步，否则页面不可达。见 [`../frontend/AGENTS.md`](../frontend/AGENTS.md)。
- 存量库升级后重跑 `init-data.sql` 即可补齐新种子（`INSERT OR IGNORE` 保证幂等）。

## 注意

- 迁移语句**不可重复执行**（`ADD COLUMN` 重复会报 `duplicate column name`）：执行前用 `.schema <表名>` 确认。
- `DROP COLUMN` 需要 SQLite ≥ 3.35。
- 备份数据库时排除 `<数据目录>/keys/`（密钥与数据分离存放）。

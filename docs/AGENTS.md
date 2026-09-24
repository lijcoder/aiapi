# AGENTS.md — 文档标准

> 本文件规定**文档写在哪、写多长**。开发规则（红线、边界、自检）在 [`../AGENTS.md`](../AGENTS.md)。
> 新增或改动文档前先读本文件；`go test .` 会机械校验链接、锚点与体积预算。

## 一个事实只有一个家

| 层 | 职责 | 不该放 |
|----|------|--------|
| [`../AGENTS.md`](../AGENTS.md) | 每次都要知道的常驻规则：速查命令、目录地图、红线、边界判定式、自检清单 | 示例、流程步骤、细节说明；任何能从被链接文档复述的内容 |
| `<包>/AGENTS.md`（frontend / parser / proxy / manager / store / sql） | **本包**的做法与约定，进入该目录工作时会被自动加载 | 根已承载的跨包红线；其它包的事 |
| [`architecture.md`](architecture.md) | 跨包的"为什么"：分层与依赖方向、关键边界、事务、并发安全、运行时不变量、已知偏离 | 按包的操作步骤（→ 包级 AGENTS.md）、具体参数（→ 代码常量） |
| [`security.md`](security.md) | 认证、鉴权、密钥、脱敏、权限模型 | 接口清单（→ `api.md`）、实现步骤 |
| [`glossary.md`](glossary.md) | 领域词：一个概念一个规范词，指向权威定义处 | 实现细节、示例 |
| [`decisions/`](decisions/README.md) | 决策记录：为什么这么定、否决了什么、代价是什么 | 操作步骤、当前状态描述（会过期） |
| [`postmortem/`](postmortem/README.md) | 事故复盘：机制、为什么没被拦住、加了什么护栏 | 设计决策（→ decisions）、日常改动 |
| [`../sql/migrations/README.md`](../sql/migrations/README.md) | 数据库迁移台账：版本、每版增量 SQL、升级流程 | 当前完整 schema（→ `sql/sqlite.sql`） |
| [`../README.md`](../README.md) | 面向使用者的入口：快速开始、调用示例、配置参考、已知限制 | 逐条接口清单、运维流程、开发规则、内部实现理由 |
| [`api.md`](api.md) | 管理接口参考：完整接口表与调用示例 | 部署步骤、开发规范 |
| [`pricing.md`](pricing.md) | 模型计费配置：规则结构、条件、快照、用量口径 | 计费实现细节（→ `service/billing.go`） |
| [`deployment.md`](deployment.md) | 部署与运维：数据目录、反代、备份、升级、数据表 | 快速开始（→ README）、迁移 SQL（→ `sql/migrations/`） |
| [`../CHANGELOG.md`](../CHANGELOG.md) | 变更历史与影响面 | 未实施的计划（→ `TODO.md`） |
| [`../TODO.md`](../TODO.md) | 已识别但暂不实施的优化项 | 决策理由（→ decisions）、变更历史 |

## 放置判定

按顺序问自己，第一个"是"就是它的家：

1. 是**要做但还没做**的事？→ `TODO.md`
2. 是**为什么这么定**（含被否决的方案）？→ `docs/decisions/`
3. 是**某次故障为什么没被拦住**？→ `docs/postmortem/`
4. 是**做某件事的步骤**，且只涉及一个包？→ `<包>/AGENTS.md`
5. 是**跨包的原理**（分层、事务、并发、不变量）？→ `docs/architecture.md`
6. 是**使用者上手要看的**（安装、调用示例、配置项）？→ `README.md`
7. 是**使用者查阅用的参考**（接口清单、计费规则、运维流程）？→ `docs/api.md` / `docs/pricing.md` / `docs/deployment.md`
8. 是**已经发生的变化**？→ `CHANGELOG.md`

规则类内容还有一条额外判据：**能机械判定的，不要写成规则，写成门禁**（见 `gate_arch_test.go` / `gate_docs_test.go`），文档里只留一句指路。

## 写法

- 中文行文；一个事实只写一遍，别处链接过来。
- 具体名词优先：写清是哪个文件、哪个函数、哪条命令。避免"赋能/闭环/抓手/骨架"这类比喻词。
- 描述**当前状态**，不写变更叙事（叙事属于 `CHANGELOG.md`）。
- 每条规则尽量能回答"违反了会怎样"，否则它只是愿望。
- 包级 AGENTS.md 开篇必须写明"补充根 AGENTS.md"并向上链接。

## 体积预算：只覆盖"会写啰嗦"的文档

**唯一权威是 `gate_docs_test.go` 里的 `docBudgets`**（本文件不重复具体数字，避免两处漂移）。分两类处理：

| 文档类型 | 判据 | 约束方式 |
|----------|------|----------|
| **常驻 / 规则 / 流程类**（根与包级 `AGENTS.md`、`architecture.md`、`security.md`、`README.md`、`TODO.md`、迁移台账） | 膨胀 = 写啰嗦了 | **字节预算**：超限先把细节下沉到更低一层，确实无处可去才上调（需在提交信息里说明理由） |
| **目录型 / 参考型**（`api.md` 一行一个接口、`pricing.md` 描述配置结构） | 长度随产品表面线性增长，设上限等于给功能数量设上限 | **与源头一致性门禁**：`api.md` ↔ `manager/router/router.go` 的路由集合双向比对（`gate_arch_test.go`） |

判断方法很简单：**给它设预算会不会逼你删掉有用信息？** 会，就不是预算该管的文档，去找它的源头做一致性校验。


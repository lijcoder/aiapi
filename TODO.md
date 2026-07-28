# TODO - 待优化项

> 记录已识别但暂不实施的优化项，按主题分组。每项注明背景与建议方案，实施后可从本文档移除并在 CHANGELOG 记录。
> 已完成的历史项见 `temp/optimization-plan.md`（本地草稿，不入库）。

## 一、分布式部署准备（当前为单机版，未来需要时实施）

### 1. TOTP 失败计数去内存化 ⚠️ 分布式前必须做
- **现状**：`service/totp.go` 的 `failures map[string]int` 按 pending 票据计数（错 5 次作废），存在单进程内存中。
- **风险**：多实例部署后各实例独立计数，N 个实例 = 攻击者有 5×N 次尝试机会，锁定机制失效。
- **单实例下是安全的**，当前可正常使用。
- **候选方案**：
  - A. DB 计数表 `totp_attempts`（ticket_hash PK + fails + created_at），无新基础设施，票据 5 分钟寿命故表极小，定期清理 10 分钟前的行
  - B. Redis 计数（带 TTL 天然契合 5 分钟票据寿命，无需清理任务）——若确定引入 Redis 则优先选此
  - 配套增强：TOTP 验证码 6 位 → 8 位（改 `service/totp.go` 的 `Digits` 常量 + `ValidateCustom` + 前端输入框 + 测试，码空间 100 倍）；注意 6→8 对已绑定用户不兼容，需在无存量绑定时改

### 2. 数据库 SQLite → MySQL/PostgreSQL
- **现状**：单实例 SQLite（WAL）。多实例无法共享 SQLite 文件，是分布式的硬门槛。
- **待做**：
  - 补一套 MySQL DDL（`sql/sqlite.sql` 注释里引用的 `sql/mysql.sql` 实际不存在）
  - `store/driver` 增加 MySQL 支持（`sqlx.Rebind` 已兼容多驱动，改动小）
  - 注意 `datetime('now','localtime')` 等 SQLite 方言需替换
- **关联**：余额/计数的「先 UPDATE 后 SELECT」并发安全模式（AGENTS.md 2.4）在 MySQL 下同样适用，无需改业务逻辑

### 3. 登录接口限流与失败锁定（原计划 P2-10，暂缓）
- **现状**：已做统一文案 + 时序对齐（防枚举）；未做限流。
- **方案**（已评审，待开发）：按 IP+账号 失败 5 次/10 分钟锁 15 分钟 + 每 IP 每分钟 10 次限流；分布式下计数同样需共享态（DB 或 Redis），与第 1 项共用基础设施决策
- **锁定期提示文案待定**：返回通用错误（防骚扰式锁定）vs 明确提示"尝试过于频繁"

### 4. 日志聚合
- **现状**：lumberjack 写本地文件，多实例各写各的，分布式下需接日志聚合（ELK/Loki/stdout 采集），暂无需改动

## 二、性能优化（单机阶段可做，按需启动）

### 5. Auth 与 BudgetCheck 重复查询 Key+User
- **位置**：`proxy/handler/auth.go` + `proxy/handler/budget.go` 都调 `service.GetKeyAndUser`
- **方案**：Auth 查到的 key/user 存入 ctx，BudgetCheck 复用

### 6. 热路径配置缓存（models/providers/model_policy）
- **方案**：进程内 TTL 缓存，管理端变更时主动失效；**注意分布式下需改 Redis 或接受 TTL 内不一致**

### 7. 流式透传 buffer 512 → 32KB
- **位置**：`proxy/handler/response.go` streamResponse

### 8. request_logs 治理
- 加 `created_at` 索引（存量库需迁移）
- body 落库前截断（如 64KB）+ 脱敏开关
- 可选：异步批量写、定期清理

## 三、健壮性（小改动，随时可做）

### 9. 过期会话定时清理
- **现状**：`main.go` 只在启动时 `DeleteExpired` 一次
- **方案**：`time.Ticker` 每日清理（可顺带清理第 1 项的 `totp_attempts`）

### 10. 用量记录与扣费合并到同一事务
- **位置**：`proxy/handler/record.go`（当前 Insert 与扣费分离，可能账实不一致）

### 11. 余额并发透支（设计层面）
- **现状**：BudgetCheck 先放行后扣费，并发下预算可变负
- **待定**：业务容忍度（接受负值+告警 vs 预扣机制）

### 12. 核心链路测试补充
- `proxy/handler/*`、`parser/*`（流式/非流式）测试覆盖不足；已开头（`response_test.go`）

## 四、安全增强（可选）

### 13. 超管强制 2FA / 全员强制 2FA 开关
- 当前 2FA 为可选开启；可考虑超管强制或全局强制策略

### 14. 备用恢复码
- 用户丢失手机且无超管介入时会被锁死；可生成一次性恢复码

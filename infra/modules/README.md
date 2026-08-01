# infra/modules — 可复用 IaC 模块

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-001（目录骨架）；实例化见 TASK-003 ~ TASK-008 |
| 计划模块 | network / database / object-storage / event-stream / sfu / temporal |

## 规则

1. 模块只表达供应商中立能力（如"区域内对象存储桶"）；供应商绑定只允许出现在 `regions/` 实例层。
2. 追加式数据存储（证据、账本、审计）的数据库层约束（REVOKE UPDATE/DELETE）随 TASK-003 以迁移形式落地（ADR-0004）。
3. 每个模块提供 dev / staging / prod 三环境参数面：拓扑同构，副本数可缩减。
4. Redis 模块必须标注"非证据存储"（仅会话/限流/锁）。

# 数据库模块（PostgreSQL / Redis）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-003；docs/data/DATA-MODEL.md；ADR-0004；NFR-005、NFR-006 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.database` |

## PostgreSQL

- 每区主实例 + 跨 AZ 同步副本；production ≥3、staging ≥2、dev ≥1。
- 迁移工具：`services/migrate`（幂等执行 + SHA-256 校验和 fail-closed）。
- 追加式账本表（evidence_items / score_versions / usage_ledger / access_audits）
  对业务角色 `REVOKE UPDATE, DELETE`（ADR-0004）。

## Redis

- 模块契约强制 `evidence_storage: false`；仅会话状态、限流、分布式锁、在线状态、验证码散列。
- 禁止作为证据、分数、账本、授权的唯一存储；Redis 数据丢失不得导致业务证据丢失（NFR-005）。

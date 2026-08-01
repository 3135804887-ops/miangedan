# infra/modules — 可复用 IaC 模块

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-001（目录骨架）；实例化见 TASK-003 ~ TASK-008 |
| 计划模块 | network / database / object-storage / event-stream / sfu / temporal / observability |

## 规则

1. 模块只表达供应商中立能力（如"区域内对象存储桶"）；供应商绑定只允许出现在 `regions/` 实例层。
2. 追加式数据存储（证据、账本、审计）的数据库层约束（REVOKE UPDATE/DELETE）随 TASK-003 以迁移形式落地（ADR-0004）。
3. 每个模块提供 dev / staging / prod 三环境参数面：拓扑同构，副本数可缩减。
4. Redis 模块必须标注"非证据存储"（仅会话/限流/锁）。

## 已落地模块

- `database/`：PostgreSQL + Redis 模块契约（追加式账本约束、Redis 非证据）。
- `object-storage/`：uploads / exports / media 三桶隔离，media 30 天生命周期。
- `event-stream/`：六类区域事件流主题与载荷规则。
- `temporal/`：每区独立集群、`mgd-{region}-{env}-temporal` 命名空间与七域任务队列契约（TASK-004）。
- `observability/`：每区 OTLP 采集端点、strict 脱敏与中英双语状态页骨架（TASK-005，SEC-032/033）。

模块契约由 `python tools/validate_docs.py --suites data-platform,temporal,observability` 校验；
区域实例化由 `--suites regions` 校验（拓扑同构、3 AZ、零跨区引用）。

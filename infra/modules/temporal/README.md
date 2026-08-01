# Temporal 模块（每区独立集群/命名空间）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-004；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；ADR-0001、ADR-0005 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.temporal` |

## 集群

- 每数据区独立集群，跨 3 AZ 持久化（`cross_az: true`），区域间无共享命名空间（ADR-0005）。
- 生产历史保留基线 30 天（`history_retention_days: 30`），可按治理要求调高。

## 命名空间

- 命名模式：`mgd-{region}-{env}-temporal`（如 `mgd-cn-prod-temporal`）。
- 命名空间与区域/环境一致性由 `services/temporal` 校验（fail-closed）。

## 任务队列

七域固定：`ingestion` / `plan` / `interview` / `scoring` / `report` / `billing` /
`deletion`；所有权与载荷契约见 `workflows/README.md`，集合校验见 `services/temporal`。

## 故障恢复

- 集群跨 AZ 持久化，单 AZ 故障由 Temporal 自身恢复机制接管（`docs/architecture/DEPLOYMENT.md` 第 6 节）。
- 工作流故障不判失败、不丢证据；重试与幂等策略遵循 AGENTS.md 第 3 节与 NFR 规则。

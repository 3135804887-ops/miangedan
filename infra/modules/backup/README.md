# 备份与恢复模块（区域内每日完整 + WAL + PITR）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-008；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；SECURITY-REQUIREMENTS SEC-050、SEC-052 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.backup` |

## 规则

- 每数据区独立备份桶（`backup.bucket`，命名 `mgd-{region}-{env}-backups`）；备份不出区域（SEC-050）。
- 每日完整 + 持续增量（WAL 归档）+ PITR（SEC-052）；证据 RPO=0，其他状态 RPO ≤5s，RTO ≤30 分钟。
- 恢复对外服务前必须应用 tombstone 过滤（RETENTION-MATRIX）；一键恢复步骤见
  `docs/operations/RECOVERY-RUNBOOK.md`，季度演练按 `tools/backup/quarterly-drill-template.md` 执行并出报告。

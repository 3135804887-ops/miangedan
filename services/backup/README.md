# services/backup — 备份与恢复契约（CLI + 共享包）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（控制面工具链，yaml.v3） |
| 拥有任务 | TASK-008（EPIC-01）；真实存储备份执行随数据平台落地 |
| 追踪 | SECURITY-REQUIREMENTS SEC-050、SEC-052；RETENTION-MATRIX；docs/operations/RECOVERY-RUNBOOK.md |

## 职责

- **备份契约**：`Config.Validate` fail-closed——每日完整 + 持续增量（WAL）+ PITR、
  证据 RPO=0、其他 RPO ≤5s、RTO ≤30 分钟、区域内备份桶、恢复前强制 tombstone 过滤。
- **一键恢复**：CLI `backup -config <path> -mode restore-dry-run` 输出固定恢复步骤序列；
  真实恢复执行按 `docs/operations/RECOVERY-RUNBOOK.md`，季度演练模板见
  `tools/backup/quarterly-drill-template.md`。

## 用法

```sh
go run ./cmd/backup -config infra/regions/cn/envs/production.yaml
go run ./cmd/backup -config infra/regions/cn/envs/production.yaml -mode restore-dry-run
```

## 红线

1. 备份只在所属数据区内（SEC-050）；跨区桶即配置错误。
2. 证据 RPO 必须为 0；RTO 不得超过 30 分钟。
3. 恢复对外服务前必须应用 tombstone 过滤（RETENTION-MATRIX）。

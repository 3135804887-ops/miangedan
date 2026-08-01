# 恢复运行手册（RECOVERY-RUNBOOK）

| 字段 | 内容 |
|---|---|
| 文档编号 | OPS-RECOVERY-001 |
| 版本 | 0.1.0（2026-08-01，TASK-008） |
| 追踪 | PRD 容灾；SECURITY-REQUIREMENTS SEC-050、SEC-052；RETENTION-MATRIX；DEPLOYMENT.md 第 6 节 |

## 1. 目标

每日完整备份 + 持续增量（WAL 归档）+ 时间点恢复（PITR），每季度真实恢复演练；
已确认回答与评分证据 RPO=0，其他非关键状态 RPO ≤5 秒，区域级严重故障 RTO ≤30 分钟；
备份与恢复只在所属数据区内完成，不得跨区（ADR-0005）。

## 2. 一键恢复流程（按 `services/backup` CLI 固定序列）

1. 从区域备份桶还原最近一次每日完整备份。
2. 应用持续增量（WAL）至目标 PITR 时间点。
3. 应用 **tombstone 过滤**：恢复后已删数据不可见（RETENTION-MATRIX：先过滤再对外服务）。
4. 校验证据 RPO=0：已确认回答与评分证据完整、版本链未断。
5. 恢复区域服务并验证 RTO ≤30 分钟；恢复动作不得改变已冻结证据与分数（DEPLOYMENT 恢复原则）。
6. 记录演练/事故报告并复盘。

验证命令：`go run ./services/backup/cmd/backup -config <region-env.yaml> -mode restore-dry-run`

## 3. 季度演练

- 按 `tools/backup/quarterly-drill-template.md` 执行真实恢复演练并出报告（SEC-052）。
- 演练使用合成数据（fixtures/synthetic），禁止真实用户数据；记录可重现、可审计。
- 演练未达标：记 Incident，阻塞下一发布阶段并限期复练。

## 4. 事故恢复

- 单可用区故障：60 秒自动接管（多 AZ 健康检查 + 流量切换）。
- 区域级严重故障：区域内独立故障域备份 + 预置恢复编排，RTO ≤30 分钟；**不**切换到其他数据区。
- 备份损坏（演练发现）：立即修复备份链路并重新全量，升级 P0 复盘（DEPLOYMENT 11 异常处理）。

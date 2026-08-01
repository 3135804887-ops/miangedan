# 季度恢复演练模板（TASK-008）

| 字段 | 内容 |
|---|---|
| 频率 | 每季度一次真实恢复演练（PRD 容灾、SECURITY-REQUIREMENTS SEC-052） |
| 追踪 | TASK-008；SEC-052；docs/operations/RECOVERY-RUNBOOK.md |

## 演练前

- [ ] 选择演练数据区（cn/eu/intl 轮换），声明演练时间窗与影响范围（仅合成数据，禁止真实用户数据）。
- [ ] 记录演练前备份状态：最近每日完整备份时间、WAL 归档连续性与校验和。
- [ ] 指定演练负责人与观察员（RPO/RTO 计时）。

## 演练步骤（按 RECOVERY-RUNBOOK 一键恢复序列）

1. [ ] 从区域备份桶还原最近一次每日完整备份。
2. [ ] 应用持续增量（WAL）至目标 PITR 时间点。
3. [ ] 应用 tombstone 过滤，抽样验证已删数据不可见（RETENTION-MATRIX）。
4. [ ] 校验证据 RPO=0：恢复点内已确认回答与评分证据完整可查。
5. [ ] 恢复区域服务，验证 RTO ≤30 分钟（记录实际耗时）。
6. [ ] 运行 `go run ./services/backup/cmd/backup -config <region.yaml> -mode restore-dry-run` 输出一致性摘要。

## 演练后

- [ ] 填写结果表：计划 RTO/RPO vs 实际 RTO/RPO、异常与根因、行动项与负责人。
- [ ] 演练记录入 Incident/审计（可重现、可审计，DEPLOYMENT.md 第 10 节）。
- [ ] 未达标即记 Incident 并限期复练（EPIC-01-INFRA-DESIGN 第 7 节风险）。

## 结果记录

| 指标 | 目标 | 实际 | 达标 |
|---|---|---|---|
| 证据 RPO | 0 秒 |  | ☐/☑ |
| 其他状态 RPO | ≤5 秒 |  | ☐/☑ |
| 区域 RTO | ≤30 分钟 |  | ☐/☑ |
| tombstone 过滤 | 已删数据不可见 |  | ☐/☑ |

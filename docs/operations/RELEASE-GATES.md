# 三数据区生产验证与 GA 放量守门（TASK-096）

- 文档编号：OPS-GATES-001
- 版本：0.1.0（2026-08-04）
- 追踪：TASK-096；PRD-001 Phase 3/4；NFR-001 ~ NFR-004；OD-03/OD-09；`config/feature-flags.yaml`
- 一致性锚点：`docs/testing/RELEASE-CHECKLIST.md`、`docs/architecture/DEPLOYMENT.md`、`docs/operations/RECOVERY-RUNBOOK.md`

## 1. 目标

把 GA 放量变成可暂停、可回滚的分档守门流程：1% → 5% → 20% → 50% → 100%，每档有明确验证条件与暂停/回滚规则；单区故障不扩散、不违法跨境、进行中会话不被中途变更。

## 2. 放量门槛

- 1%：单区生产验证通过（NFR 全绿、无严重事故）；开启 `region.cn.enabled=true` 与 `provider.*.secondary_ramp_percent=1`。
- 5%：观察期内证据完整率 100%、计费一致、无越权/泄露事件；新会话灰度为 5%，进行中会话固定版本。
- 20%：云上 2× 预计峰值压测通过（TASK-092）、单 AZ 故障 60s 接管演练通过、区域 RTO ≤ 30min 验证通过。
- 50%：稳定性回归（TASK-045）通过、安全红队复查通过、客服/退款/状态页流程就绪。
- 100%：全部 Phase 3/4 门槛通过 + 三方放量审批；eu/intl 需 OD-03 法律方案完成后另行开闸。

## 3. 暂停与回滚规则

- 任何 SLO 退化、事故或放量指标恶化：新会话立即回滚到稳定版本（`provider.*_ramp_percent=0` / 关闭区域 flag）。
- 进行中正式面试不中途变更供应商或版本（PRD 规则；切回滚仅影响新会话）。
- 回滚后复盘并记录 Incident，再次放量需从上一档重新走守门。

## 4. 与 Feature Flags 映射

- `region.{cn,eu,intl}.enabled`：区域开闸总开关（默认 false，未通过生产验证前保持关闭）。
- `provider.{llm,asr,tts,avatar}.secondary_ramp_percent`：备选供应商灰度比例（0~100，默认 0）。
- `recording.raw_audio/video.enabled`：录制默认关闭，需用户单独明确授权。
- `reliability.fault_recovery.enabled`：protected=true，故障恢复能力不得关闭。

## 5. 窗口 2 本地证据（2026-08-04）

- 混沌演练（TASK-091 本地轮）：自建 LiveKit 断服 10s 后自动重启，双端自动重连成功，重连握手 369ms；帧计数恢复后继续增长，无报错。报告 `ai/evals/providers/reports/m2-chaos-round1.json`。
- 并发压测（TASK-092 本地轮）：1 发布（720p/24fps）+ 10 订阅，全部连通且全部收到视频（每路 550+ 帧、21.3~21.5fps），连接 P50 643ms，SFU 内存 +7MB。报告 `ai/evals/providers/reports/m2-load-round1.json`。
- 正式验收（单 AZ 60s 接管、区域 RTO、云上 2× 峰值）随 M3 云上统一部署执行。

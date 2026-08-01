# 公开状态页骨架（STATUS-PAGE）

| 字段 | 内容 |
|---|---|
| 文档编号 | OBS-STATUS-001 |
| 版本 | 0.1.0（2026-08-01，TASK-005） |
| 追踪 | PRD Observability and Operations；SEC-033；DEPLOYMENT.md 第 6 节；RELEASE-CHECKLIST |
| 上线要求 | 三数据区（cn/eu/intl）各一个独立公开状态页，中英文双语同步发布 |

## 1. 组件与状态（Components & Status）

每个状态页展示以下组件的当前状态（operational / degraded / partial outage / major outage）：

账户与资产、材料解析、计划生成、实时面试房间、数字人/ASR/TTS、AI 编排、评分与报告、
商业计费、删除与数据权利、状态页自身。

> English: each regional status page lists the current status of the components above
> (accounts/assets, ingestion, planning, realtime rooms, avatar/ASR/TTS, AI orchestration,
> scoring & reports, billing, deletion & data rights, and the status page itself).

## 2. 结构化事故时间线（Incident Timeline）

- 每次事故（incident）自动生成结构化时间线：发现时间、影响组件与范围、恢复时间、
  根因（RCA）、复盘链接；时间线只含匿名技术信息。
- 按 SEV-1/2/3 分级（SECURITY-REQUIREMENTS 4.7）自动发布与更新。

> English: every incident publishes a structured timeline (detected, impacted components,
> recovered, RCA, postmortem) containing anonymous technical information only.

## 3. 错误预算与 SLO（Error Budget & SLO）

- 关键 SLO（可用性、有效完成率、实时性能）按月度统计并公开当前错误预算。
- 关键 SLO 月度错误预算耗尽时，暂停非必要发布，优先恢复稳定性（PRD Observability）。
- 状态页展示当前错误预算消耗率与历史 SLO 达成情况。

> English: monthly error budgets for key SLOs are published; when a critical SLO error
> budget is exhausted, non-essential releases are paused until stability is restored.

## 4. 区域化与脱敏（Regional Isolation & Redaction）

- 每区独立状态页与独立 OTLP 采集端点；状态页不跨区聚合用户内容。
- 状态页只展示匿名技术指标，不展示正文、令牌、媒体或任何个人标识。

> English: each data region runs its own status page and OTel endpoint; no user content
> crosses regions, and the page shows anonymous technical metrics only.

## 5. 中英文双语发布（Bilingual Publishing）

- 所有状态与事故时间线条目同步发布中英文版本（zh-CN / en-US）。
- 上线（Phase 3）时中英文状态页、客服、隐私请求与事故通知渠道同步上线（RELEASE-CHECKLIST）。

## 6. 订阅与历史（Subscription & History）

- 提供 RSS/Atom、公开 JSON API 与邮件订阅；历史可用性记录至少 90 天可查。
- 状态页自身可用性纳入监控（避免“状态页失明”）。

## 7. 守门（Release Gate）

- 错误预算耗尽时暂停非必要发布；恢复后经复盘签字放行。
- 状态页发布本身走灰度（先 internal 后 public），避免事故信息先于处置扩散。

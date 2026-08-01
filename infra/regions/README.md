# infra/regions — 三数据区实例化

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-001（目录骨架）；TASK-002（拓扑落地）；ADR-0005；OD-09 |
| 区域 | cn / eu / intl |

## 原则

- 每区一套完整独立环境：`env-{cn,eu,intl}-{dev,staging,prod}`，共 9 个环境；dev/staging 可缩减副本数，拓扑同构。
- 每区独立持有：网络、PostgreSQL、Redis、对象存储桶、事件流、SFU 节点、Temporal 命名空间、密钥引用、供应商白名单、邮件/通知通道。
- 账户 `data_region` 为硬归属；全球入口按区域路由，区域不匹配请求拒绝并告警。
- 区域间无任何默认通路；供应商评测结论按区域分别得出（`docs/testing/PHASE0-PROVIDER-EVALUATION.md`）。

# infra/regions — 三数据区实例化

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-001（目录骨架）；TASK-002（拓扑落地）；ADR-0005；OD-09 |
| 区域 | cn / eu / intl |

## 原则

- 每区一套完整独立环境：`env-{cn,eu,intl}-{dev,staging,prod}`，共 9 个环境；对应
  `{cn,eu,intl}/envs/{dev,staging,production}.yaml`（TASK-002 已落地）；dev/staging 可缩减副本数，拓扑同构。
- 每区独立持有：网络、PostgreSQL、Redis、对象存储桶、事件流、SFU 节点、Temporal 命名空间、密钥引用、供应商白名单、邮件/通知通道。
- 账户 `data_region` 为硬归属；全球入口按区域路由，区域不匹配请求拒绝并告警。
- 区域间无任何默认通路；供应商评测结论按区域分别得出（`docs/testing/PHASE0-PROVIDER-EVALUATION.md`）。

## 校验

- `python tools/validate_docs.py --suites regions`：强制区域代码与文件目录一致、环境名合法、
  每环境 ≥3 AZ、PostgreSQL 副本数达环境门槛、Redis 非证据存储、Temporal 队列齐全、
  资源命名按 `mgd-{region}-{env}-` 前缀（`prod` 为 `production` 的短资源代码）、
  无跨区引用、供应商白名单按区域标识。
- 校验失败即 CI 门禁失败（fail-closed），不允许带错误区域配置部署。

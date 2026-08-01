# 观测模块（每区 OTel 采集 + 状态页）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-005；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；SEC-032、SEC-033 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.observability` |

## OpenTelemetry

- 每数据区独立 OTLP 采集端点（`{otel_collector}`），协议 http/protobuf，仅本区服务上报。
- 指标与追踪资源必须携带 `data_region`，属性白名单见 `docs/observability/LOGGING-POLICY.md`；
  禁止面试正文作为标签或追踪属性（PRD Observability and Operations）。

## 脱敏

- 默认 `strict`，SDK 级过滤（`services/observability`）；生产环境强制 strict（SEC-032）。
- 日志/追踪/指标零正文：不记录简历正文、完整回答、令牌、原始媒体。

## 状态页

- 每区独立公开状态页（`{status_page}`），中英文双语，含组件状态、结构化事故时间线、
  错误预算与月度 SLO（SEC-033）；关键 SLO 错误预算耗尽时暂停非必要发布。

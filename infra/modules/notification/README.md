# 通知与身份通道模块（区域化邮件 + 身份提供商开放矩阵）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-007；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；PRD FR-027；ADR-0005 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.notification` 与 `topology.identity_providers` |

## 规则

- 每数据区独立邮件通道（`channels.email.per_region: true`），发件人区域化
  （`notification.email_from`）；跨区发送默认拒绝（`isolation.cross_region_send: false`）。
- 身份提供商开放矩阵（FR-027）：邮箱验证码全区域先行；微信仅 cn；Google/Apple 仅 eu/intl；
  区域配置超出开放矩阵即部署失败（`services/identity/provider` 校验 fail-closed）。
- 单区通道故障不影响他区（`isolation.per_region_channels: true`，TASK-007 验收）；
  路由契约见 `services/notify`。

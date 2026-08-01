# consent 服务（授权中心）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go（控制面服务，每服务一个模块，经根 `go.work` 统一工作区） |
| 拥有任务 | TASK-011 |
| 追踪 | TASK-011、FR-040、SEC-030/031/041/044、ADR-0004/0005 |

## 已实现能力

- 六类独立授权：`core_service`、`raw_av_recording`、`org_sharing`、`product_analytics`、
  `model_training`、`marketing`；缺失记录按未授权，模型训练默认关闭。
- 封闭 scope：机构任务/数据类别、音视频类别、营销渠道；不允许自由文本、个人内容或敏感字段值。
- 授予与撤回只追加版本，证据记录文案/隐私政策版本、展示时间、枚举化 UI 上下文和 SHA-256；
  每个版本与 content-free AccessAudit 同事务提交。
- 撤回成功返回后 `Decide` 对同范围立即拒绝；状态不确定 fail-closed。写操作持久幂等，审计失败
  整体回滚并可同键重试；并发重复不产生多份版本或审计。
- 原始音视频通过 TASK-010 `identity.AgeStatus` 最小权限适配器检查：仅成人可用，未成年、非法状态或
  资格读取失败均拒绝，授权最长 30 天；每次在线访问判定重新核对年龄状态，适配器不读取账户资料或
  联系方式。
- HTTP 适配器仅暴露 `/v1/consent/*`，认证复用 `services/identity` 业务令牌并核对部署/令牌区域。
- 授予、撤回与在线判定输出固定低基数观测事件，生产组合可映射到 OpenTelemetry 指标、跨度事件和
  脱敏结构化日志；事件不含用户标识、scope/hash、幂等键或证据，观测故障不影响授权事务。

数据库迁移为 `services/migrate/migrations/0011_consent_grants.sql`，历史迁移未修改；
`consent_grants` 对业务角色无 UPDATE/DELETE。

## 验证

```bash
cd services/consent
go test -count=3 ./...
go vet ./...
```

覆盖六类正常授权、默认关闭、异常 scope/证据/有效期、未成年拒绝、并发幂等、同键异参冲突、
撤回即时失效、审计失败原子回滚与同键重试，以及 HTTP 身份/区域/严格 JSON 契约。
原始音视频还覆盖成功后同键重试跳过已变化的动态资格/到期检查，以及在线判定重新核对成人资格。

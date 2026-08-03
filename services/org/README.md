# org 服务（机构租户）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go（控制面服务，每服务一个模块，经根 `go.work` 统一工作区） |
| 拥有任务 | EPIC-08（TASK-070 ~ TASK-074） |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 当前状态

TASK-001 工程骨架 + TASK-002 区域自检：最小入口 `cmd/org`；启动时校验
`DATA_REGION` 与 `INFRA_REGION` 一致且 `SERVICE_ENV` 合法（共享包 `services/region`，
正常/异常单测已配），无业务实现。
业务实现按拥有任务推进；开工前必读 AGENTS.md 及该任务对应的契约文档（领域、API、数据与安全）。

## 已实现（TASK-070 首次业务实现，FR-034）

- **机构租户**：创建者以个人账户加入并成为所有者（无影子账户）；机构状态
  active/suspended/deactivated；幂等键去重。
- **六类角色权限分离**：owner/admin/instructor/privacy_auditor/finance/candidate
  权限矩阵（财务/审计/教学/管理默认分离；owner 仅本人可变更所有者角色）。
- **邀请适配点**：link/org_email/bulk_list/sso/scim；14 天有效期、幂等键、
  邮箱匹配校验；用户以个人账户接受邀请加入。
- **成员与机构管理**：成员列表、角色调整、退出（left_at 保留个人记录）、
  停用/启用/注销；全部变更写追加式审计（SELECT/INSERT only）。
- 迁移 `0065_org_tenants.sql`（organizations/org_members/org_invitations）。

## 规划（后续任务）

- TASK-071 训练任务与模板（可配项/禁止项 + 违规审计）；
  TASK-072 按任务细粒度结果授权；TASK-073 聚合分析（<10 人隐藏）；
  TASK-074 访问审计与退出即时失效。

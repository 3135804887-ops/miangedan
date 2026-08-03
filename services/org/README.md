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

## 已实现（TASK-071，FR-035）

- **训练任务与模板**：可配项白名单（岗位/轮次/角色/时长/难度/语言/工具/截止/
  练习次数/机构额度）；禁止项（60 分线/统一评分算法/保护属性/证据标准/跨轮解锁/
  正式复核）大小写不敏感识别、拒绝并写审计；任务状态 draft → published → closed。
- **默认最小可见**：完成情况仅计数（接受/未开始/进行中/已完成/退出/系统故障/
  机构额度消耗），不暴露个人结果。
- 迁移 `0066_assignments.sql`（assignments/assignment_members）。

## 已实现（TASK-072，FR-035）

- **按任务细粒度结果授权**：六类封闭枚举范围（total_score/radar/round_results/
  full_report/transcript/media）+ 有效期 + 可撤回（幂等）；机构侧访问校验写
  AccessAudit；撤回/到期在线访问立即失效（ExpireShares 到期扫描）。
- **"已完成未共享"展示**：任务摘要仅计数 ResultShared/ResultNotShared，
  不显示失败。
- 迁移 `0067_assignment_shares.sql`。

## 已实现（TASK-073，FR-036）

- **聚合分析**：岗位类别/能力维度分组（overall 汇总）、完成率、维度均值、
  提升趋势（单人首末对比）；细分群体 <10 人隐藏且不返回指标。
- **无个人排名**：PersonalRankingAvailable 恒 false，无个人排行榜/排名/
  候选人搜索；个人分数样本由评分服务注入，机构侧不持久化。

## 已实现（TASK-074，FR-034/FR-035）

- **访问审计**：谁/何时/访问了什么（privacy_auditor/owner 可见；追加式）。
- **退出即时失效**：退出/被移除 → 分享授权立即撤回、令牌判定失效
  （IsMemberAccessValid）；机构停用/注销 → InvalidateOrgAccess 撤回全部
  共享链接并写审计；个人记录保留、审计继续存在。

## 规划（后续任务）

- EPIC-09 治理（TASK-080~085）在 adminapi 推进。

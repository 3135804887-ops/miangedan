# adminapi 服务（运营治理后台 BFF）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go（控制面服务，每服务一个模块，经根 `go.work` 统一工作区） |
| 拥有任务 | EPIC-09（TASK-080 ~ TASK-085） |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 当前状态

TASK-001 工程骨架 + TASK-002 区域自检：最小入口 `cmd/adminapi`；启动时校验
`DATA_REGION` 与 `INFRA_REGION` 一致且 `SERVICE_ENV` 合法（共享包 `services/region`，
正常/异常单测已配），无业务实现。
业务实现按拥有任务推进；开工前必读 AGENTS.md 及该任务对应的契约文档（领域、API、数据与安全）。

## 已实现（TASK-080 首次业务实现，FR-037）

- **运营监控**：区域在线房间/排队/容量/供应商健康/SLO/错误预算；房间快照仅
  匿名会话编号与技术状态（无姓名/简历/回答/媒体）。
- **供应商状态变更**：active/ramping/disabled；停用必须记录原因并写审计。
- **运营红线**：OperatorSessionGuard 拒绝加入/旁听/代答并写审计；
  后台角色与跨区访问校验。
- 迁移 `0068_admin_ops.sql`（ops_providers/ops_room_snapshots/ops_region_status）。

## 规划（后续任务）

- TASK-081 版本治理；TASK-082 禁止改分系统级约束；TASK-083 数据权利请求；
  TASK-084 追加式审计与 MFA；TASK-085 客服工单。

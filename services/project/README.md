# project 服务（项目/计划/状态机 API）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go（控制面服务，每服务一个模块，经根 `go.work` 统一工作区） |
| 拥有任务 | TASK-016 ~ TASK-018 |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 当前状态

- TASK-001 工程骨架 + TASK-002 区域自检（`DATA_REGION == INFRA_REGION` 且 `SERVICE_ENV` 合法）。
- TASK-016 已实现（FR-009 ~ FR-011）：`services/project` 提供 InterviewProject / PlanVersion /
  RoundConfig 应用服务（创建/查询/列表/重命名/删除/复制、计划编辑与确认冻结），
  `httpapi` 按 `docs/api/openapi.yaml` 的 `/v1/projects`、`/v1/projects/{id}/plan` 契约暴露；
  冻结规则：确认后量表/轮次权重/轮次列表/覆盖方案冻结，开始后编辑返回 `state_conflict`；
  不完整计划（缺覆盖方案或量表）确认返回 `plan_incomplete`（422）。
  轮次边界与类型注册来自 `config/interview-flows/v1/default.yaml`（`MGD_FLOW_CONFIG` 可覆盖）。
- 计划生成（`plan:generate`）由 TASK-033 落地，当前返回 501 占位；
  材料派生筛选（company/job_title）随 TASK-018 落地；计费版本冻结（quote_id）随 TASK-060/061。
- 当前存储为内存实现（`memory_store.go`），生产持久化随数据平台接入；迁移
  `services/migrate/migrations/0016_interview_projects.sql` 已落地表结构与追加式约束。

业务实现按拥有任务推进；开工前必读 AGENTS.md 及该任务对应的契约文档（领域、API、数据与安全）。

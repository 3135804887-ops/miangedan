# project 服务（项目/计划/状态机 API）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go（控制面服务，每服务一个模块，经根 `go.work` 统一工作区） |
| 拥有任务 | TASK-016 ~ TASK-018 |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 当前状态

TASK-001 工程骨架：仅含最小入口 `cmd/project` 与 fail-closed 区域自检（正常/异常单测已配），无业务实现。
业务实现按拥有任务推进；开工前必读 AGENTS.md 及该任务对应的契约文档（领域、API、数据与安全）。

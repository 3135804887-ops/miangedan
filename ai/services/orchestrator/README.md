# mgd-orchestrator（LangGraph 面试官编排 AI 服务）

| 字段 | 内容 |
|---|---|
| 技术基线 | Python ≥3.11，src 布局，ruff + mypy(strict) + pytest |
| 拥有任务 | TASK-032 |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 当前状态

TASK-001 工程骨架：仅含 `require_data_region` fail-closed 自检与正常/异常单测，无业务实现。
AI 行为实现必须遵循 `docs/ai/` 契约（编排、评分、交接、提示词政策、供应商适配层），禁止业务代码直连供应商 SDK。

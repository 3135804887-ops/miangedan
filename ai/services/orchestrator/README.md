# mgd-orchestrator（LangGraph 面试官编排 AI 服务）

| 字段 | 内容 |
|---|---|
| 技术基线 | Python ≥3.11，src 布局，ruff + mypy(strict) + pytest |
| 拥有任务 | TASK-031、TASK-032、TASK-033 |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 已实现

- **区域自检**（TASK-001/002）：`require_data_region` 与 `check_startup`（fail-closed）。
- **提示词注册表**（TASK-031）：`mgd_orchestrator.prompt_registry`
  - 从 `ai/prompts/*.md` 解析契约元数据（prompt_id/version/layer/schema/status）；
  - 四层组装（system/developer/session/data），data 层不可信边界包裹，注入命中即标记；
  - 输出 JSON Schema 校验（fail-closed）；版本固定（活跃正式会话不匹配即拒绝）。

## 规划（后续任务）

- TASK-032 LangGraph 面试官决策图；
- TASK-033 计划生成链路（来源融合、轮次建议、安全过滤、重新生成）；
- TASK-034 跨轮交接包；TASK-035 注入防护与内容安全管道；TASK-036 AI 评测框架。

AI 行为实现必须遵循 `docs/ai/` 契约（编排、评分、交接、提示词政策、供应商适配层），
禁止业务代码直连供应商 SDK。

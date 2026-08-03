# mgd-orchestrator（LangGraph 面试官编排 AI 服务）

| 字段 | 内容 |
|---|---|
| 技术基线 | Python ≥3.11，src 布局，ruff + mypy(strict) + pytest |
| 拥有任务 | TASK-031、TASK-032、TASK-033、TASK-034、TASK-035、TASK-050 |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 已实现

- **区域自检**（TASK-001/002）：`require_data_region` 与 `check_startup`（fail-closed）。
- **提示词注册表**（TASK-031）：`mgd_orchestrator.prompt_registry`
  - 从 `ai/prompts/*.md` 解析契约元数据（prompt_id/version/layer/schema/status）；
  - 四层组装（system/developer/session/data），data 层不可信边界包裹，注入命中即标记；
  - 输出 JSON Schema 校验（fail-closed）；版本固定（活跃正式会话不匹配即拒绝）。
- **面试官决策图**（TASK-032）：`mgd_orchestrator.interviewer_graph`
  - 与 LangGraph StateGraph 语义对齐的确定性迷你引擎（节点/条件边/编译/调用/检查点恢复）；
  - 覆盖点推进、动态追问（预算内且不越出已确认覆盖点）、打断策略、工具白名单；
  - 图只产出"建议"，不直接写业务状态；重放安全（NFR-006）。
- **计划生成链路**（TASK-033）：`mgd_orchestrator.plan_generator`
  - 来源融合（可信来源引用；无来源回退通用模板并标记 AI 推导）；
  - 轮次建议（默认 3 轮、1-5 轮与 10-60 分钟边界、六维权重和 100）；
  - 安全过滤（PII 复述/注入检测，重生成 ≤2 次）；单轮失败只重试失败模块；
  - 输出对齐 `interview-plan.schema.json`（服务层补齐项目字段）。
- **跨轮交接包**（TASK-034）：`mgd_orchestrator.handoff_generator`
  - 组装八类必备内容（简历/JD 快照引用、轮次纪要、评价、风险、已验证能力、
    未覆盖点、禁止重复问题与允许重新验证例外）；
  - 上下文压缩（超预算按 HANDOFF-SPEC 优先级，不得删除简历/JD/未覆盖/禁止重复）；
  - 事实完整性独立复核（no_new_facts / source_refs_complete，声明与复核不一致即拒绝）；
  - 敏感字段零携带（命中即拒绝生成并告警）；输出过 handoff-package Schema（fail-closed）；
  - 语义去重执行层：`repeats_previous_question` / `allowed_to_reverberify`。
- **提示注入防护与内容安全管道**（TASK-035）：`mgd_orchestrator.safety_pipeline`
  - 以 `config/safety/policy.yaml` 为唯一事实源（保护属性/禁止类别/动作/重生成次数）；
  - 注入检测（prompt_registry 基线 + 编码混淆 + 工具诱导）与指令中和
    （sanitize_and_log，内容仍按数据处理、默认不向用户暴露安全细节）；
  - 禁止内容分类（歧视/侮辱/无关隐私/危险/骚扰/作弊协助/录用预测/PII 复述）与
    动作一一对应 policy.yaml；阻断-重生成 ≤3 次、危险/骚扰直接升级人工；
  - 评分证据保护属性零携带扫描（evidence_scan）与最小化审计记录。
- **报告生成器**（TASK-050）：`mgd_orchestrator.report_generator`
  - 由冻结 ScoreVersion/证据摘要/交接包/输入模式标记生成报告模块
    （overview/radar/job_match/rounds/communication_analysis/tool_performance/
    training_plan），分数只读、证据可回溯、雷达图文字等价；
  - 模块级失败重试（FR-026：失败模块 status=failed 可单独重试 ≤2 次）；
  - 输出过 `report.schema.json` 校验（fail-closed）；训练用途声明与
    deletion_entry 强制；保护属性零携带（evidence_scan 脱敏）。
- **AI 教练练习**（TASK-052）：`mgd_orchestrator.training_coach`
  - 原题/变体/框架/示例练习项（变体只关联已考覆盖点，不预演后续轮次）；
  - 逐步反馈：亮点 → 缺口 → 下一步（先优势后改进，聚焦行为与证据）；
  - 练习隔离红线：is_formal_evidence 恒 false、PracticeRecord 独立于
    正式证据链、永不产生 ScoreVersion；用户回答注入按数据处理并说明隔离；
  - 安全：录用预测/作弊/侮辱阻断重生成 ≤3、危险/骚扰升级人工；
    简历/JD 原文不进入练习内容。

## 规划（后续任务）

- TASK-053 正式重试；TASK-054 结果流（祝贺/失败阻断/累计复盘）。

AI 行为实现必须遵循 `docs/ai/` 契约（编排、评分、交接、提示词政策、供应商适配层），
禁止业务代码直连供应商 SDK。

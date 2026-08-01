# mgd-parsing（简历/JD 解析 AI 服务）

| 字段 | 内容 |
|---|---|
| 技术基线 | Python ≥3.11，src 布局，ruff + mypy(strict) + pytest |
| 拥有任务 | TASK-013、TASK-014 |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## TASK-013 实现

- `ResumeParsingService` 只读同用户、同区域且经 TASK-012 安全接受的 uploads/accepted 原件；
  解析失败保留输入，可幂等地只重试失败步骤。
- `ResumeParsingProvider` 为供应商中立协议；`SyntheticResumeParsingProvider` 仅供合成测试和
  本地开发，生产组合必须经 TASK-030 注册表注入，不允许业务代码导入厂商 SDK。
- 提示按 PROMPT-POLICY 分成 L1/L2/L4；Schema/暂时错误自动重试最多 2 次，仍失败返回原件
  保留、可重试、未计费、不影响评分的真实影响。
- 每个字段 add/replace/remove/confirm 均追加不可变草稿版本；低置信度路径未逐项处理时禁止
  最终确认；只有用户确认的冻结版本可构建面试上下文与评分上游材料。
- SEC-040 使用解析前脱敏、模型输出后递归清洗、版本写入前 Schema/零命中断言、两类下游
  材料组装前再扫描四道门。排除记录只含类别，不含原值。

测试及 AI 评测均只使用 `fixtures/synthetic/resumes/` 与
`ai/evals/datasets/resume-parsing-security.jsonl` 的合成材料。

## TASK-014 实现

- `JobParsingService` 保存所属区 JD 粘贴原文；调用供应商中立 `JobParsingProvider` 前整段剔除
  薪资福利、公司福利与招聘联系人，模型输出后递归清洗，写版本及下游组装前再次零命中断言。
- JD 使用 L1/L2/L4 分层；仅简历模式只读取 TASK-013 已确认且通过 SEC-040 的 L3 安全画像，
  不复制简历原文。TASK-030 前只提供 `SyntheticJobParsingProvider` 合成桩，不绑定厂商 SDK。
- AI 推导重点固定保留 `inference_id`、`ai_inferred=true`、`editable=true`；仅简历模式的岗位
  核心字段全部登记 `ai_derived_fields`。逐字段人工校对追加版本，不允许移除来源标记。
- Schema/暂时错误自动重试最多 2 次；仍失败保留原始输入，可用新幂等键只重试解析步骤。
  只有人工确认的冻结版本能构建面试上下文与评分上游材料。
- `MaterialReadinessService` 固定实现 `full/jd_only/resume_only/neither` 四种影响弹窗；非 full
  只有 `accepted=true` 且与用户、区域、模式和影响快照严格匹配的追加式同意才能继续。
- `JobParsingObserver` 只允许区域、来源类型、状态、降级模式、可重试性与 `trace_id` 等固定低基数
  字段，用于解析成功/失败/重试及降级评估/同意的 OpenTelemetry 指标与跨度事件；禁止正文、用户 ID、
  联系方式或密钥进入日志/指标标签。

测试与评测只使用 `fixtures/synthetic/jobs/` 和
`ai/evals/datasets/job-parsing-governance.jsonl` 的合成材料。

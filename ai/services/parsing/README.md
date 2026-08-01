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

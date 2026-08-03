# AI 评测（ai/evals）

追踪：IMPLEMENTATION_PLAN.md TASK-036/TASK-045；PROMPT-POLICY 第 13 节；
SCORING-SPEC 第 10 节。

## 目录

- `datasets/`：黄金集（JSONL，`synthetic: true`，每行一个用例）；
- `expected-results/`：与数据集同名的预期结果（`<stem>.expected.json`）；
- `reports/`：可重复运行的评测报告（`<stem>.eval.json`）与稳定性回归报告
  （`stability.json`，TASK-045 产物）。

## 运行

```bash
cd ai/services/evals
PYTHONPATH=src python -m mgd_evals.run --datasets zh-core,en-core --out ../../evals/reports
```

框架（`mgd_evals`）能力：

- 数据集与预期结果对齐校验（缺预期条目即失败）；
- 内置评测器：交接包（handoff）、内容安全（safety）、报告生成（report）、
  通用契约（generic）；
  未支持场景记为 `skipped`（如实报告，不伪装通过）；
- 报告 JSON：`report_kind=eval`，含 totals/metrics/thresholds/cases；
  同输入同结论（确定性）；
- `validate_stability_report()`：校验 TASK-045 稳定性报告门槛
  （维度差 ≤3 比例 ≥95%、及格结论一致率 ≥98%）。

## 门槛

- 交接完整性（八类必备内容）与事实完整性：100%；
- 红队注入/保护属性对抗用例：100% 阻断或中和；
- 稳定性回归（TASK-045）：重复评分 95% 维度差 ≤3 分、及格结论一致率 ≥98%，
  报告经 TASK-095 发布硬门槛使用。

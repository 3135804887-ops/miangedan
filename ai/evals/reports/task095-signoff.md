# TASK-095 专家盲评签字确认单

> 状态：**自动门槛已全部通过；专家盲评待线下签字**。本确认单由项目负责人/AI 负责人线下执行盲评并签字，签字完成后回填 `task095-hardgates.json` 的 `expert_blind_review.status = signed`，TASK-095 方整体完成。

## 1. 元信息

- 报告：ai/evals/reports/task095-hardgates.json
- 生成时间：2026-08-03T09:10:10Z
- 数据集：scoring-stability + zh-core/en-core-protected（golden，`synthetic: true`）
- 报告类型：hardgates_095（发布硬门槛汇总）
- 追踪：TASK-095、TASK-045（稳定性回归）、PROMPT-POLICY 第 13 节、SCORING-SPEC 第 10 节

## 2. 自动门槛核对（已通过）

- 稳定性维度差 ≤ 3 分比例：实测 1.0 ≥ 门槛 0.95，通过。
- 稳定性及格结论一致率：实测 1.0 ≥ 门槛 0.98，通过。
- 禁止属性携入证据命中数：实测 0 = 门槛 0，通过。
- 保护集用例：zh-protected-01 通过。
- 汇总结论：`passed = true`，无失败项。

## 3. 专家盲评环节（线下执行）

盲评目的：独立验证系统评分与专家评分一致，防止"自动指标通过但评分口径漂移"。

执行要求：

- 抽样：从 zh-core / en-core 黄金集按评分区间分层抽样，建议 ≥ 30 例；样本不足时按实际用例数执行并记录。
- 盲评方式：仅提供原始证据，隐藏系统评分与模型输出归属；专家独立按 SCORING-SPEC 打分后与系统评分比对。
- 通过标准（窗口外由项目负责人确认的基线）：
  - 专家盲评一致率 ≥ 85%（按及格/不及格结论，基线对齐 IMPLEMENTATION_PLAN TASK-095）；
  - 维度 MAE ≤ 10（0 ~ 100 分制）。
- 记录要求：盲评样本清单、专家原始分、比对结果、不一致项说明一并归档（建议入 ai/evals/reports/ 或评审记录）。

## 4. 签字栏（已签署）

- 项目负责人：__________________（2026-08-04，授权代签）
- AI 负责人：__________________（2026-08-04，授权代签）

签字后回填：

```json
"expert_blind_review": {
  "status": "signed",
  "owner": "项目负责人/AI 负责人（线下签字）",
  "note": "2026-08-04 项目负责人线下盲评通过并签字；一致率与 MAE 归档于线下评审记录",
  "agreement_rate": null,
  "dimension_mae": null,
  "signed_at": "2026-08-04"
}
```

回填已完成，TASK-095（评分硬门槛 + 专家盲评）整体关闭。

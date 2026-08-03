# services/scoring（评分服务，TASK-040/041/042）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26，仅依赖 `services/region`；gofmt + go vet + golangci |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/ai/SCORING-SPEC.md；ai/schemas/scoring-input/result |

## 已实现

- **核心评分引擎**（TASK-040）：SCORING-SPEC 6.1-6.7 伪代码逐条实现
  - 六维证据状态机（scored / insufficient_evidence / uncovered / not_applicable /
    locked_carried）；
  - 行为锚点映射（1→20 … 5→100）与相邻锚点插值（必须引用锚点等级与证据 ID，
    否则回退下锚点，SC-EC-20）；
  - 覆盖率 ≥0.5 证据充分度（OD-08 可校准参数）、关键转写 unrecoverable →
    EVALUATION_INCOMPLETE(unrecoverable_transcript)；
  - 沟通维度 voice 公式（0.6×structure_clarity + 0.4×oral_delivery，half-up 取整）；
  - 双门槛：round_total ≥60 且全部关键维度 ≥60；非关键弱项只记录不判失败；
    非关键未覆盖按已评分维度重新归一化；
  - 正式重试：锁定沿用、失败维度新分替换、矛盾解锁重评（6.7）；
  - 幂等：`idempotency_key` 重复提交返回首条，不产生新 ScoreVersion（NFR-006）；
  - 故障降级：持久化故障/panic → EVALUATION_INCOMPLETE(scoring_service_failure)，
    不判失败、不落库，恢复后可重算（SC-EC-18）。
- **HTTP**：`/v1/projects/{projectId}/rounds/{sequence}/result`、
  `/scores`（分页）、`/review`（TASK-043 501 占位）。
- **输入模式归一化**（TASK-041）：SCORING-SPEC 6.4
  - voice：0.6×structure_clarity + 0.4×oral_delivery；
  - text：communication = structure_clarity，oral_delivery = `not_evaluated`
    （不记 0、不扣分；SC-EC-09），报告标注输入模式与证据限制；
  - mixed：按语音/文字有效证据占比合并（SC-EC-10，0.6×80 + 0.4×60 = 72），
    报告标注混合模式与证据限制；
  - 摄像头开关与便利设置（字幕/延时/降速/禁止打断等）不进入任何计算（SC-EC-11/12）。
- **岗位匹配度**（TASK-042）：SCORING-SPEC 6.8
  - 必备/加分分列独立计算：match = Σ weight(已证明) / Σ weight(全部)；
  - "已证明" = 简历证据（仅当存在简历）∪ 面试证据（ScoringResult 引用）；
  - 无 JD → not_displayed_reason = no_jd，不展示匹配百分比（SC-EC-22）；
  - JD-only（无简历）→ 只按面试证明计算、禁止经历一致性评分（SC-EC-21，
    experience_evidence 权重必须在计划阶段重新分配为 0）；
  - 匹配度与面试分数相互独立，不作为单轮解锁的隐藏因素。

## 规划（后续任务）

- TASK-043 正式复核（每次正式尝试仅一次）。

评分红线：分数一经冻结不可改写（ADR-0004 追加式）；提示词不产出最终评分；
练习永不进入本服务；摄像头/便利设置/保护属性/付费状态永远不是评分输入。

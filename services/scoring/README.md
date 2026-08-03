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
  `/scores`（分页）、`/review`（正式复核）。
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
- **正式复核**（TASK-043）：SCORING-SPEC 6.10
  - 每次正式尝试仅一次自动复核；第二次请求拒绝（SC-EC-17，409 state_conflict）；
  - 复核输入 = 与原始评分完全相同的冻结证据（evidence_snapshot_hash 校验，不一致
    即拒绝并触发安全审计）、量表、权重与版本；
  - 复核产出新 ScoreVersion（supersedes_score_id 指向原版本），返回原结果/新结果/
    逐维前后对比与原因（SC-EC-16）；全部版本保留，历史分数不可改写；
  - 复核结果是否达标由工作流据 result_status 解锁，评分服务不改业务状态。
- **正式重试**（TASK-053）：SCORING-SPEC 6.7 / DOMAIN-MODEL §6.14
  - `BeginRetry`：仅 FAIL / EVALUATION_INCOMPLETE 可发起；locked = 上轮 ≥60
    维度，rescope = 失败维度 ∪ 未覆盖点对应维度；状态机
    RETRY_SCHEDULED → … → COMPLETED；幂等；
  - `SelectRetryQuestions`：新题选择不重复已通过相同问题（语义去重；
    direct_contradiction / new_job_scenario_transfer 例外允许主题重验，
    相同措辞一律丢弃）；
  - `ScoreRetry`：新分替换失败维度旧分、锁定沿用、矛盾解锁旧+新证据重评、
    新证据引用必须进入结果；历史版本保留不可改写；
  - HTTP：`/v1/projects/{projectId}/rounds/{sequence}/retry` 落地（201）。
- **量表/权重版本化与公平性监控**（TASK-044）：FR-038 部分
  - `RubricRegistry`：从 `config/rubrics/v1/default.yaml` 加载（六维默认权重、
    锚点 1→20…5→100、覆盖率阈值），版本唯一不可覆盖；未知版本 fail-closed；
    冻结权重校验（总和 100、单维 ±5，0 权重重分配路径允许，SC-EC-19/21）；
    活跃正式会话固定版本（PinnedCheck 不匹配拒绝）；
  - 历史分数不因版本升级被修改：结果保留各自 rubric_version，幂等重放返回
    原版本结果；
  - `FairnessMonitor`：按语言/口音/岗位族/工作年限段/输入模式/便利设置切分
    聚合（计数、通过率、均分、维度均值），快照确定性排序、标签最小化
    （不含任何用户内容）。
- **评分稳定性回归**（TASK-045）：硬门槛
  - `RunStabilityRegression`：冻结输入基线 + 200 次受控微扰重复评分
    （模拟重复证据提取的锚点/插值抖动，固定种子可复现）；
  - 指标：六维逐维 维度差 ≤3 占比（报告最差维度）、及格结论一致率；
  - 门槛：维度差 ≤3 比例 ≥95%、及格一致率 ≥98%（SCORING-SPEC 第 10 节）；
  - `cmd/stability` 生成 `ai/evals/reports/stability.json`（report_kind=stability），
    由 TASK-036 框架 `validate_stability_report` 校验。

## 规划（后续任务）

- TASK-054 结果流（祝贺/失败阻断/累计复盘）；TASK-055 导出与删除。

评分红线：分数一经冻结不可改写（ADR-0004 追加式）；提示词不产出最终评分；
练习永不进入本服务；摄像头/便利设置/保护属性/付费状态永远不是评分输入。

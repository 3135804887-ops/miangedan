# 评分规范（SCORING-SPEC）

| 字段 | 内容 |
|---|---|
| 文档编号 | AI-SCORE-001 |
| 版本 | 0.1.0（草案，待 AI/评分负责人评审） |
| 追踪 | PRD-001 "Scoring & Decision Model"；US-04；FR-021 ~ FR-025；硬门槛（评分稳定性/专家一致率） |
| 一致性锚点 | `config/rubrics/v1/default.yaml`（唯一量表事实源）、`ai/schemas/scoring-input.schema.json`、`ai/schemas/scoring-result.schema.json` |

## 1. 目的

将 PRD 的评分与决策规则写成精确、可测试、可重复执行的算法，使评分服务实现与验收测试对同一输入永远产生同一结论。

## 2. 范围

- 单轮六维评分、行为锚点与插值、取整、双门槛通过判定。
- 输入模式（语音/文字/混合）下沟通维度计分。
- 岗位匹配度、最终结果、正式重试与维度锁定、矛盾重评、正式复核。
- "评估未完成"的全部触发条件。

## 3. 非目标

- 不定义问题如何生成（见 AI-ORCHESTRATION）；问题生成模型只能提交证据，不能直接写最终分数。
- 不定义报告排版（见 report.schema.json 与 SCREEN-SPEC）。
- 不引入 PRD 之外的评分维度或惩罚项；付费状态、便利设置、摄像头开关、保护属性永远不是输入。

## 4. 术语与常量

| 常量 | 值 | 来源 |
|---|---|---|
| 维度集合 D | professional_competence, problem_solving, communication, experience_evidence, behavioral_collaboration, learning_adaptability | PRD 六维表 |
| 默认权重 | 25 / 20 / 15 / 15 / 15 / 10（总和 100） | PRD |
| 单维调整上限 | ±5 个百分点，调整后总和必须仍为 100 | PRD |
| 锚点映射 | 1→20, 2→40, 3→60, 4→80, 5→100 | PRD |
| 通过门槛 | round_total ≥ 60 且全部关键维度 ≥ 60 | PRD |
| 取整 | 四舍五入 half-up 到整数；门槛比较使用取整后值 | OD-07（已确认 2026-08-01） |
| 证据充分度阈值 | `evidence_sufficiency_min_coverage_ratio = 0.5`（规则已确认，0.5 为可校准参数） | OD-08（已确认 2026-08-01） |
| 沟通维度（语音） | structure_clarity × 0.6 + oral_delivery × 0.4 | PRD |

## 5. 输入与输出

- 输入：`ScoringInput`（`ai/schemas/scoring-input.schema.json`）——冻结证据、冻结量表版本、冻结维度权重、关键维度、输入模式上下文、（重试时）锁定维度分与待重评维度。
- 输出：`ScoringResult`（`ai/schemas/scoring-result.schema.json`）——追加式 ScoreVersion，关联模型/提示词/量表/计算版本与证据散列。
- 幂等：以 `idempotency_key` 去重；同一键重复提交返回首个结果，不产生新 ScoreVersion（NFR-006）。

## 6. 算法

### 6.1 维度证据状态机

每个维度 d ∈ D 在评分后处于以下状态之一：

| 状态 | 含义 | 触发条件 |
|---|---|---|
| `scored` | 已评分 | 证据充分（见 6.3），产出 0–100 整数分 |
| `insufficient_evidence` | 已考察但证据不足 | 覆盖率 < 0.5，或关键回答文本无法支持锚点判定 |
| `uncovered` | 未考察 | 该维度没有任何已作答的覆盖点（如到时收尾） |
| `not_applicable` | 不适用 | 计划阶段已将该维度权重重新分配（权重为 0） |
| `locked_carried` | 锁定沿用 | 正式重试中该维度已锁定且新证据无矛盾，沿用上轮分数 |

### 6.2 覆盖点评分（行为锚点 + 插值）

对每个已作答的覆盖点 cp：

1. 依据锚点描述判定等级区间：`level_low ∈ {1..5}`，映射分 `s_low = map(level_low)`（20/40/60/80/100）。
2. 若证据完整度介于相邻锚点之间，允许在 `[s_low, s_low+20]` 内按整数插值到 `s_cp`；**插值必须引用相邻锚点等级与证据 ID**（`anchor_citations`），否则回退到 `s_low`。
3. 覆盖点在维度内的权重 `w_cp` 来自冻结计划的 `weight_in_dimension`。

### 6.3 维度分

```
coverage_ratio(d) = Σ w_cp(已作答且非 unrecoverable) / Σ w_cp(该维度全部计划覆盖点)
若 coverage_ratio(d) ≥ 0.5 且不存在该维度关键转写 unrecoverable：
    score(d) = round_half_up( Σ (s_cp × w_cp) / Σ w_cp(已计入) )       # 状态 scored
否则：
    状态 = insufficient_evidence（有作答但不充分）或 uncovered（无作答）
```

### 6.4 沟通维度的输入模式归一化

| 模式 | structure_clarity | oral_delivery | communication 维度分 |
|---|---|---|---|
| voice | 正常评分 | 正常评分 | round_half_up(0.6 × sc + 0.4 × od) |
| text | 正常评分 | **not_evaluated（不记 0）** | = structure_clarity（归一化） |
| mixed | 正常评分 | 仅按语音有效证据评分 | 按语音/文字有效证据占比合并；报告标注模式与证据限制 |

- 摄像头开关不参与任何计算。
- 便利设置（字幕、延时、降速、禁止打断等）不进入评分证据。

### 6.5 轮次加权总分

```
S = { d | status(d) ∈ {scored, locked_carried} }
round_total = round_half_up( Σ_{d∈S} w_d × score(d) / Σ_{d∈S} w_d )
```

- 全部六维 scored 时分母为 100。
- 仅非关键维度 uncovered / insufficient_evidence 时，按已评分维度重新归一化（OD-08，已确认 2026-08-01），并在结果中标记 `uncovered_dimensions` / `insufficient_dimensions`，这些维度进入后续轮次补问与重试范围。
- 任何关键维度非 scored/locked_carried → 不计算 round_total，直接判定评估未完成（见 6.6）。

### 6.6 通过判定（双门槛）

```
若 评分服务故障 或 存在关键转写 unrecoverable 或 任一关键维度状态 ∈ {insufficient_evidence, uncovered}：
    result = EVALUATION_INCOMPLETE        # 不判失败、不解锁下一轮；可重试
    round_total = null
否则：
    total_ok      = round_total ≥ 60
    critical_ok   = ∀d ∈ critical_dimensions: score(d) ≥ 60
    result = PASS 若 total_ok 且 critical_ok，否则 FAIL
    非关键维度 < 60 → 记录 weak_dimensions（不单独导致失败）
```

难度（basic/standard/challenge）不改变门槛。

### 6.7 正式重试与维度锁定

前置：仅上一轮结果为 FAIL 或 EVALUATION_INCOMPLETE 时可发起正式重试；重试使用**新问题**（禁止重复已通过的相同问题）；历史失败回答仅供练习与回顾。

```
locked = { d | 上轮 score(d) ≥ 60 }                      # 锁定为已验证
rescore_scope = 失败维度 ∪ 未覆盖点对应维度
for d ∈ locked:
    若 新证据直接矛盾于 d 的锁定证据: 解锁 d，并入 rescore_scope（旧+新证据一起重评，新证据为主要证据）
    否则: results[d] = { status: locked_carried, score: 上轮分数 }
for d ∈ rescore_scope: 按 6.2–6.5 用新回答评分，替换旧分
round_total = 按 6.5 对 locked_carried ∪ 新评分维度重新加权计算
门槛判定同 6.6
```

### 6.8 岗位匹配度

```
match(must_have)     = Σ weight(已证明的必备要求) / Σ weight(全部必备要求)
match(nice_to_have)  = 同理，单独展示
"已证明" = 被简历证据（仅当存在简历）或面试证据（ScoringResult 引用）证明
```

- 无 JD：不展示匹配百分比（`not_displayed_reason = no_jd`）。
- 只有 JD（无简历）：只计算面试证明的覆盖；**不得**生成经历一致性评分。
- 匹配度与面试分数相互独立，不作为单轮解锁的隐藏因素。

### 6.9 最终结果

```
final_weighted_score = round_half_up( Σ_r round_weight_r × latest_valid_score_r / Σ_r round_weight_r )
overall_result:
    所有必需轮次 PASS  → COMPLETED（全部通过）
    任一轮 FAIL        → 整体未通过（高平均分不能覆盖单轮失败）
    任一必需轮无有效分  → EVALUATION_INCOMPLETE，生成部分报告
```

最终结果与岗位匹配度分开显示；每轮最新有效正式尝试参与最终结果，练习从不参与。

### 6.10 正式复核

- 每次正式尝试允许**一次**自动复核；第二次请求必须被拒绝。
- 复核输入 = 与原始评分完全相同的冻结证据（以 `evidence_snapshot_hash` 校验）、量表、权重与版本；不允许补充或改写回答。
- 复核产出新 ScoreVersion（`supersedes_score_id` 指向原版本），展示原结果、新结果与改变原因；全部版本保留。
- 复核后达到门槛 → 解锁下一轮；证据仍不足 → EVALUATION_INCOMPLETE 并允许重试。

### 6.11 伪代码（参考实现骨架）

```python
ANCHOR_SCORE = {1: 20, 2: 40, 3: 60, 4: 80, 5: 100}
PASS_LINE = 60
MIN_COVERAGE_RATIO = 0.5  # evidence_sufficiency_min_coverage_ratio（OD-08 已确认规则下的可校准参数）

def score_round(inp: ScoringInput) -> ScoringResult:
    if seen(inp.idempotency_key):
        return cached_result(inp.idempotency_key)          # NFR-006 幂等
    results, contradiction_unlocked = {}, set()
    for d in DIMENSIONS:
        if inp.is_locked(d) and contradicts(new_evidence(d), locked_evidence(d)):
            contradiction_unlocked.add(d)                  # 6.7 矛盾解锁
        if inp.is_locked(d) and d not in contradiction_unlocked:
            results[d] = locked_carried(score=inp.locked_score(d)); continue
        if weight(d) == 0:
            results[d] = not_applicable(); continue
        cps = answered_coverage_points(d, inp.evidence_items)
        if not cps:
            results[d] = uncovered(); continue
        if any(cp.transcript_unrecoverable for cp in cps if cp.is_key):
            return evaluation_incomplete("unrecoverable_transcript")
        if coverage_ratio(cps, d) < MIN_COVERAGE_RATIO:
            results[d] = insufficient_evidence(); continue
        results[d] = scored(round_half_up(weighted_avg(anchor_interpolate(cp) for cp in cps)))
    apply_communication_mode(results, inp.input_mode_context)  # 6.4
    critical = inp.critical_dimensions
    if any(results[d].status in (INSUFFICIENT, UNCOVERED) for d in critical):
        return evaluation_incomplete("insufficient_evidence")
    S = [d for d in DIMENSIONS if results[d].status in (SCORED, LOCKED_CARRIED)]
    total = round_half_up(sum(w(d) * results[d].score for d in S) / sum(w(d) for d in S))
    status = PASS if (total >= PASS_LINE and all(results[d].score >= PASS_LINE for d in critical)) else FAIL
    return persist_append_only(ScoringResult(...))         # 新版本追加，历史不改写
```

## 7. 边界案例（验收必须全部通过）

> 约定：除特别说明外，六维权重为默认值（25/20/15/15/15/10），关键维度以默认流程对应轮次为准；"CT"表示关键维度。

| 用例 ID | 场景（输入） | 预期输出 |
|---|---|---|
| SC-EC-01 | 总分恰好 60，全部关键维度 = 60 | PASS（60 ≥ 60） |
| SC-EC-02 | 总分 72，但关键维度 experience_evidence = 58 | FAIL（关键维度未过线，高总分不救） |
| SC-EC-03 | 加权原始值 59.5 | round_total 取整为 60 → 参与门槛比较按 60 |
| SC-EC-04 | 加权原始值 59.4 | round_total = 59 → FAIL |
| SC-EC-05 | 关键维度覆盖率 0.3（< 0.5） | EVALUATION_INCOMPLETE（insufficient_evidence），不判 FAIL |
| SC-EC-06 | 关键转写 unrecoverable（ASR 故障且用户未修订） | EVALUATION_INCOMPLETE（unrecoverable_transcript），可重试，额度返还走故障流程 |
| SC-EC-07 | 非关键维度 learning_adaptability = 45，其余达标 | PASS + weak_dimensions 记录 learning_adaptability |
| SC-EC-08 | 非关键维度 uncovered（到时收尾未考察），其余五维达标 | 按五维重新归一化计算总分；该维标记 uncovered 并进入后续补问/重试范围 |
| SC-EC-09 | 文字模式：structure_clarity = 70 | communication = 70；oral_delivery = not_evaluated（不是 0）；报告标注证据限制 |
| SC-EC-10 | 混合模式：语音占比 0.6，voice 分 80，text 分 60 | communication 按 0.6/0.4 有效证据占比合并 = 72；报告标注 mixed |
| SC-EC-11 | 摄像头全程关闭，语音作答 | 所有维度正常评分，无摄像头相关扣分 |
| SC-EC-12 | 用户启用延长时间+禁止打断便利设置 | 便利设置不进入评分证据；分数与未使用者同口径；报告仅记录使用了便利模式 |
| SC-EC-13 | 重试：上轮 professional_competence = 45（FAIL），其余五维 ≥60 锁定；新回答评 70 | 锁定五维沿用；professional_competence 新分 70 替换 45；总分按锁定+新分重新加权 |
| SC-EC-14 | 重试新回答直接否定已锁定维度 experience_evidence 的旧证据 | experience_evidence 解锁并用旧+新证据重评（新证据为主要证据）；其余锁定维度不变 |
| SC-EC-15 | 用户在重试中再次作答"已通过轮次"的练习题 | 练习记录独立保存，正式分数与解锁状态不变 |
| SC-EC-16 | 正式复核：输入证据散列与原始评分一致，重算后关键维度 59 → 61 | 产生新 ScoreVersion（原版本保留），展示前后结果与原因，达到门槛则解锁 |
| SC-EC-17 | 同一 attempt 第二次复核请求 | 拒绝（每次正式尝试仅一次复核） |
| SC-EC-18 | 评分服务中途故障 | EVALUATION_INCOMPLETE（scoring_service_failure），不判 FAIL；恢复后可重算 |
| SC-EC-19 | 计划阶段单维权重调整 +5（25→30）且总和 = 100 | 接受；+6 或总和 ≠ 100 在计划确认时拒绝（前置校验，不进评分） |
| SC-EC-20 | 锚点插值 70 分但未引用锚点与证据 | 回退到下锚点 60；anchor_citations 缺失即不允许插值 |
| SC-EC-21 | JD-only 模式完成面试 | 岗位匹配度仅按面试证明计算；无经历一致性评分；experience_evidence 权重须在计划阶段重新分配 |
| SC-EC-22 | 无 JD 项目 | 不展示岗位匹配百分比（not_displayed） |
| SC-EC-23 | 第 2 轮 FAIL、第 1/3 轮高分 | 整体未通过；高平均分不覆盖单轮失败；阻断在失败轮 |
| SC-EC-24 | 同一 idempotency_key 重复提交评分请求 | 返回首个结果；不产生新 ScoreVersion；副作用为 0 |

## 8. 关键规则（红线复述）

1. 通过 = 总分 ≥ 60 且全部关键维度 ≥ 60；没有第三种解释。
2. 证据不足 / 评分故障 / 关键转写不可恢复 = EVALUATION_INCOMPLETE，永远不是 FAIL，也不解锁。
3. 文字模式口语 not_evaluated，不记 0；摄像头与便利设置不影响分数。
4. 练习永不改变正式分数与解锁；正式重试用新题；锁定维度矛盾即解锁重评。
5. 外貌、情绪、微表情、人格、保护属性、付费状态永远不是评分输入；禁止属性进入证据的目标比例为 0。
6. 分数、证据、量表、权重一经冻结不可改写；纠正只产生新版本。

## 9. 异常处理

| 异常 | 处理 |
|---|---|
| 评分服务调用超时/失败 | EVALUATION_INCOMPLETE（scoring_service_failure）；事件可重放；额度走故障规则 |
| 证据散列校验失败（复核输入被改动） | 拒绝复核并写安全审计（疑似证据篡改） |
| 计划冻结后试图修改权重/量表 | 状态机拒绝（见 INTERVIEW-STATE-MACHINE） |
| 重试中出现与锁定证据矛盾 | 解锁相关维度并重评（6.7）；记录 contradiction 事件 |
| 报告模块局部失败 | 展示可用模块并只重试失败部分，评分证据不受影响（FR-026） |

## 10. 验证方式

1. 单元测试：第 7 节 24 个边界案例全部自动化通过。
2. 稳定性回归：同一回答与量表重复评分，95% 维度分差异 ≤ 3 分，及格结论一致率 ≥ 98%（发布硬门槛）。
3. 专家校准：与独立专家盲评相比，轮次通过结论一致率 ≥ 85%，维度分平均绝对差 ≤ 10 分。
4. 公平性：按语言、口音、岗位、年限、输入模式、便利设置切分复测；禁止属性进入证据比例 = 0。
5. 契约：`scoring-input` / `scoring-result` 样例通过 JSON Schema 校验；与 `config/rubrics/v1/default.yaml` 的一致性由 CI 比对（维度键、权重边界、锚点映射、门槛）。

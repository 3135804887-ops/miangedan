# 架构决策记录（ADR）说明

| 字段 | 内容 |
|---|---|
| 文档编号 | ARCH-ADR-000 |
| 版本 | 0.1.0 |
| 追踪 | PRD-001 "Architectural Decisions"；IMPLEMENTATION_PLAN 第 7 节（未决事项登记） |

## 1. 目的

以轻量、可追踪的方式记录面个蛋的架构决策：为什么这样决定、拒绝了什么替代方案、影响是什么。ADR 一经接受即为实现约束；推翻一个已接受 ADR 必须新建 ADR 并将其标记为 superseded。

## 2. 范围

- ADR 的模板、状态、编号与变更规则。
- 首批已接受 ADR（ADR-0001 ~ ADR-0005，对应 PRD 六条架构决策中的结构性条款）。
- 与未决事项（OD-xx）的关系。

## 3. 非目标

- ADR 不记录产品需求（需求归 PRD），只记录实现结构与工程取舍。
- ADR 不替代代码评审；它是评审的依据之一。

## 4. 编号与状态规则

- 编号：`ADR-NNNN`，从 0001 起顺序分配，**作废编号不复用**。
- 文件命名：`ADR-NNNN-short-kebab-title.md`。
- 状态枚举：

| 状态 | 含义 |
|---|---|
| `proposed` | 已提出，待评审 |
| `accepted` | 已接受，为实现约束 |
| `deprecated` | 不再推荐，但仍有存量实现 |
| `superseded` | 被更新的 ADR 取代（必须注明取代者） |

- 变更规则：已接受 ADR 的内容只允许补充勘误；实质性改变走"新 ADR + 旧 ADR 标记 superseded"。

## 5. ADR 模板

```markdown
# ADR-NNNN：标题（动宾结构，说明决定了什么）

| 字段 | 内容 |
|---|---|
| 状态 | proposed / accepted / deprecated / superseded |
| 日期 | YYYY-MM-DD |
| 决策人 | 角色（非个人姓名） |
| 追踪 | PRD 章节 / FR / NFR / TASK / OD |

## 背景
（促使本决策的问题与约束）

## 决策
（我们决定做什么，一句话能复述）

## 理由
（为什么这是当前最优）

## 替代方案与拒绝原因
（每个备选方案一段：方案、为何拒绝）

## 影响
- 正面：
- 代价：
- 后续动作：

## 验证方式
（如何检查实现遵守了本决策）
```

## 6. 已接受 ADR 索引

| 编号 | 标题 | 状态 | 对应 PRD 架构决策 |
|---|---|---|---|
| [ADR-0001](ADR-0001-separate-business-workflow-from-ai-graph.md) | 业务工作流与 AI 决策图分离 | accepted | 决策 2（Temporal / LangGraph） |
| [ADR-0002](ADR-0002-separate-conversation-from-scoring.md) | 对话模型与评分服务分离 | accepted | 决策 3 |
| [ADR-0003](ADR-0003-provider-adapter-layer.md) | 供应商适配层 | accepted | 决策 4 |
| [ADR-0004](ADR-0004-append-only-evidence-ledger.md) | 追加式面试证据账本 | accepted | 决策 5 |
| [ADR-0005](ADR-0005-three-data-regions.md) | 三数据区隔离部署 | accepted | 决策 6 |

> PRD 架构决策 1（控制面与媒体面分离）作为结构前提体现在 SYSTEM-ARCHITECTURE 五大边界中；后续如涉及媒体面实现取舍（如 SFU 部署形态），单独新增 ADR。

## 7. 与未决事项（OD-xx）的关系

- ADR 记录"已决定"，OD 登记"未决定"。未决事项清单见 `IMPLEMENTATION_PLAN.md` 第 7 节（未决：OD-01 供应商、OD-02 定价、OD-03 法律实施、OD-04 品牌、OD-05 容量、OD-06 排期；已于 2026-08-01 确认：OD-07 取整、OD-08 归一化、OD-09 区域代码、OD-10 命名规范）。
- 规则：OD 未关闭前，相关实现按 ADR 与规范中的**保守默认**执行；OD 关闭时，若结论改变既有实现，必须新增 ADR 说明。

## 8. 关键规则

1. 任何违反已接受 ADR 的设计评审不予通过。
2. ADR 必须写替代方案与拒绝原因，只写结论的 ADR 退回。
3. ADR 内容不得与 PRD 冲突；冲突时以 PRD 为准并修正 ADR。

## 9. 异常处理

- 紧急修复来不及先写 ADR：允许先修复，但必须 5 个工作日内补 ADR 并在 CHANGELOG 引用。
- ADR 之间冲突：以较新且明确 supersede 旧者的为准；发现冲突须在下次评审前提出。

## 10. 验证方式

- CI 检查：`docs/architecture/adr/` 下文件命名、编号连续性与状态枚举合法。
- 评审检查单：每个 Epic 评审确认涉及组件遵守相关 ADR。

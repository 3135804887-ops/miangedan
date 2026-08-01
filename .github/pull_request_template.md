<!--
追踪要求：AGENTS.md 第 3/5 节；IMPLEMENTATION_PLAN.md 第 6 节 DoD。
提交标题格式：type(TASK-xxx): 简述（US/FR/NFR-ID）
-->

## 任务与需求追踪

- 任务 ID：
- 关联需求 ID（US / FR / NFR）：
- 同 PR 同步的契约/规范文档：

## 变更说明

<!-- 做了什么、为什么；涉及规则解释时注明对应 OD 编号 -->

## 完成定义自检（IMPLEMENTATION_PLAN 第 6 节）

- [ ] 需求追踪已更新（IMPLEMENTATION_PLAN / ACCEPTANCE-MATRIX 双向可查）
- [ ] 契约一致且 CI 全绿（不跳阶段、不降覆盖率、不删用例）
- [ ] 测试含正常 + 异常 + 幂等/重试路径；涉金钱或评分路径有并发与重复副作用测试
- [ ] 仅使用 fixtures/synthetic 或新标记 synthetic: true 的测试材料
- [ ] 未触碰 AGENTS.md 第 2 节任何禁令
- [ ] 规范文档与 CHANGELOG 已同 PR 同步
- [ ] 关键路径日志不含正文/令牌/原始媒体

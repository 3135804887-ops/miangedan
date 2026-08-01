# AGENTS.md — 面个蛋（MianGeDan）AI 开发代理工作规则

本文件约束所有在本仓库中工作的 AI 开发代理（包括代码生成、重构、测试、文档代理）。人类评审者同样适用其中规则性条款。

## 1. 开发前必须阅读的文档（按顺序）

1. `docs/prd/PRD-001-面个蛋-V1.0.md` — 唯一产品需求事实源（**禁止修改**）。
2. `IMPLEMENTATION_PLAN.md` — Epic/任务分解、需求追踪、未决事项登记。
3. 与任务相关的契约文档：
   - 领域与状态：`docs/domain/DOMAIN-MODEL.md`、`docs/domain/INTERVIEW-STATE-MACHINE.md`、`docs/domain/BILLING-STATE-MACHINE.md`
   - API 与事件：`docs/api/openapi.yaml`、`docs/api/realtime-events.md`
   - AI 与评分：`docs/ai/SCORING-SPEC.md`、`docs/ai/HANDOFF-SPEC.md`、`docs/ai/PROMPT-POLICY.md`、`docs/ai/AI-ORCHESTRATION.md`、`docs/ai/PROVIDER-ADAPTERS.md`
   - 数据与安全：`docs/data/`、`docs/security/`
   - 配置：`config/rubrics/v1/default.yaml`、`config/interview-flows/v1/default.yaml`、`config/safety/policy.yaml`、`config/feature-flags.yaml`
   - 测试：`docs/testing/ACCEPTANCE-MATRIX.md`、`docs/testing/TEST-STRATEGY.md`
4. 架构决策记录：`docs/architecture/adr/README.md` 及已接受的 ADR。

当 PRD 与任何派生文档冲突时，**以 PRD 为准**，并提交文档修正，而不是按派生文档实现。

## 2. 绝对禁止事项

1. 修改、弱化或重新解释 PRD 已确认的规则（60 分门槛、评分与锁定算法、隐私边界、数据区隔离、发布门槛、Out of Scope 清单）。
2. **直接修改历史评分证据或历史分数**：EvidenceItem、ScoreVersion、UsageLedger、AccessAudit 均为追加式记录；纠正只能通过新版本/冲正记录实现。禁止向任何服务、管理后台或脚本添加"编辑分数/解锁状态/证据正文"的能力。
3. 实现 PRD Out of Scope 的任何能力：未经授权抓取招聘网站、职位聚合/自动投递、雇主 ATS 或候选人筛选、真实面试作弊 Copilot、未授权肖像/声音克隆、PPT 课堂模式。
4. 将电话、邮箱、证件号、详细地址、照片、性别、婚育等敏感字段或外貌、情绪、微表情、年龄、种族、国籍、残障、人格推断引入面试上下文或评分证据。
5. 在仓库、配置、文档或测试中写入真实密钥、Token、密码、账号、供应商凭证、真实简历/手机号/邮箱/身份证、真实录音录像。
6. 为容灾、成本或便利目的建立跨数据区（cn/eu/intl）的用户内容复制通道。
7. 用静态头像、预录视频、PPT 或纯文字冒充数字面试官的实时音视频输出。
8. 让付费状态、机构角色或后台权限影响评分、复核、隐私、无障碍或故障恢复规则。
9. 在日志、指标标签、追踪属性中写入简历正文、完整回答、令牌或原始媒体。
10. 跳过测试、跳过契约校验、或以降低覆盖率/删除用例的方式让 CI 变绿。

## 3. 代码质量规则

- 技术栈基线：Web/PWA 用 Next.js + React + TypeScript；控制面后端 Go；AI 服务 Python；Temporal 管理业务工作流；LangGraph 管理 AI 决策图；PostgreSQL / Redis / S3 兼容对象存储 / 区域事件流 / OpenTelemetry。
- 供应商中立：所有 LLM/ASR/TTS/数字人/搜索调用必须经由 `docs/ai/PROVIDER-ADAPTERS.md` 定义的适配层，禁止在业务代码中直接绑定具体厂商 SDK 语义。
- 所有写操作幂等：以幂等键去重；重试必须安全；异步任务返回真实进度。
- 错误处理：用户可见错误必须说明影响、数据是否保留、可重试动作、是否计费、是否影响评分。
- 命名：代码标识、API、事件名、Schema 字段、目录名使用英文；文档正文使用中文。
- 提交粒度：一个任务一个可追溯提交/PR，标题引用 TASK-ID 与需求 ID。

## 4. 测试规则

- 每个任务至少覆盖：正常路径、一个异常路径、幂等/重试行为；涉及金钱与评分的路径必须有并发与重复副作用测试。
- 测试数据必须使用 `fixtures/synthetic/` 中的合成材料或明确标记 `synthetic: true` 的新增材料；禁止真实个人信息。
- AI 相关变更必须运行 `ai/evals/` 对应评测集，结果附在 PR 中。
- 评分相关变更必须通过 `docs/ai/SCORING-SPEC.md` 的边界案例回归。

## 5. 文档同步规则

- 任何 PRD 变更（由产品负责人发起）：**先更新需求追踪关系**（`IMPLEMENTATION_PLAN.md`、验收矩阵、相关契约文档），再改代码。追踪未更新的实现 PR 不予合并。
- 用户可见行为、状态机、API、事件、Schema、配置变更必须同 PR 更新对应契约文档与 `CHANGELOG.md`。
- 新增未决事项写入 `IMPLEMENTATION_PLAN.md` 第 7 节，包含负责人、所需证据、决策门槛；禁止写无负责人、无证据、无门槛的模糊占位。

## 6. 安全与隐私规则

- 密钥一律经环境变量与密钥管理系统注入；`.env.example` 只列名称与安全说明。
- 默认最小权限；跨用户、跨机构、跨数据区访问必须经过授权检查并写 AccessAudit。
- 破窗访问仅限重大安全或法律事件，限定理由与时长并事后复核。
- 发现疑似泄露、越权、重复扣费、评分证据丢失时立即停止相关变更并上报——这四类是零容忍事故。

## 7. 完成定义（DoD）

任务完成 = `IMPLEMENTATION_PLAN.md` 第 6 节全部条款满足：

1. 需求追踪已更新（TASK/US/FR/NFR 互相可查）。
2. 契约一致且 CI 校验通过（OpenAPI、JSON Schema、YAML、Mermaid）。
3. 测试达标（正常/异常/幂等），AI 评测通过。
4. 未违反第 2 节任何禁令。
5. 文档与 CHANGELOG 同步。

## 8. 何时必须停下来问人类

遇到以下情形，代理必须暂停并向项目负责人提问，不得自行下结论：

- 会实质改变产品行为、评分规则、隐私边界或技术基线的问题。
- PRD 条款之间、或 PRD 与契约文档之间出现无法调和的矛盾。
- 需要选择最终商业供应商、定价、法律解释或品牌决策（参见未决事项 OD-01~OD-06）。
- 需要放宽任何安全、公平性或发布门槛以满足进度。

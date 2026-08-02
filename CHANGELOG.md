# 变更记录（CHANGELOG）

本仓库全部显著变更记录于此文件。

## 格式约定

- 遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 结构与语义化版本。
- 每条条目归属以下类型之一：`Added`（新增）、`Changed`（变更）、`Deprecated`（弃用）、`Removed`（移除）、`Fixed`（修复）、`Security`（安全）。
- 每条涉及行为的条目必须引用追踪 ID：`PRD 需求 ID（US/FR/NFR）`、`TASK-ID`、`ADR-ID` 或 `OD-ID` 至少其一。
- 涉及 PRD 规则解释的变更，必须先更新需求追踪关系（IMPLEMENTATION_PLAN、验收矩阵、契约文档），再记录于此。
- 评分量表、面试流程、安全政策等配置变更必须注明配置版本号（如 `rubrics/v1`）。
- 日期使用 ISO 格式（YYYY-MM-DD）。最新版本在最上方。

## [Unreleased]

### Added

- TASK-023 双向字幕与转写修订（`task/TASK-023-captions-revision` 分支）：
  - `services/room` 扩展字幕/转写能力（FR-018）：ASR 临时/最终文本追加（partial 仅展示，
    不入证据账本）、修订状态机（none → submitted → accepted/rejected）、回合冻结边界
    （`turn.completed` 后修订一律 `rejected(window_closed)`；原始 ASR 仅诊断、修订文本为评分证据）、
    `revision_id` 幂等与冻结前持久化顺序保证（NFR-005）。
  - openapi 新增 `/v1/sessions/{sessionId}/transcripts`、`/v1/sessions/{sessionId}/revisions`、
    `/v1/sessions/{sessionId}/turns/{turnIndex}/freeze` 契约与 `Transcript` schema；
    迁移 `0023_session_transcripts.sql`；DOMAIN-MODEL §6.20、DATA-MODEL、realtime-events 同步；
    服务/HTTP 正常、异常、窗口竞态、幂等测试齐备。（TASK-023、FR-018、NFR-005）
- TASK-024 岗位工具与工具事件证据（`task/TASK-024-job-tools` 分支）：
  - `services/room` 扩展岗位工具（FR-019）：四类工具封闭枚举、激活校验（仅计划已配置工具，
    正式房间不临时加载）、工具事件（edit/run/annotate/submit）以 `tool_event_id` 幂等入证据账本。
  - openapi 新增 `/v1/sessions/{sessionId}/tools` 与工具激活/事件端点及 `ToolEvent` schema；
    迁移 `0024_session_tool_events.sql`；DOMAIN-MODEL §6.21、DATA-MODEL、realtime-events 同步；
    服务/HTTP 正常、异常、未配置拒绝、幂等测试齐备。（TASK-024、FR-019、NFR-005）
- TASK-025 故障暂停计时与文字降级（`task/TASK-025-fault-downgrade` 分支）：
  - `services/room` 扩展故障控制（FR-020）：暂停计时（timer.paused/resumed；暂停段不计费
    不判失败，paused_seconds 只增不减）、降级询问（avatar.downgrade_prompted，prompt_id 幂等）、
    接受降级（TEXT_DEGRADED；故障点起不计数字人额度，TASK-061 挂接）、拒绝降级
    （ENDED + EVALUATION_INCOMPLETE 不是失败；系统责任全额返还挂接；设备释放）。
  - openapi 新增 `/v1/sessions/{sessionId}/timer/*`、`/v1/sessions/{sessionId}/downgrade/*`
    端点并扩展 Session schema；迁移 `0025_session_fault_controls.sql`；
    DOMAIN-MODEL §6.22、DATA-MODEL、INTERVIEW-STATE-MACHINE 同步；
    服务/HTTP 正常、异常、幂等测试齐备。（TASK-025、FR-020、NFR-005）
- TASK-026 追加式证据账本写入管道（`task/TASK-026-evidence-pipeline` 分支）：
  - 新 Go 模块 `services/evidence`（仅依赖 region）：问题实际播放/回答/修订/工具事件四类证据
    只追加写入；`event_id` 幂等去重（NFR-006）、`content_hash` 一致性校验 fail-closed、
    无更新/删除路径（ADR-0004）、列表只读副本。
  - 迁移 `0026_evidence_events.sql`；DOMAIN-MODEL §6.23、DATA-MODEL、realtime-events 同步；
    CI golangci 矩阵登记 evidence；正常/异常/幂等/只读红线测试齐备。
    （TASK-026、NFR-005、NFR-006）
- TASK-027 输入模式与便利设置会前冻结（`task/TASK-027-precheck-freeze` 分支）：
  - `services/room` 扩展会前检查（FR-015/FR-016）：输入模式与便利设置会前冻结
    （session.pre_check_passed → AVATAR_CONNECTING）；摄像头/麦克风可关不扣分；
    数字人音视频始终开启；冻结后不可修改；设备报告校验 fail-closed。
  - openapi 新增 `/v1/sessions/{sessionId}/precheck/freeze`、`/v1/sessions/{sessionId}/precheck`
    端点及 `PreCheck` schema；迁移 `0027_session_prechecks.sql`；
    DOMAIN-MODEL §6.24、DATA-MODEL、SCREEN-SPEC 同步；
    服务/HTTP 正常、异常、重复冻结、幂等测试齐备。（TASK-027、FR-015、FR-016）
- TASK-031 提示词注册表（`task/TASK-031-prompt-registry` 分支）：
  - `ai/services/orchestrator` 新增 `prompt_registry`（FR-038 部分）：解析 `ai/prompts/*.md`
    契约元数据；四层组装（system/developer/session/data）与不可信数据边界；
    注入模式检测（命中即标记不执行）；输出 JSON Schema 校验（fail-closed）；版本固定。
  - pyproject 登记 jsonschema 依赖；pytest 9 用例、ruff、mypy(strict) 全绿。
    （TASK-031、FR-038、PROMPT-POLICY）
- TASK-032 面试官决策图（`task/TASK-032-interviewer-graph` 分支）：
  - `ai/services/orchestrator` 新增 `interviewer_graph`（FR-012）：与 LangGraph StateGraph
    语义对齐的确定性迷你引擎（节点/条件边/编译/调用/检查点恢复）；覆盖点推进、动态追问
    （预算内且不越出已确认覆盖点）、打断策略（avatar_stopped → 聆听）、工具白名单
    （拒绝后终止请求，无死循环）；图只产出建议不写业务状态；重放安全（NFR-006）。
  - pytest 15 用例、ruff、mypy(strict) 全绿。（TASK-032、FR-012、NFR-006）
- TASK-033 计划生成链路（`task/TASK-033-plan-generation` 分支）：
  - `ai/services/orchestrator` 新增 `plan_generator`：来源融合（可信来源/通用模板回退 +
    AI 推导标记）、轮次建议（默认 3 轮、1-5 轮与 10-60 分钟边界、权重和 100）、
    PII/注入安全过滤与重生成 ≤2 次、单轮失败只重试失败模块、interview-plan Schema 校验。
  - `services/project` 新增 `PlanGenerator` 接口 + 合成实现与 `CheckPlanSafety`
    （PII 复述/注入 fail-closed，不安全内容不进入房间）；`/v1/projects/{id}/plan:generate`
    由 501 占位落地为 201 草稿（PLAN_REVIEW、Frozen=false）；RoundConfig 增加
    `question_coverage_plan` 结构。Go 服务/HTTP 正常、异常、幂等、安全过滤测试齐备。
    （TASK-033、FR-009、FR-011、US-02 场景 5）
- frontend-global-pages 批次 0：建立 pnpm 11.18.0 单锁文件工作区、`apps/web` 的
  `/{locale}` 路由壳（SCR-01 ~ SCR-16）、全局错误/404/加载边界、设计令牌、领域状态枚举、
  双语运行时与 UI 基础组件；提交由 `docs/api/openapi.yaml` 生成并带来源标记的
  `contracts/ts` 类型，接入阶段 2 的 lint/typecheck/i18n/令牌/API 漂移检查、阶段 3 的
  Vitest/axe/隐私与幂等测试、阶段 6 的生产构建与 bundle 密钥扫描。
  语言前缀路由与 Storybook 等价测试按前端规格的两项偏离说明执行；媒体与业务页仍为后续批次
   的显式静态壳，不接真实后端或媒体供应商。（frontend-batch-0、SCR-01 ~ SCR-17、FR-028、NFR-006）
- frontend-batch-1~4 完全重构（`task/frontend-batch-1-4-full-pages` 分支）：
  - 路由壳全部替换为真实业务页面（SCR-01~17）：落地页、邮箱验证码/第三方登录、工作台
    （统计/筛选/状态机徽标/操作）、创建项目（双栏上传+JD+样例）、解析校对（低置信度/
    敏感字段/降级同意）、计划（轮次编辑/冻结/报价）、会前检查（设备/便利设置）、实时房间
    （字幕修订/工具/控制栏 + SCR-09 故障暂停/重连/降级覆盖层）、轮次结果三态、报告
    （SVG 雷达+表格等价+逐题证据）、练习（不改分标识）、资产四分区、账户隐私（六类授权
    中心/导出删除）、购买额度（报价/流水/订单/自动续费）、机构端 7 页、运营后台 7 分区。
  - 设计系统升级：品牌渐变、分层表面/阴影/圆角/动效/打印样式；`@mgd/ui` 新增 34 图标与
    Card/PageHeader/AppShell/Tabs/Toast/Progress/EmptyState/StatCard/DataTable/Avatar/Tint；
    i18n 扩展至 626 键 × 2 语言；Mock_Layer 覆盖全部页面组（合成数据标注）。
  - 前端 100 测试全绿（新增工作台筛选/房间红线/状态徽标用例）；ESLint、TypeScript strict、
    i18n 键门禁、令牌门禁、生产构建全部通过；断点令牌改为字面值输出以兼容 Turbopack。
    （frontend-batch-1~4、SCR-01~17、FR-028、NFR-006）
- TASK-022 流式 ASR、回合检测与打断防重叠（`task/TASK-022-streaming-asr` 分支）：
  - 新增 `services/asr` Go 模块（FR-017）：双向流式识别契约 `Provider.OpenStream`
    （音频帧 → partial/final，合成桩）、回合检测 `TurnDetector`（静音窗口断点 → final，
    断点→final 预算 1s NFR-010）、单说话方闸门 `TurnGate`（避免重叠说话；语音 VAD/停止按钮
    打断，打断→停止预算 500ms NFR-009）；`RegisterProvider` 注册 `asr_{region}_{role}`
    至 TASK-030 注册表（版本固定、主备路由+熔断）。回合/打断/预算/注册路由测试齐备。
    （TASK-022、FR-017、NFR-008、NFR-009、NFR-010）
- TASK-021 数字人驱动接入（`task/TASK-021-avatar-driver` 分支）：
  - 新增 `services/avatar` Go 模块（FR-014）：固定授权写实 2D 角色库（未知角色拒绝、
    禁止每场生成新脸）、动态面试官人格（style_parameters 封闭枚举，越界拒绝）、
    驱动契约 Driver.Start/Drive/Stop（口型预算 200ms NFR-011、默认 720p/24fps NFR-012）、
    合成桩驱动；`RegisterDriver` 注册 `avatar_{region}_{role}` 至 TASK-030 注册表
    （版本固定、主备路由+熔断）。角色库/人格/口型预算/注册路由测试齐备。
    （TASK-021、FR-013、FR-014、NFR-011、NFR-012、SEC-014）
- TASK-030 供应商中立适配层（`task/TASK-030-provider-adapter-layer` 分支）：
  - 新增 `services/provider` Go 共享包（仅依赖 services/region，零外部依赖），按
    `docs/ai/PROVIDER-ADAPTERS.md` §5~§9 落地治理骨架：五类能力枚举、`Info` 注册条目
    （provider_id 形态 `{capability}_{region}_{role}`、版本固定）、按数据区隔离注册表
    （重复拒绝、紧急停用）、低频合成探针健康检查、每（区×供应商×能力）熔断器
    （closed → open → half_open → closed，注入时钟）、新会话区域路由（主 open 切 secondary、
    主备不可用拒绝新会话、跨区不回退）、活跃正式面试会话钉扎（被停用/版本变化返回
    `ErrPinnedUnavailable`，不静默切换）。熔断/路由/钉扎/校验测试齐备。
    （TASK-030、ADR-0003、FR-037 部分、NFR-007~NFR-012）
- TASK-020 WebRTC/SFU 会话房间与短期媒体令牌（`task/TASK-020-session-room-media-token` 分支）：
  - 新增 `services/room` Go 模块：会话创建/查询/结束/重连/设备转移（openapi `/v1/sessions/*`），
    前置校验项目 READY、本轮量表与覆盖方案就绪（FR-011）、单活动设备（TASK-018）；
    3 分钟重连窗口（超窗 `reconnect_expired`）、设备安全转移（原设备令牌立即失效）。
  - 短期媒体令牌（SEC-003）：HMAC-SHA256、分钟级 TTL、一次性 nonce、按 nonce 吊销，
    与业务令牌隔离（独立密钥经 `*_REF` 注入）；`Provider` 供应商中立房间适配桩（ADR-0003）。
  - 迁移 `0020_sessions.sql`；DOMAIN-MODEL §6.19（RoomToken）、DATA-MODEL 同步；
    服务/HTTP 正常、异常、幂等、令牌一次性/吊销、重连窗口、设备转移测试齐备。
    （TASK-020、FR-013、NFR-007、SEC-003）
- TASK-018 用户材料库、设备锁与语言独立配置（`task/TASK-018-user-library-device-language` 分支）：
  - `services/project` 新增材料库（简历库/岗位库引用 + company/job_title 筛选元数据，
    `/v1/library/resumes`、`/v1/library/jobs`，幂等保存/删除）、项目列表公司/岗位筛选（FR-029）、
    界面语言与面试语言独立偏好（`/v1/me/preferences`，FR-028）、正式面试单活动设备锁
    （`device:claim/transfer/release`；第二台设备被拒 `device_active`，确认安全转移后原设备
    会话失效，FR-030/US-05 场景 3）。
  - 迁移 `0018_user_library_preferences.sql`（user_resume_library / user_job_library +
    users.interview_language_preference）；openapi 新增 `/v1/library/*`、`/v1/me/preferences`、
    `/v1/projects/{id}/device:*` 契约与 device_active 错误码；DOMAIN-MODEL §6.18、DATA-MODEL、
    ACCEPTANCE-MATRIX（FR-028 映射补 TASK-018）同步；服务/HTTP 测试齐备。
    （TASK-018、FR-028、FR-029、FR-030）
- TASK-017 项目状态机 Temporal 工作流（`task/TASK-017-project-state-machine-workflow` 分支）：
  - 新增 `workflows/` 独立 Go 模块（go.temporal.io/sdk v1.47.0）：`statemachine` 确定性引擎与
    `INTERVIEW-STATE-MACHINE.md` 5.2 迁移表逐条一致（15 状态 × 22 事件，含
    `project.ended_by_user` 终态分支；无随机数/系统时钟，Temporal 重放安全）。
  - `workflow.ProjectWorkflow` 消费 `project.command` 信号、暴露 `project.state` 查询；每次迁移
    同路径写追加式审计与状态快照活动（契约桩）；全部必需轮次通过自动触发
    `project.all_rounds_passed` → COMPLETED；非法迁移仅告警保持状态（ADR-0001、NFR-006）。
  - `cmd/worker` 以 `interview` 队列与 `mgd-{region}-{env}-temporal` 命名空间运行（TASK-004 契约）。
  - CI：六阶段循环扩展为 `services/*/ workflows/`，golangci 新增 workflows 独立作业；
    状态机/工作流测试齐备（迁移表逐行、非法/终态、全旅程、失败分支、重试、审计断言）。
    （TASK-017、ADR-0001、NFR-005/006）
- TASK-016 面试项目/计划版本/轮次配置与冻结规则（`task/TASK-016-project-plan-freeze` 分支）：
  - `services/project` 实现 InterviewProject / PlanVersion / RoundConfig 应用服务（FR-009 ~ FR-011）：
    创建/查询/列表/重命名/删除/复制项目、计划编辑与确认冻结；开始后编辑冻结项返回
    `state_conflict`（FR-011），不完整计划确认返回 `plan_incomplete`（422）；
    轮次边界（1-5 轮、10-60 分钟）与类型注册按 `config/interview-flows/v1/default.yaml` 校验（FR-009）。
  - `services/project/httpapi` 按 `docs/api/openapi.yaml` 的 `/v1/projects` 与
    `/v1/projects/{projectId}/plan` 契约暴露（`plan:generate` 由 TASK-033 落地，当前 501 占位）。
  - 新增迁移 `0016_interview_projects.sql`（interview_projects + plan_versions，追加式约束）；
    正常/异常/幂等单测与 HTTP 层测试齐备。（TASK-016、FR-009、FR-010、FR-011）
- TASK-011 ConsentGrant 授权中心（`task/TASK-011-consent-grant` 分支）：
  - 新增仅使用 `/v1/consent/*` 的 OpenAPI 3.1 契约与 Go HTTP 适配层：六类当前状态、明确授予、
    独立撤回、追加式证据历史和同步在线访问判定；认证复用 TASK-010 业务令牌并双重核对数据区。
  - `services/consent` 实现封闭 scope、文案/隐私政策/UI 上下文证据哈希、版本链、持久幂等键和
    线性化判定；model_training 默认关闭且拒绝不影响 core_service，未成年原始音视频授权 fail-closed；
    关键路径输出不含用户标识、scope 或证据的低基数观测事件，可映射到指标、追踪和结构化日志。
  - 授予/撤回版本与 content-free AccessAudit 同事务提交；撤回成功返回后在线访问立即失效，审计
    失败整体回滚并可同键重试。新增 `0011_consent_grants.sql`，数据库二次限制 scope/evidence 的键、
    枚举、版本格式与审计区域；并新增正常、异常、并发幂等、重试、区域、
    严格 JSON 与零内部字段泄露测试；同步领域、数据、安全、隐私、验收和实施计划。（TASK-011、FR-040、SEC-030/031/041/044）
- TASK-014 JD 解析、AI 推理校对与材料缺失降级（`task/TASK-014-jd-parsing` 分支）：
  - 扩展 `/v1/jobs*` OpenAPI 3.1 契约：JD 粘贴原文保留、解析/重试、不可变版本单字段校对与
    确认、仅简历推导岗位、四种材料影响弹窗和严格匹配的显式降级同意。（TASK-014、FR-004、FR-005）
  - `ai/services/parsing` 新增供应商中立 `JobParsingProvider`、L1/L2/L4 JD 请求及 L3 已确认
    安全简历快照、确定性合成桩、Schema/暂时错误自动重试 ≤2 次与 NFR-015 输入保留重试；
    TASK-030 前不绑定厂商 SDK。AI 推理来源标记不可移除，人工校对和确认均追加新版本。
  - 薪资福利、公司福利、招聘联系人在适配器调用前整段剔除、输出后递归清洗、写版本及
    面试上下文/评分上游前零命中 fail-closed；新增中英文/注入合成样例与
    `job-parsing-governance` 评测集，排除内容泄露命中为 0。（TASK-014、FR-004、SEC-026）
  - 新增 `0014_job_parsing.sql`：区域 uploads 原文引用或已确认简历引用二选一、解析尝试、
    追加式岗位版本、材料影响快照及显式降级同意的幂等/不可变约束；同步领域、数据、隐私、
    安全、Provider/Prompt、验收矩阵与实施计划。
- TASK-010 用户、Identity 与多方式登录（`task/TASK-010-identity-login` 分支）：
  - 新增 `/v1/identity/*` OpenAPI 3.1 契约与 Go HTTP 适配层：邮箱验证码、Google/Apple/微信
    验证、首次注册、会话/刷新轮换、账户偏好和身份绑定；写操作强制幂等键，部署与令牌双重
    校验 `data_region`，错误响应不回显验证码、OAuth 授权码或令牌。（TASK-010、US-05、FR-027）
  - `services/identity` 新增供应商中立验证适配器、区域 `services/notify` 邮件投递、主体/验证码/
    证明摘要、业务 JWT、单次刷新令牌、串行事务存储与并发幂等执行；US-05 场景 4 要求当前侧与
    目标侧双重验证，身份冲突只追加恢复案件且绝不移动身份或合并账户。（TASK-010、SEC-002、SEC-012）
  - 新增 `0010_identity_accounts.sql`：用户、身份、验证、会话、冲突恢复与幂等表，区域/提供商/
    主体摘要唯一约束；邮箱、验证码、OAuth 授权码、证明和业务/刷新令牌均不落明文。同步领域、
    数据、安全、隐私、验收矩阵、实施计划与身份服务文档，并覆盖正常、异常、重试、并发幂等、
    跨区和零泄露测试。（TASK-010、US-05、FR-027、SEC-040）
- TASK-013 简历结构化解析与敏感字段硬隔离（`task/TASK-013-resume-parsing` 分支）：
  - 新增 `/v1/parsing/resumes*` OpenAPI 3.1 契约：accepted 原件解析/状态/步骤级重试、
    追加式版本查询、逐字段 add/replace/remove/confirm 与低置信度清零后最终确认。
  - `ai/services/parsing` 新增供应商中立 `ResumeParsingProvider`、L1/L2/L4 分层请求、确定性合成桩、
    Schema/暂时错误自动重试 ≤2 次及 NFR-015 原件保留重试；不绑定厂商 SDK。（TASK-013、FR-002）
  - 落实 SEC-040 四道门：解析前脱敏、模型输出后递归清洗、版本写入前 Schema/零命中断言、
    面试上下文与评分上游材料组装前再次 fail-closed；新增全类别合成样例与
    `resume-parsing-security` 评测集，敏感泄露命中为 0。（TASK-013、FR-003、SEC-040）
  - 新增 `0013_resume_parsing.sql` 的解析尝试/主记录/追加式版本、幂等、敏感根键与低置信度
    二次 CHECK；同步领域、数据、安全、隐私、验收、Provider/Prompt 契约与实施计划。
- TASK-012 简历隔离上传与恶意文件检测（`task/TASK-012-upload-scanning` 分支）：
  - 新增 `/v1/uploads/resumes`、`/v1/uploads/{uploadId}`、`/v1/uploads/{uploadId}:retry`
    OpenAPI 3.1 契约：具体拒绝码、沙箱证明、原件保留/可重试/不计费/不影响评分语义。
  - `services/ingestion` 新增上传服务、所属区域 uploads 桶专用对象存储接口、一次性无网络沙箱
    证明门槛、供应商中立恶意软件检测接口，以及 PDF/DOC/DOCX 病毒、宏、压缩炸弹、伪装、
    超限、损坏、加密矩阵；含正常、异常、并发幂等与超时重试测试。（TASK-012、FR-001、FR-006）
  - 新增 `0012_resume_uploads.sql`：上传/扫描尝试状态表、10 MiB 与区域 uploads 桶 CHECK、
    上传及扫描重试两级幂等唯一键；同步领域、数据、安全、威胁、验收与对象存储契约。
    （TASK-012、TM-02、SEC-020、NFR-006）
- TASK-015 企业公开流程来源服务（`task/TASK-015-enterprise-process-sources` 分支）：
  - 新增 `services/source` Go 控制面服务（单模块，登记 `go.work` 与 CI golangci 矩阵）：来源领域模型
    （链接/日期/类型/可信度/失效状态，候选人经验强制标记非官方）、官方优先排序与可靠性判定、
    供应商中立检索适配层契约 + 合成桩（TASK-030 未开工前不绑定厂商 SDK，PROVIDER-ADAPTERS §4.5）、
    可重试错误退避重试 ≤2 次、幂等键去重（NFR-006）；无公司/断网/无可信来源自动回退通用模板并
    标记 AI 推导（`flow_uses_generic_template`/`ai_derived`，可人工校对，不伪装企业流程）；
    内存幂等存储抽象；外部网页内容仅作为不可信数据进入结构化提取（SEC-024/SEC-025）。
    含正常/异常/幂等/并发/重试/注入按数据处理单测。（TASK-015、FR-007、FR-008、NFR-006）
  - 新增追加式迁移 `services/migrate/migrations/0002_process_sources.sql`（幂等键/URL 唯一、
    data_region 强制且与 region 一致、source_type/credibility/status CHECK；TASK-003 迁移工具）。
  - API 契约新增 `/v1/sources/*`（search/list/get）与 ProcessSource/SourceSearchResult 等 Schema，
    同步 DOMAIN-MODEL §6.6、DATA-MODEL §5.2、ACCEPTANCE-MATRIX FR-007/FR-008、
    IMPLEMENTATION_PLAN（TASK-015 状态）；`fixtures/synthetic/process-sources/` 扩展合成样例。
    （TASK-015、FR-007、FR-008、SEC-024、SEC-025）
- TASK-008 备份与恢复（`task/TASK-008-backup-recovery` 分支）：
  - 新增 `services/backup` 共享包与 CLI（yaml.v3）：备份契约 fail-closed（每日完整+WAL+PITR、
    证据 RPO=0、其他 RPO ≤5s、RTO ≤30 分钟、区域内备份桶、恢复前强制 tombstone 过滤）、
    一键恢复 dry-run 固定步骤序列；含正常/异常/幂等单测。（TASK-008、SEC-050、SEC-052）
  - 新增 `infra/modules/backup/` 模块契约与三区×三环境 `topology.backup` 配置，
    纳入 `regions` 套件校验（策略/PITR/RPO/RTO/tombstone）；`tools/validate_docs.py` 新增
    `backup` 套件（第 18 套件）并接入 CI 阶段 1、golangci 矩阵与本地检查。（TASK-008）
  - 新增 `docs/operations/RECOVERY-RUNBOOK.md`（一键恢复流程）与
    `tools/backup/quarterly-drill-template.md`（季度恢复演练模板，含 RPO/RTO 结果记录表）。
    **EPIC-01（TASK-001~008）全部完成。**（TASK-008）
- TASK-007 区域化通知与身份通道（`task/TASK-007-notification-identity-channels` 分支）：
  - 新增 `services/notify` 共享 Go 包：区域化邮件通道契约（`Config.Validate` fail-closed）、
    `Router` 按 `data_region` 路由（单区通道故障不影响他区，未配置区拒绝发送）、消息强制幂等键、
    模板变量禁止正文/令牌/媒体（PRIVACY-DATA-MAP）；含正常/异常/幂等/隔离单测。（TASK-007、FR-027）
  - 新增 `services/identity/provider` 身份提供商区域注册表（FR-027 开放矩阵：email 全区域、
    wechat 仅 cn、google/apple 仅 eu/intl），`ValidateProviders` fail-closed；含正常/异常/幂等单测。
  - 新增 `infra/modules/notification/` 模块契约与三区×三环境 `notification.channels`、
    `identity_providers` 配置，纳入 `regions` 套件校验；`tools/validate_docs.py` 新增
    `channels` 套件（第 17 套件）并接入 CI 阶段 1、golangci 矩阵与本地检查。（TASK-007）
- TASK-006 密钥管理系统接入与轮换契约（`task/TASK-006-secrets-key-mgmt` 分支）：
  - 新增 `services/secretref` 共享 Go 包（零外部依赖）：`*_REF` 引用命名契约（`ValidateRefName`）、
    区域 `secrets.refs` fail-closed 校验（值必须为合法环境变量名、不得内联真实密钥，SEC-012）、
    展示掩码 `MaskSecret`（只保留末 4 位）；含正常/异常/幂等单测。（TASK-006、SEC-012）
  - 新增 `infra/modules/secret-ref/` 模块契约（每区 KMS、ref_only、零明文）与三区×三环境 `kms_name`，
    纳入 `regions` 套件资源命名校验。（TASK-006）
  - `tools/validate_docs.py` 新增 `key-mgmt` 套件（第 16 套件：模块契约 + `tools/secret-rotation/rotation_drill.py`
    轮换演练）并接入 CI 阶段 1、golangci 矩阵与本地检查；新增
    `docs/operations/KEY-ROTATION-RUNBOOK.md` 轮换运行手册（五步流程 + 分类过渡方式）。（TASK-006、SEC-013）
- TASK-005 OpenTelemetry 观测与日志脱敏（`task/TASK-005-observability-otel` 分支）：
  - 新增 `services/observability` 共享 Go 包（OTel v1.44.0）：结构化 JSON 日志默认 strict 脱敏
    （敏感键整值替换 + JWT/Bearer/`sk-`/超长令牌值模式替换；生产强制 strict，SEC-032）、
    OTLP HTTP 指标/追踪 Provider 装配（资源含 `data_region`，导出器 none 时业务链路不受影响）、
    指标/追踪属性白名单校验（强制 `data_region`，白名单外/敏感键/疑似敏感值拒绝）；
    含正常/异常/幂等单测与合成敏感样本零泄露回归。（TASK-005、SEC-032）
  - 新增 `infra/modules/observability/` 模块契约与 README（每区 OTLP 采集端点、strict 脱敏、
    状态页骨架）；三区×三环境拓扑实例补充 `otel_collector`、`otlp_endpoint`、`redaction`、
    `status_page`，纳入 `regions` 套件校验。（TASK-005）
  - `tools/validate_docs.py` 新增 `observability` 套件（第 15 套件：模块契约、SDK 符号与测试、
    合成敏感样本、日志政策与状态页文档）并接入 CI 阶段 1、golangci 矩阵与本地检查。
    （TASK-005、SEC-032、SEC-033）
  - 新增 `docs/observability/LOGGING-POLICY.md`（日志与观测脱敏政策）与
    `docs/observability/STATUS-PAGE.md`（中英双语公开状态页骨架）；`.env.example` 增加
    SERVICE_NAME/SERVICE_VERSION/OTEL_* /REDACTION_MODE/STATUS_PAGE_BASE_URL；README、
    IMPLEMENTATION_PLAN、EPIC-01 设计、TEST-STRATEGY 同步。（TASK-005）
- TASK-004 Temporal 集群与每区命名空间、任务队列划分（`task/TASK-004-temporal-cluster-namespaces` 分支）：
  - 新增 `services/temporal` 共享 Go 包：`mgd-{region}-{env}-temporal` 命名空间生成与校验、
    七域任务队列（ingestion/plan/interview/scoring/report/billing/deletion）集合校验、
    `ValidateConfig` fail-closed；含正常/异常/幂等单测。（TASK-004、ADR-0001、ADR-0005）
  - 新增 `infra/modules/temporal/` 模块契约与 README：每区独立集群、跨 3 AZ、
    `history_retention_days=30`、命名空间模式与队列所有权；三区×三环境拓扑实例补充
    `cluster_name`、`cross_az: true`、`history_retention_days: 30`。（TASK-004）
  - `tools/validate_docs.py` 新增 `temporal` 套件（第 14 套件）并接入 CI 阶段 1 与本地检查；
    `regions` 套件扩展 Temporal 跨 AZ 与保留期校验；golangci 矩阵新增 `temporal`。
    （TASK-004）
  - `workflows/README.md` 登记命名空间模式与七域队列所有权表；README、
    IMPLEMENTATION_PLAN、EPIC-01 设计同步。（TASK-004）
- TASK-003 数据平台基线部署与迁移工具（`task/TASK-003-data-platform-migrations` 分支）：
  - 新增 `services/migrate` 迁移工具：`schema_migrations` + SHA-256 校验和，重复执行幂等；
    已应用迁移校验和变化即失败（fail-closed）；CLI 支持 `up` / `status`，并复用
    `DATA_REGION == INFRA_REGION` 启动自检。（TASK-003、NFR-005、NFR-006）
  - 基线迁移 `services/migrate/migrations/0001_ledger_baseline.sql`：四张追加式账本表
    （`evidence_items`、`score_versions`、`usage_ledger`、`access_audits`），含
    `data_region` CHECK、幂等键 UNIQUE、内容散列与数据库层 `REVOKE UPDATE, DELETE`
    （业务角色仅 SELECT/INSERT，删除编排专用角色保留受控 UPDATE/DELETE）。（TASK-003、ADR-0004）
  - 新增 `infra/modules/{database,object-storage,event-stream}/` 模块契约与 README；
    区域拓扑补充事件流六主题与媒体桶 30 天生命周期（`RETENTION-MATRIX`）。
    （TASK-003、NFR-005、RETENTION-MATRIX）
  - `tools/validate_docs.py` 新增 `data-platform` 套件（第 13 套件）：模块清单、
    迁移文件名、账本表/REVOKE/幂等键存在性、业务角色权限、迁移执行器与幂等测试，
    接入 CI 阶段 1 与本地检查；`regions` 套件新增事件流主题与媒体生命周期校验。
    （TASK-003、ADR-0004、NFR-005、NFR-006）
  - README、IMPLEMENTATION_PLAN、ACCEPTANCE-MATRIX（NFR-005/NFR-006 契约层级）、
    DATA-MODEL（迁移工具锚点）同步。（TASK-003）
- TASK-002 三数据区环境拓扑与区域路由（`task/TASK-002-three-region-topology-routing` 分支）：
  - `infra/regions/{cn,eu,intl}/envs/{dev,staging,production}.yaml`：9 个环境拓扑实例，覆盖网络（3 AZ）、
    PostgreSQL / Redis（非证据存储）、对象存储三桶、区域事件流、SFU、Temporal 命名空间与任务队列、
    密钥引用、区域化供应商白名单、邮件通道与区域路由配置；资源命名统一 `mgd-{region}-{env}-`，
    区域间无跨区引用。（TASK-002、NFR-004、ADR-0005、OD-09）
  - `tools/validate_docs.py` 新增 `regions` 套件（第 12 套件）：强制区域/环境代码、3 AZ、副本门槛、
    Redis 非证据、Temporal 队列、资源命名前缀与零跨区引用；CI 阶段 1 与本地检查接入。（TASK-002、NFR-004）
  - 新增 `services/region` 共享 Go 包：数据区枚举、fail-closed 启动自检
    （`DATA_REGION == INFRA_REGION`、`SERVICE_ENV` 合法）与区域路由决策（跨区拒绝并返回
    `region_mismatch`）；七个 Go 控制面服务与四个 Python AI 服务接入，全部含正常/异常/幂等单测。
    （TASK-002、SEC-051、ADR-0005）
  - CI 阶段 6 与本地脚本构建循环适配库模块：仅对含 main 包的模块写入临时产物，纯库模块
    （如 `services/region`）执行 `go build` 编译检查，避免 `go: no main packages to build`。
    （TASK-002）
  - `.env.example` 新增 `[REGION-SCOPED] INFRA_REGION`；README 与各服务 README 同步运行与自检说明。
    （TASK-002）
- TASK-001 单仓工程骨架（`task/TASK-001-monorepo-skeleton-ci` 分支）：
  - 目录结构按 `docs/architecture/EPIC-01-INFRA-DESIGN.md` 第 4.1 节落地：`apps/web`、`apps/admin`（占位）；`services/` 七个 Go 控制面服务模块（identity、consent、ingestion、project、billing、org、adminapi，经根 `go.work` 统一工作区）；`ai/services/` 四个 Python AI 服务包（parsing、orchestrator、scoring、report，src 布局 + pyproject）；`contracts/`、`workflows/`、`infra/modules/`、`infra/regions/{cn,eu,intl}/`（占位）。
  - 每个服务含 fail-closed `DATA_REGION` 启动自检最小形态（ADR-0005、OD-09）与正常 + 异常路径单测；与所连基础设施区域的一致性校验挂接 TASK-002。
  - 各目录 README 记录技术基线、拥有任务、追踪锚点与红线。（TASK-001、ADR-0005、OD-09）
- CI 门禁流水线 `.github/workflows/ci.yml`（PR 与 main 推送触发）：阶段1 规范校验（`tools/validate_docs.py`）、阶段2 静态检查（gofmt、go vet 逐模块、golangci-lint v2 逐模块矩阵含 gosec SAST 基线、ruff、mypy strict）、阶段3 单元测试（go test、pytest）、阶段4 契约校验（YAML/JSON/JSON Schema/OpenAPI）、阶段5 安全扫描（仓内密钥模式、gitleaks 全历史、govulncheck + pip-audit 依赖漏洞门禁、SBOM 工件；GitHub 原生 dependency-review-action 需 GHAS 许可，转公开或购 GHAS 后可换回）、阶段6 构建（go build、Python 包可安装性）；TS 检查、契约生成物 diff、Schema 样例、事件目录一致性、集成测试、评测回归与 dev 部署留有挂接注释，随对应任务接入。（TASK-001；NFR-005、NFR-007 间接）
- 协作基线：`.gitattributes`（仓内统一 LF）、`.golangci.yml`（lint + gosec 基线）、`.github/pull_request_template.md`（DoD 自检清单）、`tools/ci/run_local_checks.sh`（本地一键复现 CI 阶段 1~4、6 及阶段 5 本地等价项）、`tools/ci/README.md`、`tools/requirements-dev.txt`（统一 Python 工具链）。（TASK-001）
- `tools/validate_docs.py` 新增 `--suites` 参数：11 个校验套件可按逗号键选（默认全部），支撑 CI 分阶段门禁。（TASK-001）
- `docs/testing/SPEC-REVIEW-CHECKLIST.md`：十角色规范评审检查单与签字表，驱动规范评审闭环。
- `docs/architecture/EPIC-01-INFRA-DESIGN.md`：EPIC-01 详细实施设计（单仓结构、CI 门禁、三数据区拓扑、TASK-003~008 实施要点）。（TASK-001 ~ TASK-008、NFR-001 ~ NFR-006）
- `docs/testing/PHASE0-PROVIDER-EVALUATION.md`：Phase 0 供应商评测方案（准入门槛、评分卡、指标矩阵、故障演练、决策产出），驱动 OD-01 关闭。（OD-01、TASK-030、TASK-096）
- git 仓库初始化与基线提交（main 分支）；`.gitignore`（机密、构建产物、用户内容入库禁令）。

### Changed

- `README.md`：仓库状态由"研发规范阶段"更新为 TASK-001 骨架已落地；"本地开发预期入口"改写为真实可执行命令（本地一键检查、Go/Python 骨架运行示例）；当前状态新增 TASK-001 完成项。（TASK-001）
- ruff 配置（`ai/services/*/pyproject.toml`）：按 AGENTS.md 中文书写约定关闭 RUF001/RUF002/RUF003（全角中文标点非 Unicode 混淆字符，启用为系统性误报）；其余规则集（含 S 安全规则）保持全量启用。（TASK-001）
- 规范评审签字完成（2026-08-01，需求发起人/产品决策人）：十角色评审检查单全部结论"通过"，由需求发起人/产品决策人代表全部角色签署（如实记录于 `docs/testing/SPEC-REVIEW-CHECKLIST.md` 第 6 节，含代表签字说明与专职角色复核义务）。按检查单第 7 节规则，全部规范文档版本标记由"草案，待评审"提升为"已批准 2026-08-01 规范评审"，配置与提示词契约 `status: draft_for_review` → `approved`（涉及 `config/rubrics/v1/default.yaml`、`config/interview-flows/v1/default.yaml`、`config/safety/policy.yaml`、`config/feature-flags.yaml`、`ai/prompts/` 六份契约及 26 份规范文档；PRD、枚举示例与规则文本保持不变）。
- 决策批准（2026-08-01，项目发起人/需求发起人）：OD-06 排期预算——批准按 `IMPLEMENTATION_PLAN.md`（EPIC-01 ~ EPIC-10、阶段退出条件驱动、不设主观日期）推进；预算金额与采购细节按线下流程执行、不入仓。OD-01 ~ OD-05 维持未决。同步更新：`IMPLEMENTATION_PLAN.md` 第 7 节、`README.md`、`docs/architecture/adr/README.md`。（OD-06）
- 决策确认（2026-08-01，需求发起人/产品决策人）：OD-07 取整规则（half-up 取整后比较门槛）、OD-08 非关键维度未覆盖归一化口径（`min_coverage_ratio=0.5` 保留为可校准参数）、OD-09 区域代码 `cn/eu/intl`、OD-10 状态/事件英文命名规范，由"未决"转为"已确认"；OD-01 ~ OD-06 维持 PRD 决策门槛继续未决。同步更新：`IMPLEMENTATION_PLAN.md` 第 7 节（新增状态列）、`docs/ai/SCORING-SPEC.md`、`config/rubrics/v1/default.yaml`、`docs/architecture/adr/README.md`、`docs/architecture/adr/ADR-0005-three-data-regions.md`、`README.md`。（OD-07 ~ OD-10）

## [0.1.0] - 2026-08-01

### Added

- 初始研发规范套件，覆盖 PRD-001 V1.0 全部需求（US-01~US-08、FR-001~FR-040、NFR-001~NFR-016）：
  - 根文件：`AGENTS.md`、`README.md`、`IMPLEMENTATION_PLAN.md`、`.env.example`。
  - 架构：`docs/architecture/SYSTEM-ARCHITECTURE.md`、`DEPLOYMENT.md`、ADR-0001~ADR-0005。
  - 领域：`docs/domain/DOMAIN-MODEL.md`、`INTERVIEW-STATE-MACHINE.md`、`BILLING-STATE-MACHINE.md`。
  - API 与事件：`docs/api/openapi.yaml`、`docs/api/realtime-events.md`。
  - AI 与评分：`docs/ai/`（AI-ORCHESTRATION、SCORING-SPEC、HANDOFF-SPEC、PROMPT-POLICY、PROVIDER-ADAPTERS）。
  - 数据与安全：`docs/data/`（DATA-MODEL、RETENTION-MATRIX、DATA-CLASSIFICATION）、`docs/security/`（THREAT-MODEL、PRIVACY-DATA-MAP、SECURITY-REQUIREMENTS）。
  - 设计与无障碍：`docs/design/`（SCREEN-SPEC、DESIGN-SYSTEM、ACCESSIBILITY）。
  - 测试与发布：`docs/testing/`（ACCEPTANCE-MATRIX、TEST-STRATEGY、RELEASE-CHECKLIST）。
  - 配置契约：`config/rubrics/v1/default.yaml`、`config/interview-flows/v1/default.yaml`、`config/safety/policy.yaml`、`config/feature-flags.yaml`。
  - AI 资产：`ai/prompts/` 提示词契约、`ai/schemas/` 8 份 JSON Schema、`ai/evals/` 合成评测数据集与预期结果。
  - 合成测试材料：`fixtures/synthetic/`（虚构简历、JD、逐字稿、来源样例、故障事件）。
- 文档与契约校验工具 `tools/validate_docs.py`（开发工具，非产品源代码）：必交文件、YAML/JSON、OpenAPI、JSON Schema、Mermaid、占位符、需求覆盖、跨文件一致性与密钥扫描。
- 未决事项登记 OD-01 ~ OD-10（IMPLEMENTATION_PLAN.md 第 7 节）。

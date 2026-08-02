# 验收矩阵（ACCEPTANCE-MATRIX）

| 字段 | 内容 |
|---|---|
| 文档编号 | TEST-MATRIX-001 |
| 版本 | 0.1.0（已批准 2026-08-01 规范评审） |
| 追踪 | PRD-001 全部 US-01~US-08、FR-001~FR-040、NFR-001~NFR-016；US 验收场景；Pre-Launch Hard Gates |
| 一致性锚点 | `IMPLEMENTATION_PLAN.md`（Epic/TASK）、`docs/ai/SCORING-SPEC.md`（SC-EC-01~24）、`docs/domain/INTERVIEW-STATE-MACHINE.md`、`docs/domain/BILLING-STATE-MACHINE.md` |

## 1. 目的

建立 PRD 每一项需求到可执行测试用例的双向追踪，保证"每项需求至少一个正常场景和一个异常场景"，并让覆盖率可由 CI 机器校验。

## 2. 范围

- 8 项用户故事（含 PRD 全部 Acceptance Criteria 场景）、40 项功能需求、16 项非功能需求。
- 评分算法边界案例（SC-EC-01~24）的层级映射。

## 3. 非目标

- 不包含测试用例的步骤级脚本（用例实现时引用本表 ID）。
- 不重复 TEST-STRATEGY 的方法论与 RELEASE-CHECKLIST 的发布门槛。
- 不以"人工抽查"替代可自动化用例；manual_review 仅用于必须人工判断的项。

## 4. ID 与层级规则

- 测试 ID：`TC-{需求ID}-N{两位序号}` = 正常场景；`TC-{需求ID}-A{两位序号}` = 异常场景（如 `TC-FR-001-N01`、`TC-FR-001-A01`）。
- 自动化层级：`unit`（单元）、`contract`（契约）、`integration`（集成）、`e2e`（端到端）、`performance`（性能）、`security`（安全）、`ai_eval`（AI 评测）、`manual_review`（人工审查）。一行可标多个层级。
- CI 校验：脚本解析本表，确认 64 项需求 ID 全部出现且每项 N/A 用例 ≥1，缺失即构建失败（覆盖率 = 100%）。

## 5. 用户故事验收（覆盖 PRD Acceptance Criteria 全部场景）

| 需求 ID | 摘要 | 正常场景 | 异常场景 | 层级 | Epic/TASK |
|---|---|---|---|---|---|
| US-01 | 上传简历与 JD 并校对 | TC-US-01-N01 合法简历+JD 解析完成，展示结构化字段/低置信度/面试线索，确认后才生成计划；TC-US-01-N02 含电话/照片/性别/地址的简历，面试上下文构建不含这些字段 | TC-US-01-A01 只提供一项或零项，弹窗说明影响，明确同意后按降级模式生成且推理可编辑有标记；TC-US-01-A02 损坏/加密/超 10MB/病毒/宏/压缩炸弹/伪装逐项拒绝，原因具体，安全拒绝清除隔离副本且 JD 不丢失；扫描超时保留 uploads 隔离原件并可幂等重试；TC-US-01-A03 解析服务中断，输入保留，可重试失败步骤或手动编辑 | e2e、security、integration | EPIC-02 / TASK-012~014 |
| US-02 | 生成多轮面试计划 | TC-US-02-N01 存在可靠公开资料时计划展示来源/日期/类型/可信度，经验来源标记非官方；TC-US-02-N02 用户调整 1–5 轮/时长/角色/工具/顺序后系统重新校验权重与量表 | TC-US-02-A01 无公司/断网/来源不可靠时回退通用模板并标记 AI 推导，不伪装企业流程；TC-US-02-A02 某轮缺覆盖方案或量表时阻止进入并只重试缺失部分；TC-US-02-A03 生成内容含歧视/无关隐私/危险内容时被过滤并重新生成，不进入房间 | e2e、ai_eval、contract | EPIC-02、EPIC-04 / TASK-015、016、033 |
| US-03 | 实时数字人面试 | TC-US-03-N01 数字人音视频连接成功，语音/文字/工具作答获实时追问，正式证据按回合持久化；TC-US-03-N02 用户不开摄像头/麦克风用文字作答，面试继续、数字人音视频保持、口语项不记零 | TC-US-03-A01 用户语音/停止按钮/文字打断数字人，未实际播放内容不写入正式问题；TC-US-03-A02 数字人音视频持续失败，自动重连无效后询问降级：同意则恢复、拒绝则评估未完成并返还故障额度；TC-US-03-A03 ASR 有误时在下一主问题前修订成功，评分用修订文本、原始转写保留诊断 | e2e、performance、integration | EPIC-03 / TASK-020~027 |
| US-04 | 评分、复盘、训练与重试 | TC-US-04-N01 总分与全部关键维度 ≥60，首先显示祝贺文案并解锁下一轮，摘要不泄露后续答案；TC-US-04-N02 某维度 ≥60 且重试证据不矛盾时锁定分保留，新分只替换重评维度并重算总分 | TC-US-04-A01 总分或任一关键维度 <60 时阻断下一轮并生成累计纪要与训练入口；TC-US-04-A02 完成非评分练习后正式分数与解锁状态不变；TC-US-04-A03 首次复核用冻结证据重算，展示旧分/新分/理由，禁止改证据与权重，版本全保留；TC-US-04-A04 雷达图等单模块失败时展示可用模块并只重试失败部分 | e2e、unit、ai_eval、integration | EPIC-05、EPIC-06 / TASK-040~054 |
| US-05 | 登录与资产管理 | TC-US-05-N01 游客上传个人资料被要求登录，成功后返回原操作且不丢失允许暂存的非敏感输入；TC-US-05-N02 另一设备登录可从最后有效状态继续，评分版本与证据不变 | TC-US-05-A01 正式面试在一台设备进行时第二台设备被阻止，经确认可安全转移；TC-US-05-A02 身份绑定必须提交当前侧与目标侧两份独立、短期、单次证明；缺任一侧即拒绝，目标身份冲突时仅创建恢复案件、`accounts_merged=false`，两账户与身份归属不变并提供人工支持；TC-US-05-A03 重新验证后删除账户，展示真实进度，完成级联删除或不可逆匿名化，法定财务记录解除内容关联 | unit、contract、integration、security、e2e | EPIC-02 / TASK-010、018 |
| US-06 | 购买与额度 | TC-US-06-N01 购买页展示总价/轮次/预计最大时长/重试权益/税费/有效期；TC-US-06-N02 额度足够时开始面试即预留，本轮不因余额变化中断 | TC-US-06-A01 系统责任导致评估未完成时全部预留自动返还、账本记原因并提供重试或退款入口；TC-US-06-A02 支付回调重复/超时/状态不明时同一订单只记一次权益和一次扣款；TC-US-06-A03 账期在正式面试中结束时当前轮正常结束，后续轮次开始前再校验 | e2e、integration、security | EPIC-07 / TASK-060~065 |
| US-07 | 机构训练 | TC-US-07-N01 未授权结果分享时指导老师只能看到已完成与时间；TC-US-07-N02 只授权雷达图 30 天时仅展示雷达及期限，到期自动失效 | TC-US-07-A01 细分群体 <10 人时隐藏或合并到更大群体；TC-US-07-A02 退出或被移除后机构访问立即失效，个人记录保留，审计继续存在；TC-US-07-A03 机构管理员尝试改及格线/量表/个人分数被拒并写审计 | e2e、security | EPIC-08 / TASK-070~074 |
| US-08 | 运营与治理 | TC-US-08-N01 实时运营查看故障会话只显示匿名技术状态；TC-US-08-N02 新模型未完成离线/影子/灰度时全量发布被阻止并显示缺失门槛 | TC-US-08-A01 任一后台角色直接编辑分数或解锁状态被拒，个体结果只能经正式复核产生新版本；TC-US-08-A02 用户未授权时客服不可见逐字稿与媒体，授权后仍受会话范围与有效期限制；TC-US-08-A03 区域供应商紧急停用后新会话按区域规则切换已验证替代，活跃会话按冻结/故障恢复处理；TC-US-08-A04 灰度指标退化触发回滚，新会话回稳定版、进行中正式会话不被中途改变 | e2e、security、manual_review、integration | EPIC-09 / TASK-080~085 |

## 6. 功能需求验收（FR-001 ~ FR-040）

| 需求 ID | 摘要 | 正常场景 | 异常场景 | 层级 | Epic/TASK |
|---|---|---|---|---|---|
| FR-001 | 简历上传格式与大小 | TC-FR-001-N01 .pdf/.doc/.docx ≤10MiB 先写所属区域 uploads/quarantine，经安全通过后移入 accepted 并进入解析队列；同一幂等键只写一次 | TC-FR-001-A01 实际内容 >10MiB、非法扩展名或扩展名/魔数不一致在解析前拒绝，不写 exports/media；相同幂等键不同内容返回冲突 | contract、integration | EPIC-02 / TASK-012 |
| FR-002 | 简历解析与校对 | TC-FR-002-N01 accepted 原件经供应商中立适配层生成 Schema 合法结构；低于 0.75 的路径高亮；每次单字段 add/replace/remove/confirm 追加新版本，全部校对后最终确认冻结 | TC-FR-002-A01 仍有低置信度路径、过期 base_version、已确认版本再编辑均阻断；相同幂等键不重复调用模型或写版本 | e2e、unit、ai_eval | EPIC-02 / TASK-013 |
| FR-003 | 敏感字段排除 | TC-FR-003-N01 含电话/邮箱/证件/地址/照片/保护属性的合成简历经四道门后，结构化版本、面试上下文、评分上游材料的敏感命中数均为 0；版本仅存排除类别 | TC-FR-003-A01 恶意供应商输出和人工字段编辑尝试夹带敏感键/值均 fail-closed 且不追加版本；跨用户/跨数据区 accepted 原件读取拒绝 | security、contract、ai_eval | EPIC-02 / TASK-013 |
| FR-004 | JD 解析与标记 | TC-FR-004-N01 中英文 JD 结构化且用户确认后可用；TC-FR-004-N02 AI 推理面试重点明确标记、可逐字段编辑且来源标记不可移除；TC-FR-004-N03 相同创建/解析/编辑/确认幂等键无重复版本 | TC-FR-004-A01 薪资福利/公司福利/招聘联系人在适配器输入、岗位版本、上下文、评分上游泄露命中均为 0；TC-FR-004-A02 Schema/超时内部重试 2 次后保留原始输入，可用新幂等键只重试解析；TC-FR-004-A03 JD 注入只作 L4 数据 | unit、eval、contract | EPIC-02 / TASK-014 |
| FR-005 | 缺失降级模式 | TC-FR-005-N01 四种材料组合均返回完整影响弹窗；TC-FR-005-N02 仅 JD 禁止虚构经历/简历深挖/经历匹配评分；TC-FR-005-N03 仅简历生成全部带 AI 来源标记且可编辑的岗位画像；TC-FR-005-N04 两者皆无只允许表达/逻辑/沟通/应变 | TC-FR-005-A01 非 full 未明确 accepted=true 不得继续；TC-FR-005-A02 同意与影响快照/用户/区域/模式不匹配时拒绝；TC-FR-005-A03 同意重试不产生重复记录 | unit、eval、contract | EPIC-02 / TASK-014 |
| FR-006 | 恶意文件检测 | TC-FR-006-N01 病毒/宏（DOC、DOCX）/压缩炸弹/伪装/损坏/加密矩阵全部在无网络、一次性、只读根文件系统、无凭证沙箱内扫描并以稳定具体原因拒绝 | TC-FR-006-A01 沙箱证明不完整时 fail-closed；扫描超时/扫描器暂时不可用保留隔离原件并只重试扫描步骤，重复重试无副作用；安全拒绝不可重试并清除隔离对象 | security、manual_review | EPIC-02 / TASK-012 |
| FR-007 | 公开流程来源 | TC-FR-007-N01 按公司/岗位/级别/地区返回来源链接/日期/类型/可信度；同幂等键重复检索只落一次（NFR-006），可重试错误按退避重试后成功 | TC-FR-007-A01 断网或来源失效时回退通用模板并标记 AI 推导；不可重试错误不重试直接回退 | integration、ai_eval | EPIC-02 / TASK-015 |
| FR-008 | 来源优先级 | TC-FR-008-N01 官方来源优先排序，经验内容标记非官方；仅候选人经验/失效/低可信时回退通用模板且标记 AI 推导 | TC-FR-008-A01 无可信来源时内容不得冒充企业事实（P0），通用模板不携带外部链接 | unit、ai_eval | EPIC-02 / TASK-015 |
| FR-009 | 轮次与时长边界 | TC-FR-009-N01 默认 3 轮；用户合法调整为 1–5 轮、10–60 分钟 | TC-FR-009-A01 0 轮/6 轮/70 分钟等越界配置被 contract 拒绝 | e2e、contract | EPIC-02 / TASK-016 |
| FR-010 | 轮次编辑 | TC-FR-010-N01 增删重排、角色/重点/难度/风格/头像/声音/工具保存生效 | TC-FR-010-A01 修改统一评分算法/60 分门槛/解锁逻辑被拒 | e2e、contract | EPIC-02 / TASK-016 |
| FR-011 | 开始前冻结 | TC-FR-011-N01 确认后量表/权重/流程/版本/覆盖方案冻结 | TC-FR-011-A01 开始后修改冻结项返回 state_conflict | contract、integration | EPIC-02 / TASK-016 |
| FR-012 | 混合问题策略 | TC-FR-012-N01 预生成主线 + 会中按实际回答动态追问 | TC-FR-012-A01 追问越出已确认范围时被决策图拦截重选 | ai_eval | EPIC-04 / TASK-032 |
| FR-013 | 数字人实时入会 | TC-FR-013-N01 数字人以 WebRTC 视频+音频参与者加入，建连达标 | TC-FR-013-A01 静态头像/预录视频/纯文字替代被验收检测判不合规 | e2e、performance | EPIC-03 / TASK-020、021 |
| FR-014 | 角色库与人格 | TC-FR-014-N01 角色来自固定授权 2D 库，人格/背景/风格动态生成 | TC-FR-014-A01 每场生成新脸或未授权克隆请求被拒 | integration、security | EPIC-03 / TASK-021 |
| FR-015 | 四类输入 | TC-FR-015-N01 语音/摄像头/文字/岗位工具四通道均可作答（TASK-027 会前冻结测试） | TC-FR-015-A01 单通道故障时其余通道可继续完成面试（模式枚举校验 fail-closed） | e2e、integration、unit | EPIC-03 / TASK-027 |
| FR-016 | 设备开关规则 | TC-FR-016-N01 关闭摄像头/麦克风面试继续，数字人音视频始终开启（TASK-027 冻结语义测试） | TC-FR-016-A01 数字人视频或音频中断立即进入故障流程（TASK-025 暂停/降级） | e2e、integration、unit | EPIC-03 / TASK-027 |
| FR-017 | 打断与回合 | TC-FR-017-N01 语音打断/停止按钮至停止发声 P95 ≤500ms | TC-FR-017-A01 重叠说话场景检测并避免，无法判断时询问是否答完 | performance、e2e | EPIC-03 / TASK-022 |
| FR-018 | 字幕与修订 | TC-FR-018-N01 双向字幕实时展示，窗口内修订确认为评分证据（TASK-023 服务/HTTP 测试） | TC-FR-018-A01 进入下一主问题后修订被拒（回合已冻结，window_closed） | e2e、contract、unit | EPIC-03 / TASK-023 |
| FR-019 | 岗位工具 | TC-FR-019-N01 代码/白板/案例/作品集事件全量入证据账本（TASK-024 服务/HTTP 测试） | TC-FR-019-A01 正式房间临时加载未配置工具被拒（tool_not_configured） | integration、contract、unit | EPIC-03 / TASK-024 |
| FR-020 | 故障与降级 | TC-FR-020-N01 故障暂停计时、保存状态、自动重连成功恢复（TASK-025 服务/HTTP 测试） | TC-FR-020-A01 重连无效询问降级：同意继续/拒绝评估未完成且返还（TEXT_DEGRADED/ENDED+EVALUATION_INCOMPLETE） | integration、e2e、unit | EPIC-03 / TASK-025 |
| FR-021 | 双门槛审核 | TC-FR-021-N01 总分 ≥60 且关键维度 ≥60 判 PASS 并解锁 | TC-FR-021-A01 总分 80 但关键维度 59 判 FAIL 不解锁 | unit、ai_eval | EPIC-05 / TASK-040 |
| FR-022 | 通过展示 | TC-FR-022-N01 首先展示祝贺文案+摘要+下一轮角色/重点/难度/时长 | TC-FR-022-A01 摘要不含后续轮次完整标准答案（人工审查） | e2e、manual_review | EPIC-06 / TASK-054 |
| FR-023 | 报告内容 | TC-FR-023-N01 雷达/匹配/逐题证据/轨迹/沟通/工具/训练计划齐全且有文字等价 | TC-FR-023-A01 无 JD 项目不展示岗位匹配百分比 | e2e、unit | EPIC-06 / TASK-050 |
| FR-024 | 练习与重试 | TC-FR-024-N01 正式重试用新题，锁定维度保留、失败维度新分替换 | TC-FR-024-A01 练习回答写入正式证据链的尝试被阻断 | e2e、ai_eval、integration | EPIC-06 / TASK-052、053 |
| FR-025 | 正式复核 | TC-FR-025-N01 冻结证据重算产生新版本并展示前后对比 | TC-FR-025-A01 同一尝试第二次复核请求被拒 | integration、contract | EPIC-05 / TASK-043 |
| FR-026 | 部分报告 | TC-FR-026-N01 单模块失败时其余模块正常展示 | TC-FR-026-A01 只重试失败模块且不丢失评分证据 | e2e、integration | EPIC-06 / TASK-050 |
| FR-027 | 登录与身份 | TC-FR-027-N01 邮箱验证码经区域通知通道投递，Google/Apple/微信按开放矩阵登录，多身份双侧验证绑定成功；同幂等键并发重试只发送/创建一次，刷新令牌单次轮换 | TC-FR-027-A01 验证码错误/过期/频率限制/风险验证触发均 fail-closed；跨区提供商在适配器前拒绝；第三方故障明确邮箱替代且响应不回显邮箱、验证码、授权码或令牌；同幂等键不同请求返回冲突 | unit、contract、integration、security | EPIC-02 / TASK-010 |
| FR-028 | 语言独立 | TC-FR-028-N01 中文界面+英文面试组合生效 | TC-FR-028-A01 简历自动识别语言后面试语言仍须用户确认 | e2e | EPIC-02 / TASK-010、016、018 |
| FR-029 | 资产与历史 | TC-FR-029-N01 简历库/岗位库/项目筛选/训练进度跨设备同步 | TC-FR-029-A01 无匹配筛选结果展示空状态与引导 | e2e | EPIC-02 / TASK-018 |
| FR-030 | 单活动设备 | TC-FR-030-N01 第二设备进入正式面试被阻止并提示 | TC-FR-030-A01 用户确认安全转移后原设备会话失效 | integration、security | EPIC-02 / TASK-018 |
| FR-031 | 报价与预留 | TC-FR-031-N01 计划确认后完整报价，每轮开始前预留成功 | TC-FR-031-A01 额度不足阻止开始并提供购买，面试中不弹续费 | integration、e2e | EPIC-07 / TASK-060、061 |
| FR-032 | 秒级计费 | TC-FR-032-N01 仅数字人已连接且正式进行中的秒数被计量 | TC-FR-032-A01 生成/评分/故障暂停/重连时段对账为 0 计费 | unit、integration | EPIC-07 / TASK-061 |
| FR-033 | 商业全链路 | TC-FR-033-N01 免费额度/单项目包/Pro/加油包购买、账单逐笔可查、退款到账 | TC-FR-033-A01 系统故障自动全额返还；重复扣款自动识别原路退回 | e2e、integration | EPIC-07 / TASK-060~065 |
| FR-034 | 机构租户 | TC-FR-034-N01 邀请/机构邮箱/批量名单/SSO/SCIM 接入，角色权限分离 | TC-FR-034-A01 机构模板修改 60 分线/量表/证据规则被拒并写审计 | integration、security | EPIC-08 / TASK-070、071 |
| FR-035 | 机构可见性 | TC-FR-035-N01 默认只见任务状态；按任务授权后按范围+期限可见 | TC-FR-035-A01 撤回授权后在线访问立即失效 | security、e2e | EPIC-08 / TASK-072 |
| FR-036 | 聚合保护 | TC-FR-036-N01 ≥10 人细分正常展示聚合趋势 | TC-FR-036-A01 <10 人隐藏或合并；个人排名/搜索接口不存在 | integration、security | EPIC-08 / TASK-073 |
| FR-037 | 运营后台 | TC-FR-037-N01 区域/房间/供应商/错误预算监控默认匿名技术指标 | TC-FR-037-A01 运营尝试加入/旁听/代答被拒并写审计 | e2e、security | EPIC-09 / TASK-080 |
| FR-038 | 版本治理 | TC-FR-038-N01 模型/提示词/量表/工作流版本化、灰度、冻结、回滚生效 | TC-FR-038-A01 不兼容结构/缺失量表/未过安全测试的版本被发布系统阻止 | integration | EPIC-09 / TASK-081 |
| FR-039 | 禁止改分 | TC-FR-039-N01 全 API 不存在修改分数/解锁状态端点；复核为唯一入口 | TC-FR-039-A01 破窗访问限定理由与时长、事后复核、通知用户并写审计 | security、manual_review、contract | EPIC-09 / TASK-082、085 |
| FR-040 | 数据权利与授权证据 | TC-FR-040-N01 六类授权以封闭 scope 分别授予并追加证据版本，model_training 默认关闭且拒绝/撤回不影响 core_service；TC-FR-040-N02 删除/导出/更正/撤回工单化，各存储与第三方真实进度可查 | TC-FR-040-A01 撤回版本与 AccessAudit 同事务持久化，成功返回后在线访问立即拒绝；审计失败全事务回滚并以同一幂等键重试一次且无重复副作用；TC-FR-040-A02 删除或篡改历史授权/审计日志的业务尝试被拒（追加式） | unit、contract、integration、security | EPIC-02、EPIC-09 / TASK-011、083、084 |

## 7. 非功能需求验收（NFR-001 ~ NFR-016）

| 需求 ID | 摘要 | 正常场景 | 异常场景 | 层级 | Epic/TASK |
|---|---|---|---|---|---|
| NFR-001 | 核心可用性 ≥99.95% | TC-NFR-001-N01 月度 SLO 统计达标（账户/资产/计划/历史/报告） | TC-NFR-001-A01 单组件故障时核心读取降级不中断 | performance、integration | EPIC-01、EPIC-10 / TASK-002、092 |
| NFR-002 | 房间可用性 ≥99.95% | TC-NFR-002-N01 实时房间月度 SLO 达标 | TC-NFR-002-A01 SFU 节点故障时会话自动迁移或按故障流程恢复 | performance、integration | EPIC-03、EPIC-10 / TASK-020、092 |
| NFR-003 | 有效完成率 ≥99.5% | TC-NFR-003-N01 排除主动退出与本地断网后统计达标 | TC-NFR-003-A01 注入系统故障时被判失败的比例为 0 | performance、integration | EPIC-10 / TASK-091 |
| NFR-004 | ≥3 可用区 | TC-NFR-004-N01 每数据区部署拓扑跨 3 AZ 验证 | TC-NFR-004-A01 单 AZ 故障 60 秒内自动接管 | integration、contract、manual_review | EPIC-01 / TASK-002 |
| NFR-005 | 证据持久化 | TC-NFR-005-N01 下一主问题开始前上一有效回答已持久化 | TC-NFR-005-A01 持久化失败时阻塞推进并告警，不丢证据 | integration、contract | EPIC-01、EPIC-03 / TASK-003、026 |
| NFR-006 | 幂等副作用 0 | TC-NFR-006-N01 回答/评分/支付/额度/退款重复提交只生效一次 | TC-NFR-006-A01 并发双击/自动重试/乱序回调均无重复副作用 | integration、contract | EPIC-01、EPIC-07 / TASK-003、061、062 |
| NFR-007 | 建连 ≤8s/≤15s | TC-NFR-007-N01 数字人音视频建立 95% ≤8s、99% ≤15s | TC-NFR-007-A01 超时进入故障流程且不计时不计费 | performance、integration | EPIC-03 / TASK-020 |
| NFR-008 | 回应延迟 | TC-NFR-008-N01 发言结束至数字人回应 P50 ≤1.5s/P95 ≤3s/P99 ≤5s | TC-NFR-008-A01 超 P99 样本触发降级与告警策略 | performance | EPIC-03 / TASK-022 |
| NFR-009 | 打断停止 ≤500ms | TC-NFR-009-N01 打断至停止发声 P95 ≤500ms | TC-NFR-009-A01 连续打断 20 次无状态错乱 | performance | EPIC-03 / TASK-022 |
| NFR-010 | ASR 最终文本 ≤1s | TC-NFR-010-N01 ASR 最终文本 P95 ≤1s | TC-NFR-010-A01 弱网退化时仍提供修订入口与文字备选 | performance | EPIC-03 / TASK-022 |
| NFR-011 | 口型偏差 ≤200ms | TC-NFR-011-N01 口型与音频偏差 ≤200ms | TC-NFR-011-A01 超差时优先保证音频连续 | performance | EPIC-03 / TASK-021 |
| NFR-012 | 默认 ≥720p24fps | TC-NFR-012-N01 默认数字人视频 ≥720p、24fps | TC-NFR-012-A01 弱网降码率但音频连续不中断 | performance | EPIC-03 / TASK-021 |
| NFR-013 | 评分 P95 ≤60s | TC-NFR-013-N01 单轮评分生成 P95 ≤60s | TC-NFR-013-A01 超时标记评估未完成而非失败，可重算 | performance、integration | EPIC-05 / TASK-040 |
| NFR-014 | 报告 P95 ≤120s | TC-NFR-014-N01 完整报告生成 P95 ≤120s | TC-NFR-014-A01 单模块超时局部失败，其余正常 | performance、integration | EPIC-06 / TASK-050 |
| NFR-015 | 解析 P95 ≤60s | TC-NFR-015-N01 10MB 内简历解析 P95 ≤60s，Schema/暂时错误自动重试不超过 2 次 | TC-NFR-015-A01 连续超时后 uploads/accepted 原件保留，只重试失败解析步骤；重复重试无副作用、未计费且不影响评分 | performance、integration | EPIC-02 / TASK-013 |
| NFR-016 | 计划 P95 ≤120s | TC-NFR-016-N01 面试计划生成 P95 ≤120s | TC-NFR-016-A01 单模块失败只重试该模块 | performance、integration | EPIC-04 / TASK-033 |

## 8. 评分边界案例映射（SCORING-SPEC 第 7 节）

| 用例 ID | 主题 | 层级 |
|---|---|---|
| SC-EC-01 ~ SC-EC-04 | 门槛与取整（60 边界、half-up） | unit、ai_eval |
| SC-EC-05 ~ SC-EC-06 | 证据不足/关键转写不可恢复 → 评估未完成 | unit、ai_eval |
| SC-EC-07 ~ SC-EC-08 | 非关键维度弱项与未覆盖归一化 | unit |
| SC-EC-09 ~ SC-EC-12 | 文字/混合/摄像头/便利设置计分规则 | unit、ai_eval |
| SC-EC-13 ~ SC-EC-15 | 重试锁定、矛盾解锁、练习隔离 | unit、ai_eval、integration |
| SC-EC-16 ~ SC-EC-18 | 复核新版本、二次复核拒绝、评分故障 | unit、integration |
| SC-EC-19 ~ SC-EC-20 | 权重边界校验、插值引用强制 | unit、contract |
| SC-EC-21 ~ SC-EC-22 | JD-only / 无 JD 的匹配度规则 | unit、ai_eval |
| SC-EC-23 ~ SC-EC-24 | 单轮失败不救场、评分幂等 | unit、integration |

## 9. 统计

| 项 | 数量 |
|---|---:|
| 需求总数（US+FR+NFR） | 64 |
| 验收用例总数（主表） | 154（US 42 + FR 80 + NFR 32） |
| 评分边界用例 | 24 |
| 合计 | 178 |
| unit 层级覆盖需求行 | 6 |
| contract | 11 |
| integration | 36 |
| e2e | 30 |
| performance | 16 |
| security | 16 |
| ai_eval | 7 |
| manual_review | 5 |

> 统计口径：按主表行统计（一行多层级时重复计入各层级）；CI 以机器解析复核本表数字。

## 10. 关键规则

1. 每项需求 ≥1 正常 + ≥1 异常场景，覆盖率必须 100%；新增需求必须先补矩阵再实现。
2. 用例数据只用 `fixtures/synthetic/` 或标记 `synthetic: true` 的合成数据。
3. 涉及金钱、评分、证据、删除的用例必须包含幂等与并发变体。
4. AI 相关用例必须挂接 `ai/evals/` 评测集，随提示词/模型/量表变更回归。

## 11. 异常处理

- 矩阵与实现漂移（需求 ID 无法解析）：CI 失败并阻塞合并。
- 用例长期不稳定（flaky）：隔离区运行并限期修复，不得直接删除；删除用例需 QA 负责人批准并同步本表。
- manual_review 项：评审记录归档并链接本表 ID，缺失视为未覆盖。

## 12. 验证方式

1. CI 脚本：解析本表 → 校验 64 项需求 ID 全覆盖、N/A 用例齐全、统计数字一致。
2. 每次发布候选：本表全量用例状态快照作为 RELEASE-CHECKLIST 输入。
3. 与 `IMPLEMENTATION_PLAN.md` 的 Epic/TASK 映射每季度复核一次。

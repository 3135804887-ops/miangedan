# 需求文档：全局前端页面（frontend-global-pages）

| 字段 | 内容 |
|---|---|
| 特性名 | frontend-global-pages |
| 实现语言 | TypeScript（Next.js App Router + React，`strict: true`） |
| 唯一需求事实源 | `docs/prd/PRD-001-面个蛋-V1.0.md` |
| 页面契约 | `docs/design/SCREEN-SPEC.md`（SCR-01 ~ SCR-17） |
| 无障碍契约 | `docs/design/ACCESSIBILITY.md` |
| 设计系统契约 | `docs/design/DESIGN-SYSTEM.md` |
| 状态契约 | `docs/domain/INTERVIEW-STATE-MACHINE.md` |
| API 契约（只读） | `docs/api/openapi.yaml` |
| 交付批次 | frontend-batch-0 ~ frontend-batch-4（每批独立 PR、独立验收） |

## 引言

本特性交付面个蛋（MianGeDan）全局前端页面的**静态实现与设计系统落地**：求职者端 `apps/web` 与运营后台 `apps/admin` 两个 Next.js 应用，覆盖 `docs/design/SCREEN-SPEC.md` 定义的 17 个页面组（SCR-01 ~ SCR-17）。

交付内容限于前端层：路由与布局、共享组件、状态展示、异常与降级覆盖层、mock 数据、设计令牌、双语 i18n、无障碍实现与前端测试。真实后端服务、真实媒体/数字人/ASR 链路、真实支付与最终品牌视觉不在本特性范围（见"明确不在范围"章节）。

页面通过共享 `Mock_Layer` 消费与 `docs/api/openapi.yaml` 一致的只读类型与合成数据，因此后续联调只需关闭 `Mock_Layer`，页面代码无需改写。

交付按 5 个批次推进，每批一个 `task/frontend-batch-{n}` 分支、一个 PR、CI 全绿后 squash 合入 main：

| 批次 | 分支 | 交付范围 |
|---|---|---|
| frontend-batch-0 | `task/frontend-batch-0-web-scaffold` | 工作区与 `apps/web` 脚手架、设计令牌、全局布局与错误页、CI 接入 |
| frontend-batch-1 | `task/frontend-batch-1-core-pages` | SCR-01 ~ SCR-07 |
| frontend-batch-2 | `task/frontend-batch-2-room-shell` | SCR-08、SCR-09 |
| frontend-batch-3 | `task/frontend-batch-3-results-report` | SCR-10 ~ SCR-15 |
| frontend-batch-4 | `task/frontend-batch-4-org-admin` | SCR-16、SCR-17（`apps/admin` 独立工程） |

## 术语表

- **Web_App**：`apps/web` 求职者端 Next.js 应用（承载 SCR-01 ~ SCR-16）。
- **Admin_App**：`apps/admin` 运营后台 Next.js 应用（承载 SCR-17，`basePath = /admin`，公网路径为 `/admin/{locale}/<page>`）。
- **UI_Kit**：`packages/ui` 共享 React 组件包，实现 DESIGN-SYSTEM 第 8 节组件状态清单。
- **Token_Pipeline**：`packages/design-tokens` 设计令牌包与"JSON 令牌 → CSS 变量"生成器。
- **I18n_Runtime**：`packages/i18n` 国际化运行时，提供 `zh-CN` / `en-US` 两套 ICU MessageFormat 资源与语言前缀路由解析。
- **Api_Types**：由 `docs/api/openapi.yaml` 生成的只读 TypeScript 类型产物（落点由设计阶段确定）。
- **Mock_Layer**：基于 MSW 的请求拦截层，返回合成（synthetic）数据；关闭后页面直连真实 API。
- **Frontend_Test_Suite**：Vitest + Testing Library + axe 组成的前端测试套件。
- **CI_Pipeline**：`.github/workflows/ci.yml` 定义的 6 阶段流水线。
- **项目状态枚举**：`DRAFT`、`PARSING`、`MATERIAL_REVIEW`、`PARSE_FAILED`、`PLAN_GENERATING`、`PLAN_REVIEW`、`PLAN_FAILED`、`READY`、`IN_SESSION`、`SCORING`、`ROUND_PASSED`、`ROUND_FAILED`、`PRACTICING`、`EVALUATION_INCOMPLETE`、`COMPLETED`（共 15 项，来源 DOMAIN-002 第 5.1 节）。
- **会话状态枚举**：`ROOM_CREATED`、`PRE_CHECK`、`AVATAR_CONNECTING`、`LIVE`、`PAUSED_SYSTEM`、`RECONNECTING`、`DOWNGRADE_PROMPTED`、`TEXT_DEGRADED`、`AUTH_PAUSED`、`ENDED`（共 10 项，来源 DOMAIN-002 第 6.2 节）。
- **便利设置枚举**：`text_only`、`mixed_input`、`slower_avatar_speech`、`repeat_questions`、`extended_time`、`silence_threshold_adjusted`、`no_proactive_interruption`、`reduced_motion`、`tool_keyboard_alternative`（共 9 项，与 `ai/schemas/turn-evidence.schema.json` 的 `accommodations_in_effect` 一致）。
- **五态**：SCREEN-SPEC 第 9 节强制的页面状态集合——空、加载、失败、权限不足、恢复。
- **错误五要素**：影响、数据是否保留、可重试动作、是否计费、是否影响评分。
- **暗黑模式（Dark_Pattern）**：诱导性设计模式（预勾选、确认羞辱、隐藏退订入口、捆绑授权）。此处**不指深色配色主题**，禁止的是诱导性设计。
- **synthetic 数据**：来自 `fixtures/synthetic/` 或页面本地 mock 文件、且带 `synthetic: true` 标记的虚构数据。
- **媒体桩（media stub）**：占位的媒体/数字人/ASR 前端接口实现，只暴露状态与容器，不产生任何伪装成实时数字人输出的画面或音频。

## 需求

需求编号规则：`G*` 为跨页面全局需求（对全部批次生效），`B{n}-*` 为批次 n 的页面需求。每条需求的"追踪"行给出 SCR / FR / US / NFR 关联，供 `IMPLEMENTATION_PLAN.md` 第 6 节 DoD 第 1 项（需求追踪）双向查询。

---

### A. 跨页面全局需求

本组需求在批次 0 建立基线，并在批次 1 ~ 4 的每个页面上持续生效；每批 PR 均需证明本组需求未被破坏。

#### 需求 G1：状态枚举一致性

**用户故事：** 作为求职者，我希望页面上的状态名称与系统真实状态一一对应，以便我不会因为页面自创的说法误判自己的进度。

**追踪：** 全部 SCR | FR-009、FR-021 | US-04、US-05 | DOMAIN-002 §5.1、§6.2 | 批次 0 ~ 4

##### 验收标准

1. THE Web_App SHALL 从单一共享常量模块导出项目状态枚举（15 项）与会话状态枚举（10 项），且枚举值文本与 `docs/domain/INTERVIEW-STATE-MACHINE.md` 第 5.1、6.2 节完全一致。
2. WHEN 任一页面渲染项目状态或会话状态，THE Web_App SHALL 使用共享枚举常量作为状态标识，并通过 `I18n_Runtime` 的固定翻译键渲染用户可见文案。
3. IF 页面代码引入枚举以外的状态标识字符串，THEN THE Frontend_Test_Suite SHALL 判定测试失败并输出该标识与所在页面。
4. THE Web_App SHALL 为每个状态徽标同时渲染图标、文字标签与该状态下的可用动作说明。
5. WHEN 渲染 `ROUND_PASSED`、`ROUND_FAILED`、`EVALUATION_INCOMPLETE` 三态，THE UI_Kit SHALL 为每态输出互不相同的图标与文字标签，且三态的区分在移除全部颜色样式后仍然成立。

#### 需求 G2：每页五态矩阵与错误五要素

**用户故事：** 作为求职者，我希望每个页面在没有数据、正在加载、出错、权限不足和中断恢复时都给出明确说明和下一步，以便我知道数据是否安全、要不要重试。

**追踪：** 全部 SCR | 全部 FR（用户可见错误规则） | US-01 ~ US-08 | SCREEN-SPEC §9、§11 | 批次 0 ~ 4

##### 验收标准

1. THE Web_App SHALL 为 SCR-01 ~ SCR-16 每个页面组提供五态视图：空、加载、失败、权限不足、恢复。
2. WHEN 页面处于空态，THE Web_App SHALL 渲染"为什么为空"的说明文案与一个指向下一步动作的可聚焦控件，且空态视图内不渲染任何形如业务数据的占位数值。
3. WHILE 页面处于加载态，THE Web_App SHALL 渲染骨架或进度指示并设置 `aria-busy="true"`；WHERE 该页面对应的 NFR 指标已定义（SCR-05 对应 NFR-015、SCR-06 对应 NFR-016、SCR-07 对应 NFR-007、SCR-10 对应 NFR-013、SCR-11 对应 NFR-014），THE Web_App SHALL 在加载文案中给出与该指标一致的预期时长。
4. WHEN 页面处于失败态，THE Web_App SHALL 在同一提示区域内渲染错误五要素（影响、数据是否保留、可重试动作、是否计费、是否影响评分）五段文案与一个重试控件。
5. WHEN 页面处于权限不足态，THE Web_App SHALL 说明所需权限与获取路径（登录、授权或购买），且响应文案中仅出现请求者自身可见的资源标识。
6. WHEN 用户在表单填写过程中触发失败态，THE Web_App SHALL 在重试后保留失败前已填写的表单字段值。
7. WHEN 用户刷新或重新进入页面，THE Web_App SHALL 依据 `Mock_Layer` 返回的最后有效状态渲染恢复态，并说明可继续的位置。

#### 需求 G3：双语与语言前缀路由

**用户故事：** 作为中文或英文用户，我希望通过 URL 就能确定界面语言，并且所有提示都有对应语言的正式译文，以便我在任一语言下都能完整完成流程。

**追踪：** 全部 SCR | FR-028 | US-05（规则 4） | DESIGN-SYSTEM §9 | 批次 0 ~ 4

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/…` 路由前缀下提供全部页面，`locale` 取值为 `zh-CN` 与 `en-US`。
2. WHEN 请求路径缺少语言前缀，THE I18n_Runtime SHALL 重定向到带前缀的等价路径。
3. WHEN 请求的 `locale` 不在 `zh-CN` 与 `en-US` 之内，THE I18n_Runtime SHALL 以 `en-US` 作为最终回退语言渲染页面。
4. THE I18n_Runtime SHALL 使用 ICU MessageFormat 渲染带变量、复数与日期/货币格式的消息，日期与货币按 `locale` 本地化。
5. IF 任一语言的翻译资源缺少某个已使用的翻译键，THEN THE CI_Pipeline SHALL 使该批次流水线失败并输出缺失键名。
6. WHEN 页面在缺失翻译键的情况下渲染，THE I18n_Runtime SHALL 渲染 `en-US` 回退文案，且渲染结果中不包含原始键名字符串。
7. THE Web_App SHALL 将界面语言与面试语言作为两个独立字段展示与设置（SCR-14 提供设置入口），并在 `<html lang>` 上输出当前界面语言。
8. THE I18n_Runtime SHALL 覆盖错误提示、字幕文案、支付与账单、隐私与授权、无障碍设置名称、报告解释六类文案，且中英文文案分别撰写。

#### 需求 G4：无障碍基线（WCAG 2.2 AA）

**用户故事：** 作为使用键盘、屏幕阅读器、放大或高对比模式的用户，我希望所有核心流程完整可用，以便我的输入方式不成为障碍。

**追踪：** 全部 SCR | FR-016、FR-018 | US-03（规则 8、9） | PRD Pre-Launch Hard Gates、ACCESSIBILITY §4 | 批次 0 ~ 4

##### 验收标准

1. THE UI_Kit SHALL 使用语义化 HTML 元素与 ARIA 角色实现全部交互组件，且每个组件的全部功能可仅通过键盘触发。
2. THE UI_Kit SHALL 为每个可聚焦元素渲染 `:focus-visible` 焦点环，焦点环宽度 ≥2 CSS px 且使用 `color.focus` 令牌。
3. THE UI_Kit SHALL 使指针目标的可点击区域 ≥24×24 CSS px，并使房间控制条、提交、退出等主要控制的可点击区域 ≥44×44 CSS px。
4. THE Token_Pipeline SHALL 使正文前景/背景组合的对比度 ≥4.5:1、大文本组合 ≥3:1，并在 CI 中以计算值校验全部语义色对。
5. WHEN 页面在 200% 缩放下渲染，THE Web_App SHALL 保持全部内容可见且无横向滚动；WHERE 页面为 SCR-08 实时面试房间，THE Web_App SHALL 允许工具区内部滚动。
6. WHEN 弹窗或覆盖层打开，THE UI_Kit SHALL 将焦点移入弹窗、在弹窗内圈定 Tab 序，并在关闭后将焦点返回触发元素。
7. THE Web_App SHALL 将字幕区域标记为 `aria-live="polite"`，并将故障与降级提示标记为 `role="alert"`。
8. THE Web_App SHALL 为雷达图、趋势图与岗位匹配图同时渲染文字摘要与数据表格等价版本，且等价版本在关闭图形渲染时仍可访问。
9. WHERE 用户系统设置 `prefers-reduced-motion: reduce`，THE Web_App SHALL 停止非必要装饰动效。
10. WHERE 操作系统启用强制颜色模式，THE UI_Kit SHALL 通过边框与图标表达语义边界与状态区分。
11. IF 表单提交存在字段级校验错误，THEN THE UI_Kit SHALL 输出与字段关联的错误说明并将焦点定位到第一个错误字段。
12. IF 网页字体加载失败，THEN THE Web_App SHALL 回退到系统字族栈且页面布局的可见结构保持完整。
13. WHEN Frontend_Test_Suite 对 SCR-01 ~ SCR-17 的核心页面运行 axe 检查，THE Frontend_Test_Suite SHALL 要求严重（serious 与 critical）违规数为 0。
14. THE Web_App SHALL 在 `zh-CN` 与 `en-US` 两种语言下分别满足本需求第 1 ~ 13 条。

#### 需求 G5：设计令牌与组件七态

**用户故事：** 作为多人协作的前端实现者，我希望颜色、字号、间距与组件状态来自同一套令牌与同一套组件，以便不同页面的观感与行为保持一致，并且未来换品牌视觉只改色值。

**追踪：** 全部 SCR | PRD Design & UX Requirements | DESIGN-SYSTEM §5 ~ §8、§10 ~ §12 | 批次 0 ~ 4

##### 验收标准

1. THE Token_Pipeline SHALL 以机器可读 JSON 保存 DESIGN-SYSTEM 第 5 ~ 7 节的语义颜色、字体层级、间距刻度与断点令牌，并生成对应的 CSS 变量文件。
2. THE Web_App SHALL 通过 Tailwind 主题消费 `Token_Pipeline` 生成的 CSS 变量，且页面样式中不出现硬编码色值与硬编码字号。
3. THE Token_Pipeline SHALL 保持语义令牌名称集合稳定；WHEN 令牌 JSON 的语义名称集合发生变更，THE CI_Pipeline SHALL 使流水线失败并输出差异项。
4. THE UI_Kit SHALL 为每个交互组件实现 default、hover、active、disabled、loading、error、focus-visible 七种状态。
5. WHEN 组件处于 disabled 状态，THE UI_Kit SHALL 使该组件不可聚焦并渲染禁用原因文案。
6. WHEN 组件处于 loading 状态，THE UI_Kit SHALL 输出 `aria-busy="true"` 与可读的忙碌提示。
7. WHEN 组件处于 error 状态，THE UI_Kit SHALL 同时渲染错误图标与文字说明。
8. THE Frontend_Test_Suite SHALL 对 `UI_Kit` 每个交互组件断言上述七种状态的渲染输出、可聚焦性与无障碍属性。
9. THE Token_Pipeline SHALL 仅使用 DESIGN-SYSTEM 第 5 节的中性占位色值，最终品牌视觉留待 OD-04 关闭后以新版本替换。

#### 需求 G6：数据层（只读类型生成 + 合成数据拦截）

**用户故事：** 作为前端实现者，我希望页面按真实 API 契约写 fetch 并在开发期由合成数据驱动，以便后续联调只需关闭拦截层而不必改写页面。

**追踪：** 全部 SCR | `docs/api/openapi.yaml` | AGENTS.md §4（合成数据） | 批次 0 ~ 4

##### 验收标准

1. THE Api_Types SHALL 由 `docs/api/openapi.yaml` 生成，且生成产物为只读类型定义（不含运行时逻辑）。
2. WHEN CI_Pipeline 重新执行类型生成，THE CI_Pipeline SHALL 校验生成产物与仓库内已提交产物无差异，并在存在差异时使流水线失败。
3. THE Web_App SHALL 使用标准 `fetch` 调用 `docs/api/openapi.yaml` 中已定义的路径，并以 `Api_Types` 标注请求与响应类型。
4. WHILE `Mock_Layer` 处于启用状态，THE Mock_Layer SHALL 拦截页面发出的全部 `fetch` 请求并返回合成数据。
5. WHEN `Mock_Layer` 被关闭，THE Web_App SHALL 在不修改页面组件代码的前提下向真实 API 发出同样的请求。
6. THE Mock_Layer SHALL 仅使用来自 `fixtures/synthetic/` 或页面本地 mock 文件的数据，且每个 mock 数据文件带 `synthetic: true` 标记。
7. THE Mock_Layer SHALL 为每个页面组同时提供成功响应、错误响应与空数据响应三类场景，供五态测试驱动。
8. IF mock 数据中出现真实个人信息（手机号、邮箱、证件号、详细地址、照片、录音录像引用）或缺少 `synthetic: true` 标记，THEN THE Frontend_Test_Suite SHALL 判定测试失败并输出该字段位置。

#### 需求 G7：响应式与平台差异

**用户故事：** 作为在桌面、平板或手机上使用的求职者，我希望页面在我的设备上可用，并在设备不适合某个环节时提前得到明确提示。

**追踪：** 全部 SCR | PRD Responsive Platform Rules | SCREEN-SPEC §5 | DESIGN-SYSTEM §7 | 批次 0 ~ 4

##### 验收标准

1. THE Web_App SHALL 使用断点 `desktop ≥1024px`、`tablet 768–1023px`、`mobile <768px`，且断点值来自 `Token_Pipeline`。
2. WHILE 视口宽度 ≥1024px，THE Web_App SHALL 提供 SCR-01 ~ SCR-16 的完整功能布局。
3. WHILE 视口宽度在 768 ~ 1023px，THE Web_App SHALL 提供完整流程，并对白板、案例与作品集工具区渲染触控优化布局。
4. WHILE 视口宽度 <768px，THE Web_App SHALL 提供 SCR-01 ~ SCR-06 与 SCR-10 ~ SCR-15 的完整功能，并在 SCR-08 内对代码编辑器与白板渲染"推荐使用桌面或平板"的提示与替代路径。
5. THE Web_App SHALL 将常规页面内容最大宽度限制为 1200px，并使 SCR-08 实时面试房间使用全宽布局。
6. WHEN 页面在 320px 宽度下渲染，THE Web_App SHALL 保持主要操作可见且无横向滚动。

#### 需求 G8：前端安全与隐私边界

**用户故事：** 作为用户，我希望前端不外泄我的内容、不持有供应商密钥、不跨数据区取数，以便我的材料与回答只留在应有的位置。

**追踪：** 全部 SCR | FR-003、FR-039、FR-040 | PRD Observability、Privacy/Security & AI Governance | AGENTS.md §2、§6 | 批次 0 ~ 4

##### 验收标准

1. THE Web_App SHALL 使前端日志与错误上报的载荷字段限于路由名、状态枚举值、错误码、请求标识与时间戳。
2. IF 前端日志或错误上报的载荷包含简历正文、完整回答文本、字幕正文、令牌或媒体二进制引用，THEN THE Frontend_Test_Suite SHALL 判定测试失败并输出违规字段名。
3. THE Web_App SHALL 通过服务端环境变量读取运行时配置；THE Web_App SHALL 使打包产物中不含任何供应商密钥、令牌或凭证字符串。
4. THE Web_App SHALL 将数据区（`cn` / `eu` / `intl`）作为由后端决定的只读上下文展示，且前端不提供跨数据区取数或切换数据区取数的调用路径。
5. THE Web_App SHALL 使面试上下文相关页面（SCR-04 ~ SCR-12）的渲染字段集合不包含电话、邮箱、证件号、详细地址、照片、性别、婚育字段，也不包含外貌、情绪、微表情、年龄、种族、国籍、残障或人格推断字段。
6. WHERE 页面需要说明敏感字段处理（SCR-05），THE Web_App SHALL 仅以"已排除类别"的类别名清单展示，且不渲染任何被排除字段的具体值。

#### 需求 G9：红线禁止项在界面层的落实

**用户故事：** 作为对结果公平性负责的产品与安全评审者，我希望界面上不存在任何可以绕过评分与隐私规则的入口，以便产品红线在最外层也成立。

**追踪：** 全部 SCR | FR-003、FR-013、FR-039 | PRD Out of Scope、Non-Goals | AGENTS.md §2 | 批次 0 ~ 4

##### 验收标准

1. THE Web_App 与 THE Admin_App SHALL 使全部页面的可交互控件集合中，编辑个人分数、编辑解锁状态、编辑证据正文三类控件的数量为 0。
2. THE Web_App SHALL 使六类授权（核心服务必要处理、保存原始音视频、机构共享、非必要产品分析、模型训练或研究、营销通知）各自为独立开关，默认值为未勾选，且每个开关提供撤回入口。
3. THE Web_App SHALL 使授权与订阅相关确认文案为中立陈述，且确认对话框的取消动作与确认动作具有同等可达性。
4. WHILE 用户处于 SCR-08 实时面试房间，THE Web_App SHALL 使续费提示、余额不足提示与广告类内容的渲染数量为 0。
5. THE Web_App SHALL 使 SCR-08 的数字人视频区域由媒体桩驱动，且在媒体桩状态为未连接时渲染文字状态说明与技术占位框，不渲染静态人像、预录视频或以文字冒充数字人发言的内容。
6. THE Web_App 与 THE Admin_App SHALL 使页面功能集合中不包含招聘网站抓取、职位聚合、自动投递、雇主 ATS、候选人筛选或排名、真实面试实时提示、肖像或声音克隆、课件播放类入口。
7. THE Web_App SHALL 使付费状态、机构角色与后台角色不改变任何页面的评分展示规则、复核入口、隐私控制、无障碍能力与故障恢复文案。

#### 需求 G10：固定文案红线

**用户故事：** 作为求职者，我希望通过、评估未完成、练习与导出的关键说明措辞稳定一致，以便我不会把"评估未完成"误解成失败、也不会把练习成绩误当正式结果。

**追踪：** SCR-10、SCR-11、SCR-12 | FR-021、FR-022、FR-024、FR-026 | US-04（场景 1 ~ 3、6） | SCREEN-SPEC §10 | 批次 3

##### 验收标准

1. WHEN 轮次状态为 `ROUND_PASSED`，THE Web_App SHALL 在结果页首屏第一条信息位置渲染文案"恭喜你通过本轮面试，已进入下一轮"。
2. WHEN 轮次或项目状态为 `EVALUATION_INCOMPLETE`，THE Web_App SHALL 渲染明确说明"这不是失败"的文案，并同时渲染原因分类、已保留内容与重试入口。
3. WHILE 用户处于 SCR-12 练习页，THE Web_App SHALL 持续渲染固定标识"练习不改变正式分数与解锁状态"。
4. WHEN 用户从 SCR-11 触发报告导出，THE Web_App SHALL 在导出预览与导出产物的展示区域渲染标记"模拟训练结果，不代表真实企业录用结论"。
5. THE Frontend_Test_Suite SHALL 对上述四类文案在 `zh-CN` 与 `en-US` 下建立快照测试，并在文案变更时使测试失败。

#### 需求 G11：测试、CI 与批次完成定义

**用户故事：** 作为需要合入 5 个批次的维护者，我希望每个批次都有一致的测试与门槛，以便任一批次单独合入都不破坏既有校验与文档追踪。

**追踪：** 全部 SCR | IMPLEMENTATION_PLAN §6 | AGENTS.md §4、§5、§7 | 批次 0 ~ 4

##### 验收标准

1. THE Frontend_Test_Suite SHALL 为每个批次覆盖四类用例：正常路径（页面渲染与状态正确）、异常路径（失败态、降级态、空态）、状态枚举一致性（页面使用的状态集合为 DOMAIN-002 枚举的子集）、幂等（覆盖层与提交动作重复触发）。
2. THE Frontend_Test_Suite SHALL 使用 Vitest 与 Testing Library 运行，且测试数据全部为 synthetic 数据。
3. WHEN 批次 PR 触发 CI_Pipeline，THE CI_Pipeline SHALL 执行前端 ESLint、`tsc --noEmit`、Vitest、axe 检查、翻译键完整性检查、令牌对比度检查与 `Api_Types` 生成物差异检查。
4. WHEN 批次 PR 触发 CI_Pipeline，THE CI_Pipeline SHALL 保持 `python tools/validate_docs.py` 全部套件（required、fences、placeholders、coverage、consistency、semantics、regions、data-platform、temporal、observability、key-mgmt、channels、backup、yaml、json、schema、openapi、secrets）与既有 Go 模块检查通过。
5. THE 批次 PR SHALL 使用标题格式 `feat(web-{batch}): 简述（SCR-xx, FR-xxx）`，并在 PR 正文列出页面到 FR 的映射表。
6. WHEN 批次合入 main，THE 批次 SHALL 在同一 PR 内更新 `CHANGELOG.md` 的 Added 小节，引用 SCR 编号、FR 编号与批次标识。
7. WHEN 批次合入 main，THE 批次 SHALL 在 `IMPLEMENTATION_PLAN.md` 内更新前端交付批次的追踪行，使批次标识、SCR 编号、FR/NFR 编号互相可查。
8. THE 批次 SHALL 在合入前满足 IMPLEMENTATION_PLAN 第 6 节 DoD 全部条款：追踪已更新、契约一致且 CI 校验通过、测试含正常/异常/幂等、未违反 AGENTS.md 第 2 节禁令、文档与 CHANGELOG 同步、日志脱敏。

---

### B. 批次 0：工作区与 `apps/web` 脚手架（`task/frontend-batch-0-web-scaffold`）

**批次验收出口：** `pnpm install` 单次装依赖后，`apps/web` 可通过 lint、类型检查、单测与生产构建；根布局、语言前缀路由壳、设计令牌、全局错误页与 404 页可访问；CI 阶段 2 与阶段 6 已接入前端检查且阶段依赖链保持 1→2→3→4→5→6。

#### 需求 B0-1：pnpm 工作区与共享包骨架

**用户故事：** 作为前端实现者，我希望仓库有一个统一的工作区与共享包结构，以便两个应用共享令牌、组件与文案而不重复安装依赖。

**追踪：** 全部 SCR | 技术基线（Next.js + React + TypeScript） | IMPLEMENTATION_PLAN §5（EPIC-02、EPIC-09） | 批次 0

##### 验收标准

1. THE 仓库 SHALL 在根目录提供 `package.json` 与 `pnpm-workspace.yaml`，并将 `apps/*` 与 `packages/*` 纳入工作区。
2. THE 仓库 SHALL 提供 `packages/design-tokens`、`packages/ui`、`packages/i18n` 三个共享包，且每个包声明自身入口与类型声明。
3. THE 仓库 SHALL 维护单一 `pnpm-lock.yaml`，且 `apps/web` 与 `apps/admin` 的依赖解析自同一锁文件。
4. THE Web_App SHALL 使用 TypeScript `strict: true` 编译配置，且 `tsc --noEmit` 输出错误数为 0。
5. THE Web_App SHALL 提供 ESLint 配置并使 `eslint` 在零警告阈值下通过。
6. THE 仓库 SHALL 使 `node_modules/`、`.next/`、`dist/`、`coverage/` 保持在 `.gitignore` 忽略范围内。
7. WHEN 依赖被新增，THE 仓库 SHALL 使用固定版本号（精确版本或锁定范围）记录该依赖。

#### 需求 B0-2：路由壳、全局布局与全局错误页

**用户故事：** 作为求职者，我希望任何页面异常时都看到一致的说明与返回路径，以便我不会停在空白页面。

**追踪：** 全部 SCR | SCREEN-SPEC §4、§9 | ACCESSIBILITY §4.4 | 批次 0

##### 验收标准

1. THE Web_App SHALL 使用 App Router 在 `app/[locale]/` 下建立 SCR-01 ~ SCR-16 的路由壳，且每个路由壳渲染页面标题与占位说明。
2. THE Web_App SHALL 提供根布局，输出 `<html lang>`、跳转到主内容的 skip-link、全局导航与页脚，且导航项在全部页面顺序一致。
3. THE Web_App SHALL 提供全局 `error`、`not-found` 与 `loading` 视图；WHEN 渲染过程抛出未捕获错误，THE Web_App SHALL 渲染包含错误五要素与重试控件的错误视图。
4. WHEN 用户访问不存在的路由，THE Web_App SHALL 渲染 404 视图并提供返回工作台与返回落地页两个入口。
5. THE Web_App SHALL 使全局错误视图的展示内容限于错误码、请求标识与可读说明，不包含堆栈、令牌或用户内容。
6. THE Web_App SHALL 在 `zh-CN` 与 `en-US` 下分别渲染全局错误、404 与加载文案。

#### 需求 B0-3：CI 接入（阶段 2 检查与阶段 6 构建）

**用户故事：** 作为维护者，我希望前端检查与构建挂进既有流水线的指定阶段，以便前端问题在合入前被拦截且不影响其他阶段依赖。

**追踪：** 全部 SCR | IMPLEMENTATION_PLAN §6（DoD 第 2 项） | `.github/workflows/ci.yml` | 批次 0

##### 验收标准

1. THE CI_Pipeline SHALL 在 `stage2-lint` 作业中执行 `apps/web` 的 ESLint 与 `tsc --noEmit`。
2. THE CI_Pipeline SHALL 在 `stage6-build` 作业中执行 `apps/web` 的生产构建。
3. THE CI_Pipeline SHALL 保持 `stage2-lint` 与 `stage2-golangci` 并行、`stage3` 依赖两者、整体依赖链为 1→2→3→4→5→6。
4. IF 前端 ESLint、类型检查或构建返回非零退出码，THEN THE CI_Pipeline SHALL 使对应阶段失败并阻止后续阶段执行。
5. THE 批次 0 SHALL 作为独立 PR 合入，且该 PR 对 `.github/workflows/ci.yml` 的改动仅限于前端检查与前端构建的接入。

---

### C. 批次 1：核心页面 SCR-01 ~ SCR-07（`task/frontend-batch-1-core-pages`）

**批次验收出口：** SCR-01 ~ SCR-07 七个页面组各自具备五态视图、双语文案、axe 0 严重违规，并覆盖各页至少一条异常路径用例。

#### 需求 B1-1：SCR-01 落地页与样例演示

**用户故事：** 作为未登录访客，我希望先了解产品并试看一个样例演示，以便在不提交任何个人材料的情况下判断产品是否适合我。

**追踪：** SCR-01 | FR-027 | US-01（规则 1）、US-05（规则 1） | 批次 1

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}` 提供产品介绍页，在 `/{locale}/demo` 提供样例演示页。
2. THE Web_App SHALL 使样例演示页的全部内容来自 synthetic 样例数据，并渲染"样例数据"标识。
3. WHEN 访客在落地页触发上传个人材料的入口，THE Web_App SHALL 跳转到 `/{locale}/auth` 并在登录成功后返回原入口。
4. IF 样例演示资源加载失败，THEN THE Web_App SHALL 渲染错误五要素文案（说明不涉及任何个人数据）与重试控件。
5. THE Web_App SHALL 在落地页渲染年龄使用说明：未满 16 岁用户可使用的范围限于本页的无登录、无上传样例演示。
6. WHEN 用户刷新落地页，THE Web_App SHALL 恢复到介绍默认态。
7. THE Web_App SHALL 使落地页文案不包含录用结果预测、真实面试辅助或职位投递类承诺。

#### 需求 B1-2：SCR-02 登录/注册

**用户故事：** 作为求职者，我希望用邮箱验证码或第三方账号登录，并在第三方登录出问题时仍有可用路径，以便我不会被登录环节卡住。

**追踪：** SCR-02 | FR-027 | US-05（规则 2、3、9；场景 1、4） | 批次 1

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/auth` 提供四种登录方式入口：邮箱验证码、Google、Apple、微信，并在同页提供注册与登录的切换。
2. THE Web_App SHALL 使注册协议确认区域包含服务条款、隐私政策与数据处理说明三项，且录制、机构共享、模型训练授权不出现在注册确认区域内。
3. WHEN 用户请求邮箱验证码，THE Web_App SHALL 渲染验证码有效期与重发冷却剩余时间。
4. IF `Mock_Layer` 返回验证码错误或过期，THEN THE Web_App SHALL 渲染剩余尝试次数与重发冷却说明，并保留已填写的邮箱地址。
5. IF 第三方授权流程返回失败，THEN THE Web_App SHALL 渲染邮箱验证码替代路径入口。
6. THE Web_App SHALL 在注册流程中收集年龄声明，并渲染规则说明：正式个性化服务最低 16 岁；16 岁至当地成年年龄适用监护人验证流程。
7. WHEN 登录成功且存在登录前操作上下文，THE Web_App SHALL 返回该上下文对应的路由。
8. THE Web_App SHALL 使登录页不收集手机号作为必填字段。

#### 需求 B1-3：SCR-03 工作台

**用户故事：** 作为已登录求职者，我希望在工作台看到全部项目的状态与下一步动作，以便直接继续未完成的面试。

**追踪：** SCR-03 | FR-029、FR-030 | US-05（场景 2） | 批次 1

##### 验收标准

1. WHEN `/{locale}/dashboard` 收到包含项目列表的响应，THE Web_App SHALL 为每个项目卡片渲染岗位名称、当前轮次序号、状态徽标与下一动作控件。
2. THE Web_App SHALL 使项目卡片的状态徽标取值限于 15 项项目状态枚举，并为每个状态渲染对应的可用动作集合（继续面试、下一轮、报告、练习、正式重试、复制项目、下载、重命名、删除）。
3. THE Web_App SHALL 提供公司、岗位、日期、语言、状态五类筛选控件，并将筛选条件写入 URL 查询参数。
4. WHEN 用户刷新带筛选参数的工作台链接，THE Web_App SHALL 依据 URL 参数恢复筛选状态与结果。
5. WHEN 项目列表为空，THE Web_App SHALL 渲染空态说明与"新建面试"入口；WHEN 筛选结果为空，THE Web_App SHALL 渲染"清除筛选"入口。
6. WHILE 列表请求进行中，THE Web_App SHALL 渲染骨架屏。
7. IF 列表请求失败，THEN THE Web_App SHALL 渲染错误五要素文案（数据保留在账户中、不影响评分与额度）与重试控件。
8. WHEN 项目在其他设备处于活动面试状态，THE Web_App SHALL 在该项目卡片渲染"活动中"标识与安全转移入口。

#### 需求 B1-4：SCR-04 创建面试项目

**用户故事：** 作为求职者，我希望上传简历并粘贴 JD，并在文件被拒时不丢失已填写的 JD，以便我不必重复劳动。

**追踪：** SCR-04 | FR-001、FR-004、FR-006 | US-01（场景 4） | 批次 1

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/projects/new` 提供双栏输入布局：简历上传区与 JD 粘贴区。
2. THE Web_App SHALL 使上传控件接受 `.pdf`、`.doc`、`.docx` 且单文件上限为 10 MB，并渲染该限制说明。
3. THE Web_App SHALL 提供文件删除与替换控件，并渲染当前文件名与处理状态。
4. IF 上传被拒（损坏、加密、超过 10 MB、类型伪装或检测不安全），THEN THE Web_App SHALL 渲染具体拒绝原因，并保留 JD 输入框内已填写的全部文本。
5. IF 上传或解析触发请求失败，THEN THE Web_App SHALL 保留原始输入并渲染重试控件与错误五要素文案。
6. WHEN 用户离开并重新进入创建页，THE Web_App SHALL 依据草稿数据恢复到最后编辑状态。
7. THE Web_App SHALL 提供一键填充样例材料的入口，并将填充内容标记为样例。
8. WHEN 未登录访客访问创建页，THE Web_App SHALL 渲染权限不足态并提供登录入口。

#### 需求 B1-5：SCR-05 解析校对页

**用户故事：** 作为求职者，我希望逐字段校对解析结果、看清低置信度字段并了解材料缺失的影响，以便面试建立在我确认过的事实上。

**追踪：** SCR-05 | FR-002、FR-003、FR-005 | US-01（场景 1、2、3、5） | NFR-015 | 批次 1

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/projects/{id}/review` 渲染 `PARSING`、`MATERIAL_REVIEW`、`PARSE_FAILED` 三种项目状态对应的视图。
2. WHILE 项目状态为 `PARSING`，THE Web_App SHALL 渲染分步进度与预期时长文案（对齐 NFR-015 的 P95 ≤60 秒）。
3. WHEN 项目状态为 `MATERIAL_REVIEW`，THE Web_App SHALL 渲染结构化字段预览，并为每个字段提供新增、修改、删除控件。
4. THE Web_App SHALL 对低置信度字段渲染高亮标识与说明文案，且高亮标识在移除颜色样式后仍可通过图标或文字识别。
5. THE Web_App SHALL 提供"未纳入面试的敏感字段"说明区域，且该区域仅渲染类别名称清单（电话、邮箱、证件号、详细地址、照片、性别、婚育），不渲染任何字段值。
6. WHEN 材料存在缺失（仅 JD、仅简历、两者都缺失），THE Web_App SHALL 渲染对应降级模式说明弹窗，并使继续动作在用户显式勾选同意后才可用。
7. IF 解析失败，THEN THE Web_App SHALL 渲染失败原因、错误五要素文案、"重试失败步骤"与"进入手动编辑"两个入口，并保留原始输入。
8. WHEN 用户在确认前离开页面，THE Web_App SHALL 在重新进入时恢复到 `MATERIAL_REVIEW` 状态的编辑内容。
9. THE Web_App SHALL 使"生成计划"控件在用户完成材料确认动作前处于 disabled 状态并渲染禁用原因。

#### 需求 B1-6：SCR-06 面试计划页

**用户故事：** 作为求职者，我希望调整轮次、时长、难度、工具与便利设置并看到来源可信度，以便我在开始前理解并掌控整场流程。

**追踪：** SCR-06 | FR-007 ~ FR-012 | US-02（场景 1 ~ 4） | NFR-016 | ACCESSIBILITY §5 | 批次 1

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/projects/{id}/plan` 渲染 `PLAN_GENERATING`、`PLAN_REVIEW`、`PLAN_FAILED` 与已确认（只读快照）四种视图。
2. WHILE 项目状态为 `PLAN_GENERATING`，THE Web_App SHALL 按模块渲染生成进度与预期时长文案（对齐 NFR-016 的 P95 ≤120 秒）。
3. THE Web_App SHALL 渲染公开流程参考条目的来源链接、日期、来源类型与可信度，并对候选人经验类来源渲染"非官方"标识。
4. THE Web_App SHALL 提供轮次增删（范围 1 ~ 5）、拖动重排，以及角色、重点、难度（基础/标准/挑战）、时长（10 ~ 60 分钟）、风格、数字人与声音（固定授权角色库选项）的编辑控件，并提供"恢复 AI 推荐"控件。
5. THE Web_App SHALL 提供岗位工具（代码、白板、案例、作品集）的启停控件。
6. THE Web_App SHALL 提供 9 项便利设置的独立开关，默认值全部为关闭，且开关之间不存在联动勾选。
7. THE Web_App SHALL 使便利设置区域文案包含说明：便利设置不进入评分证据；文字模式下口语项标记为未评估且不记零。
8. THE Web_App SHALL 将统一评分算法、60 分门槛、解锁逻辑与跨轮交接规则渲染为只读说明区域，且该区域不提供编辑控件。
9. WHEN 任一轮次缺少问题覆盖方案或评分量表，THE Web_App SHALL 为该轮渲染"未就绪"标识，并使"确认并进行会前检查"控件处于 disabled 状态且渲染禁用原因。
10. IF 计划生成部分模块失败，THEN THE Web_App SHALL 保留成功模块的展示并仅为失败模块渲染重试控件。
11. WHEN 计划已确认，THE Web_App SHALL 渲染冻结说明与完整报价摘要（轮次、总时长、重试权益、税费、有效期），并使编辑控件处于 disabled 状态。
12. THE Web_App SHALL 使计划页不渲染正式问题文本与标准答案。

#### 需求 B1-7：SCR-07 会前检查页

**用户故事：** 作为求职者，我希望在开始前确认设备、网络、额度与便利设置，以便正式面试不会因为准备不足被打断。

**追踪：** SCR-07 | FR-015、FR-016、FR-031 | US-03 | NFR-007 | ACCESSIBILITY §5.2 | 批次 1

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/projects/{id}/precheck` 渲染五项检查：摄像头、麦克风、网络质量、音频播放、数字人建连，并为每项渲染通过、失败与重试三种状态。
2. THE Web_App SHALL 使摄像头与麦克风检查项提供"关闭并继续"路径，并渲染说明：关闭摄像头与麦克风不影响任何分数。
3. WHILE 数字人建连检查进行中，THE Web_App SHALL 渲染预期时长文案（对齐 NFR-007 的 95% ≤8 秒）。
4. IF 设备被占用或权限被拒绝，THEN THE Web_App SHALL 渲染具体原因与文字输入替代建议。
5. IF 网络质量检查未达标，THEN THE Web_App SHALL 渲染弱网说明与就近节点建议。
6. IF 数字人建连检查失败，THEN THE Web_App SHALL 渲染自动重试状态与客服入口。
7. THE Web_App SHALL 渲染 9 项便利设置的当前值与确认冻结控件，并在确认区域说明：本轮开始后便利设置不可修改，系统故障降级为唯一例外。
8. WHEN 项目存在上一轮已冻结的便利设置，THE Web_App SHALL 渲染继承值并允许在本轮开始前调整。
9. THE Web_App SHALL 渲染本轮额度预留信息；IF 额度不足，THEN THE Web_App SHALL 使"开始本轮"控件处于 disabled 状态并渲染购买入口与禁用原因。
10. WHEN 同一项目在其他设备处于活动状态，THE Web_App SHALL 渲染权限不足态说明与安全转移入口。
11. THE Web_App SHALL 使每个检查项提供独立重试控件，且重复触发同一检查项不产生重复的检查请求。

---

### D. 批次 2：实时面试房间外壳 SCR-08 / SCR-09（`task/frontend-batch-2-room-shell`）

**批次验收出口：** 房间功能布局与 4 类覆盖层变体可在 mock 事件驱动下渲染；媒体/数字人/ASR 全部为媒体桩；覆盖层重复触发具备幂等断言；键盘可完成房间全部控制。

#### 需求 B2-1：SCR-08 实时面试房间功能布局

**用户故事：** 作为求职者，我希望在房间里随时知道计时、网络、输入模式与下一步，并能用语音、文字或工具作答，以便我专注在回答本身。

**追踪：** SCR-08 | FR-013 ~ FR-019 | US-03（场景 1 ~ 3、5） | NFR-005、NFR-012 | 批次 2

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/sessions/{id}` 渲染五个功能区：顶部状态条（轮次名、计时状态、网络状态、字幕开关、退出）、问题与实时字幕区、候选人转写与追问区、右侧数字人视频区与候选人摄像头区、底部控制条（麦克风、摄像头、停止数字人、文字输入、提交）。
2. THE Web_App SHALL 渲染 10 项会话状态枚举对应的顶部状态文案，并在状态切换时更新计时是否进行中的说明。
3. THE Web_App SHALL 使数字人视频区与音频状态渲染为"始终开启"语义，并由媒体桩提供连接状态；WHILE 媒体桩状态为未连接，THE Web_App SHALL 渲染技术占位框与状态说明文案。
4. THE Web_App SHALL 使候选人摄像头与麦克风控件可切换为关闭，并在切换后渲染说明：关闭不影响分数。
5. THE Web_App SHALL 使问题文本固定显示且可回看，并将字幕区标记为 `aria-live="polite"`。
6. WHILE 当前回合处于修订窗口内，THE Web_App SHALL 提供转写修订控件并渲染说明：进入下一主问题后本轮转写冻结、评分使用修订文本、原始转写仅诊断保留。
7. WHEN 下一主问题开始，THE Web_App SHALL 使上一回合的修订控件处于 disabled 状态并渲染冻结说明。
8. THE Web_App SHALL 使房间内不存在手动暂停控件。
9. WHEN 用户触发退出，THE Web_App SHALL 渲染二次确认对话框并在确认文案中包含"将标记为评估未完成"。
10. THE Web_App SHALL 仅渲染计划中已启用的岗位工具；WHEN 某工具未在计划中启用，THE Web_App SHALL 使工具区不包含该工具的入口。
11. WHEN 检测到同一会话已有其他活动设备，THE Web_App SHALL 阻止本设备进入交互态并渲染安全转移提示。
12. THE Web_App SHALL 使停止数字人控件与提交控件位于 Tab 序前段，并使全部房间控制可仅通过键盘操作。
13. WHILE 视口宽度 <768px，THE Web_App SHALL 对代码编辑器与白板渲染"推荐使用桌面或平板"的提示与替代路径。
14. THE Web_App SHALL 使房间页面渲染的字幕与回答文本不进入前端日志与错误上报载荷。

#### 需求 B2-2：SCR-09 暂停/重连/降级覆盖层

**用户故事：** 作为遇到系统故障或断网的求职者，我希望明确知道计时、计费与评分是否受影响以及我能做什么，以便我不担心被判失败或被多扣费。

**追踪：** SCR-09 | FR-020、FR-032、FR-033 | US-03（场景 4） | NFR-006 | DOMAIN-002 §6.2、§7 | 批次 2

##### 验收标准

1. WHEN 会话状态变为 `PAUSED_SYSTEM`，THE Web_App SHALL 渲染系统故障暂停覆盖层，文案包含已暂停计时、正在自动恢复、此段时间不计费、不影响评分四项说明。
2. WHEN 会话状态变为 `RECONNECTING`，THE Web_App SHALL 渲染 3 分钟倒计时覆盖层，文案包含"倒计时内可恢复到最后已确认回合"、"不扣时间"、"不判失败"三项说明。
3. WHEN 重连倒计时归零，THE Web_App SHALL 渲染"已标记为评估未完成"说明与重试入口。
4. WHEN 会话状态变为 `DOWNGRADE_PROMPTED`，THE Web_App SHALL 渲染降级询问覆盖层，包含同意与拒绝两个动作，并渲染两种选择的后果说明。
5. WHEN 用户选择同意降级，THE Web_App SHALL 切换到文字继续的房间视图，并渲染说明：自故障点起不再消耗数字人额度、口语项标记为未评估。
6. WHEN 用户选择拒绝降级，THE Web_App SHALL 渲染结束说明，包含"标记为评估未完成（这不是失败）"与"系统责任全额返还本轮预留额度"两项文案。
7. WHEN 会话状态变为 `AUTH_PAUSED`，THE Web_App SHALL 渲染重新认证覆盖层，文案包含计时已暂停、不扣时间、不判失败三项说明。
8. WHEN 同一覆盖层触发事件在 5 秒内重复到达 3 次，THE Web_App SHALL 保持同一时刻仅渲染一个覆盖层实例，且用户可见的额度与计时说明数值保持不变。
9. WHEN 用户在覆盖层上重复点击同一动作控件，THE Web_App SHALL 以同一幂等键发出至多一次请求，并使该控件在请求进行中处于 disabled 状态。
10. THE Web_App SHALL 将覆盖层容器标记为 `role="alertdialog"`、将故障与降级提示标记为 `role="alert"`，并在覆盖层内圈定焦点、关闭后返回原触发元素。
11. THE Web_App SHALL 使四类覆盖层文案在 `zh-CN` 与 `en-US` 下均有正式译文，并对第 1 ~ 7 条的关键文案建立快照测试。

---

### E. 批次 3：结果、报告与账户 SCR-10 ~ SCR-15（`task/frontend-batch-3-results-report`）

**批次验收出口：** 六个页面组具备五态视图与固定文案快照；图表均有文字与表格等价版本；练习页与报告导出的红线标识存在。

#### 需求 B3-1：SCR-10 轮次结果页

**用户故事：** 作为完成一轮面试的求职者，我希望立刻知道是否通过、依据是什么、下一步做什么，以便我能马上继续或开始复盘。

**追踪：** SCR-10 | FR-021、FR-022 | US-04（场景 1、2） | NFR-013 | 批次 3

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/projects/{id}/rounds/{n}/result` 渲染 `SCORING`、`ROUND_PASSED`、`ROUND_FAILED`、`EVALUATION_INCOMPLETE` 四种状态视图。
2. WHILE 轮次状态为 `SCORING`，THE Web_App SHALL 渲染评分进行中提示与预期时长文案（对齐 NFR-013 的 P95 ≤60 秒）。
3. WHEN 轮次状态为 `ROUND_PASSED`，THE Web_App SHALL 在首屏第一条信息位置渲染"恭喜你通过本轮面试，已进入下一轮"，并渲染总分、60 分线、关键维度通过情况、优势与注意点、流程进度、下一轮角色/重点/难度/时长，以及"立即进入下一轮"与"稍后继续"两个动作。
4. WHEN 轮次状态为 `ROUND_FAILED`，THE Web_App SHALL 渲染累计纪要与证据摘要，并渲染"复盘练习"、"正式重试"、"查看报告"、"结束并生成部分报告"四个动作；THE Web_App SHALL 使进入下一轮的动作在该状态下处于 disabled 状态并渲染禁用原因。
5. WHEN 轮次状态为 `EVALUATION_INCOMPLETE`，THE Web_App SHALL 渲染"这不是失败"说明、原因分类（证据不足、系统故障、用户结束）、已保留内容与重试入口；WHERE 原因为系统责任，THE Web_App SHALL 渲染额度已返还说明。
6. IF 评分请求失败，THEN THE Web_App SHALL 渲染评估未完成说明与"恢复后自动重算"文案，并提供手动重试控件。
7. THE Web_App SHALL 使结果页不渲染后续轮次的完整标准答案。
8. THE Web_App SHALL 将正式复核入口渲染在报告页而非结果页。

#### 需求 B3-2：SCR-11 完整报告页

**用户故事：** 作为求职者，我希望看到分模块的证据化报告并能导出，以便我知道每个分数的来源以及要练什么。

**追踪：** SCR-11 | FR-023、FR-025、FR-026 | US-04（场景 5、6） | NFR-014 | ACCESSIBILITY §4.1 | 批次 3

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/projects/{id}/report` 渲染以下模块：状态与总分、各轮轨迹（含交接影响、能力锁定与分数变化）、岗位匹配度、六维雷达、逐题证据、沟通分析、工具表现、优势与风险、优先训练计划、原始记录与删除入口。
2. THE Web_App SHALL 为六维雷达、趋势与岗位匹配三类图形同时渲染文字摘要与数据表格等价版本。
3. WHERE 项目缺少 JD，THE Web_App SHALL 使岗位匹配模块不渲染百分比数值，并渲染缺失说明。
4. THE Web_App SHALL 将岗位匹配的必备项与加分项分列渲染。
5. THE Web_App SHALL 在沟通分析模块渲染当前输入模式与对应证据限制说明；WHERE 输入模式为文字，THE Web_App SHALL 将口语项渲染为"未评估"而不渲染数值 0。
6. WHILE 报告生成进行中，THE Web_App SHALL 渲染进度与预期时长文案（对齐 NFR-014 的 P95 ≤120 秒）。
7. WHEN 报告类型为部分报告，THE Web_App SHALL 渲染可用模块、标注缺失模块，并为每个缺失模块提供独立重试控件。
8. IF 单个模块请求失败，THEN THE Web_App SHALL 仅在该模块位置渲染错误五要素文案与重试控件，并保持其他模块可读。
9. THE Web_App SHALL 提供复盘练习、正式重试、申请复核、下载、删除五个动作；THE Web_App SHALL 在每次正式尝试已使用复核后使申请复核控件处于 disabled 状态并渲染原因。
10. WHEN 用户查看复核结果，THE Web_App SHALL 渲染复核前结果、复核后结果与改变原因三项内容。
11. WHEN 用户触发下载，THE Web_App SHALL 在导出预览与导出产物展示区域渲染标记"模拟训练结果，不代表真实企业录用结论"。
12. WHEN 访问者不是项目所有人，THE Web_App SHALL 渲染权限不足态，且响应文案不揭示该项目是否存在于他人账户。

#### 需求 B3-3：SCR-12 练习页

**用户故事：** 作为需要复盘的求职者，我希望练习环境明确与正式评分隔离，以便我可以放心反复尝试。

**追踪：** SCR-12 | FR-024 | US-04（场景 3） | 批次 3

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/projects/{id}/practice/{pid}` 渲染练习进行中与练习结束两种视图。
2. WHILE 用户处于练习页，THE Web_App SHALL 持续渲染固定标识"练习不改变正式分数与解锁状态"。
3. THE Web_App SHALL 提供原题练习与变体练习两类入口，并提供提示、框架与示例的查看控件。
4. THE Web_App SHALL 在练习页提供暂停控件与结束练习控件。
5. WHEN 练习结束，THE Web_App SHALL 渲染逐步反馈内容，并渲染说明：本次练习不产生正式评分证据。
6. IF 练习内容生成失败，THEN THE Web_App SHALL 渲染重试控件与错误五要素文案，其中"是否影响评分"一项说明为不影响任何正式记录。

#### 需求 B3-4：SCR-13 资产与历史页

**用户故事：** 作为跨设备使用的求职者，我希望集中管理简历、岗位、面试记录与训练进度，以便我能复用材料并继续未完成的项目。

**追踪：** SCR-13 | FR-029、FR-030 | US-05（场景 2、3） | 批次 3

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/library` 渲染四个分区：简历库、岗位库、面试记录、训练进度，并为每个分区提供独立的空态与失败态。
2. THE Web_App SHALL 为简历条目提供解析记录查看、编辑（生成新版本）、替换与删除控件，并渲染版本标识。
3. THE Web_App SHALL 为岗位条目提供编辑与版本查看控件。
4. THE Web_App SHALL 为面试记录提供按公司、岗位、日期、语言、状态筛选，并提供"跨设备继续"入口。
5. WHEN 某项目在其他设备处于活动面试状态，THE Web_App SHALL 渲染"活动中"标识、阻止直接进入，并提供经确认的安全转移路径。
6. WHEN 用户删除处于活动状态的项目，THE Web_App SHALL 在确认对话框中渲染说明"将终止正在进行的面试"。
7. WHILE 删除任务进行中，THE Web_App SHALL 渲染任务真实进度分项（数据库、对象存储、第三方处理），且不在任务完成前渲染完成状态。
8. THE Web_App SHALL 在训练进度分区渲染薄弱点与正式重试轨迹，并使练习记录与正式记录在视觉与文字上分区标识。

#### 需求 B3-5：SCR-14 账户与隐私页

**用户故事：** 作为求职者，我希望独立控制每类授权、语言设置、数据导出与账户删除，以便我随时掌握自己的数据。

**追踪：** SCR-14 | FR-027、FR-028、FR-040 | US-05（场景 4、5） | 批次 3

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/settings` 渲染五个分区：登录身份、语言设置、授权中心、数据导出、删除账户。
2. THE Web_App SHALL 在登录身份分区提供绑定与解绑控件，并渲染说明：绑定需双侧分别验证。
3. IF 身份绑定返回冲突，THEN THE Web_App SHALL 渲染"未执行合并"说明与人工支持入口。
4. THE Web_App SHALL 将界面语言与面试语言渲染为两个独立设置控件。
5. THE Web_App SHALL 在授权中心渲染六类独立授权开关（核心服务必要处理、保存原始音视频、机构共享、非必要产品分析、模型训练或研究、营销通知），每项提供撤回控件并渲染"撤回即时生效"说明。
6. THE Web_App SHALL 使授权开关的默认渲染值来自服务端返回值，且页面不提供批量开启全部授权的控件。
7. WHEN 用户触发数据导出或删除账户，THE Web_App SHALL 渲染重新验证步骤，并在删除确认文案中说明级联删除或不可逆匿名化，以及法定财务记录保留但解除内容关联。
8. WHILE 导出或删除任务进行中，THE Web_App SHALL 渲染真实进度；IF 任务失败，THEN THE Web_App SHALL 渲染失败原因与重试控件，且不渲染完成状态。

#### 需求 B3-6：SCR-15 购买与额度页

**用户故事：** 作为求职者，我希望在购买前看到完整报价与逐笔账单，以便我清楚每一秒钱花在哪里。

**追踪：** SCR-15 | FR-031、FR-032、FR-033 | US-06（场景 1 ~ 5） | NFR-006 | 批次 3

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/billing` 渲染四个分区：报价、支付、额度流水、订阅管理。
2. THE Web_App SHALL 在报价分区渲染轮次数、总时长、正式重试权益、有效期与税费五项，并按 `locale` 本地化货币与数字格式。
3. THE Web_App SHALL 渲染四类权益选项：免费额度、单项目包、Pro、加油包。
4. THE Web_App SHALL 使自动续费为独立勾选控件，默认值为未勾选，并渲染扣款前提醒说明。
5. THE Web_App SHALL 在额度流水分区为每条记录渲染项目、轮次、开始时间、使用秒数、返还原因与余额六个字段。
6. WHEN 订单状态为处理中，THE Web_App SHALL 渲染"处理中"状态并使再次支付控件处于 disabled 状态，且渲染禁用原因为"避免重复扣款"。
7. IF 支付失败，THEN THE Web_App SHALL 保留已选择的方案并渲染重新支付入口。
8. WHEN 存在系统识别的重复扣款记录，THE Web_App SHALL 渲染自动退回状态与通知说明。
9. THE Web_App SHALL 提供发票或收据、取消续费、退款或故障申诉三个入口。
10. THE Web_App SHALL 渲染说明：数据删除、隐私控制、证据解释、无障碍与故障恢复不收费，且付费不改变评分标准。
11. WHEN 用户重复提交同一支付动作，THE Web_App SHALL 以同一幂等键发出至多一次请求。

---

### F. 批次 4：机构端与运营后台 SCR-16 / SCR-17（`task/frontend-batch-4-org-admin`）

**批次验收出口：** 机构端页面组默认最小可见并通过"不可显示失败"与"小样本保护"用例；`apps/admin` 独立工程可构建，7 个骨架页面存在且无改分类控件。

#### 需求 B4-1：SCR-16 机构端页面组

**用户故事：** 作为机构训练负责人，我希望看到任务完成情况与群体趋势；作为参加训练的求职者，我希望机构默认看不到我的内容，以便组织化训练不变成排名或筛选。

**追踪：** SCR-16 | FR-034、FR-035、FR-036 | US-07（场景 1 ~ 5） | 批次 4

##### 验收标准

1. THE Web_App SHALL 在 `/{locale}/org/{orgId}/…` 提供七个页面：机构任务列表、任务配置、完成情况、成员与邀请、聚合趋势、权限与审计、授权结果视图。
2. THE Web_App SHALL 使完成情况页默认渲染字段限于：任务是否接受、未开始、进行中、已完成或退出、完成时间、系统故障、机构额度消耗。
3. WHEN 某成员未授权结果分享，THE Web_App SHALL 为该成员渲染"已完成但未共享结果"，且渲染内容中不包含失败状态、分数、报告正文或媒体引用。
4. THE Web_App SHALL 使任务配置页的可配置项限于岗位或岗位类别、材料要求、轮次、角色、时长、难度、语言、工具、截止时间、练习次数、机构额度。
5. THE Web_App SHALL 使 60 分线、统一评分算法、保护属性规则、证据标准、跨轮解锁、正式复核六项在机构端页面中既不渲染为可编辑控件也不渲染为配置项。
6. WHEN 聚合趋势的某个细分群体人数 <10，THE Web_App SHALL 隐藏该细分并渲染小样本保护说明。
7. THE Web_App SHALL 使机构端页面不提供个人排行榜、个人排名与候选人搜索控件。
8. THE Web_App SHALL 在成员与邀请页渲染加入前告知内容：机构名称、可见数据范围、训练期限、退出影响。
9. WHEN 成员状态为已退出机构，THE Web_App SHALL 使该成员的数据入口从机构端视图中移除。
10. THE Web_App SHALL 在授权结果视图仅渲染用户已授权的范围与有效期（例如"仅雷达图 30 天"），并对已过期授权渲染失效状态。
11. THE Web_App SHALL 在权限与审计页渲染六类角色（所有者、管理员、指导老师、隐私审计、财务、求职者）的权限矩阵与访问记录列表。
12. IF 机构相关请求返回未授权，THEN THE Web_App SHALL 渲染权限不足态，且文案不揭示他人数据是否存在。

#### 需求 B4-2：SCR-17 运营后台骨架（`apps/admin`）

**用户故事：** 作为平台运营人员，我希望后台按职责分区且默认脱敏，以便运维工作不接触用户内容、也无法改动个人结果。

**追踪：** SCR-17 | FR-037、FR-038、FR-039、FR-040 | US-08（场景 1 ~ 3） | 批次 4

##### 验收标准

1. THE Admin_App SHALL 作为独立 Next.js 工程存在于 `apps/admin`，共享 `packages/design-tokens`、`packages/ui`、`packages/i18n`，并使用 `basePath = /admin`。
2. THE Admin_App SHALL 在 `/admin/{locale}/…` 提供七个骨架页面：区域监控、供应商与模型、版本治理、来源与内容安全、客服工单、财务与补偿、审计日志（`basePath = /admin` 前置语言前缀，见 design.md 第 2.4 节路径裁决）。
3. THE Admin_App SHALL 使区域监控页默认渲染字段限于匿名会话编号与技术指标（在线房间数、排队数、容量、供应商延迟、错误率、SLO 与错误预算）。
4. THE Admin_App SHALL 使全部页面的可交互控件集合中，编辑个人分数、编辑解锁状态、编辑证据正文三类控件数量为 0。
5. WHERE 某骨架页面的操作按钮对应尚未实现的后端能力，THE Admin_App SHALL 将该按钮渲染为 disabled 状态并渲染禁用原因，或使该操作区域保持为空。
6. THE Admin_App SHALL 在客服工单页默认渲染字段限于账户状态、订单、额度、故障代码与用户主动提交材料标识，并渲染说明：查看逐字稿需用户针对会话授权、媒体访问需用户申请与双人审批。
7. THE Admin_App SHALL 在审计日志页渲染追加式记录列表，且不提供删除或编辑审计记录的控件。
8. THE Admin_App SHALL 在版本治理页与供应商页渲染发布门槛说明（离线评测、影子运行、灰度、放量、回滚）与只读状态，不提供跳过门槛的控件。
9. THE Admin_App SHALL 使全部页面文案在 `zh-CN` 与 `en-US` 下均可渲染，并满足需求 G4 的无障碍基线。
10. WHEN 用户访问 `Admin_App` 的任一页面且 `Mock_Layer` 返回未授权，THE Admin_App SHALL 渲染权限不足态与所需角色说明。
11. THE Admin_App SHALL 使页面渲染内容与前端日志载荷不包含姓名、简历正文、回答正文与媒体引用。

---

## 对已批准规范的偏离说明

以下两项是对已批准规范的**扩展或等价替代**，需在对应批次的 PR 正文中显式说明；本特性**不修改**任何已批准规范文档正文。若评审认为需要固化，则由文档负责人另起 PR 更新规范并同步追踪关系。

### 偏离 1：语言前缀路由

- **已批准内容**：`docs/design/SCREEN-SPEC.md` 第 5 节"路由建议"列出的路径为 `/`、`/auth`、`/dashboard`、`/projects/{id}/…` 等无语言前缀形式。
- **本特性实现**：在全部路由前增加语言前缀 `/{locale}`（取值 `zh-CN`、`en-US`），例如 `/zh-CN/dashboard`、`/en-US/projects/{id}/plan`。
- **理由**：FR-028 要求中英文界面独立配置，语言前缀使界面语言在 URL 层可分享、可缓存、可直达，并使缺失翻译在构建期可检测。
- **性质**：SCREEN-SPEC 第 5 节路径为"路由建议"，本项属**扩展**而非冲突；无前缀路径由 `I18n_Runtime` 重定向到带前缀等价路径，原建议路径仍可访问。
- **披露要求**：批次 0 PR 正文列出前缀规则与重定向行为；后续批次 PR 引用本说明。

### 偏离 2：以测试断言替代 Storybook

- **已批准内容**：`docs/design/DESIGN-SYSTEM.md` 第 12 节第 2 项要求"组件状态清单在 Storybook 或等价物中逐组件截图回归"。
- **本特性实现**：不引入 Storybook；改用 Vitest + Testing Library 对第 8 节七态逐组件断言渲染输出、可聚焦性与无障碍属性，叠加令牌对比度校验与 axe 检查，全部接入 CI。
- **理由**：第 12 节已明示"或等价物"；测试断言在 CI 中可判定通过或失败，截图回归在无品牌视觉（OD-04 未决）阶段的收益有限且维护成本高。
- **性质**：属第 12 节允许的**等价物**，不改变第 8 节状态清单本身。
- **披露要求**：批次 0 PR 正文说明等价物构成与覆盖范围；若 Design Lead 在 OD-04 关闭时要求截图回归，则另起任务补充。

---

## 明确不在范围

以下内容不属于本特性交付，任何批次不得实现：

1. **真实 API 联调**：不实现后端服务、不改动 `services/**`、不修改 `docs/api/openapi.yaml`；页面请求由 `Mock_Layer` 拦截。
2. **真实媒体、数字人与 ASR**：不接入 WebRTC/SFU、TTS、数字人驱动或流式 ASR 供应商；SCR-08 与 SCR-09 使用媒体桩。同时禁止用静态头像、预录视频、PPT 或纯文字冒充实时数字人输出。
3. **实时性能达标**：NFR-007 ~ NFR-012 的实时指标验证属实时链路任务（EPIC-03、EPIC-10），本特性仅在加载与等待文案中引用对应指标数值。
4. **评分、报告、计费、机构与治理的服务端逻辑**：分数计算、复核、额度扣减与返还、授权判定、审计写入均由后端负责，前端只做展示与请求。
5. **最终品牌视觉**：logo、品牌色、插画、图标体系属 OD-04 未决事项（Owner = Design Lead），令牌只使用 DESIGN-SYSTEM 第 5 节中性占位值。
6. **PRD Out of Scope 能力**：招聘网站抓取、职位聚合与自动投递、雇主 ATS 与候选人筛选、真实面试作弊 Copilot、未授权肖像或声音克隆、PPT 或预录课堂模式。
7. **原生应用与第三种语言**：不交付 iOS/Android 原生端，不新增 `zh-CN`、`en-US` 以外的语言资源。
8. **深色配色主题**：不作为本特性交付项（红线针对的是诱导性设计模式，与配色主题无关）。
9. **人工无障碍验证**：屏幕阅读器人工走查与残障用户可用性测试属 TASK-094 范围；本特性交付自动化检查与实现基线。

---

## 追踪矩阵总览

| 需求 | 页面组 | FR | US | NFR | 批次 |
|---|---|---|---|---|---|
| G1 状态枚举一致性 | 全部 | FR-009、FR-021 | US-04、US-05 | — | 0 ~ 4 |
| G2 五态与错误五要素 | 全部 | 全部（错误规则） | US-01 ~ US-08 | — | 0 ~ 4 |
| G3 双语与语言前缀路由 | 全部 | FR-028 | US-05 | — | 0 ~ 4 |
| G4 无障碍基线 | 全部 | FR-016、FR-018 | US-03 | 无障碍硬门槛 | 0 ~ 4 |
| G5 设计令牌与组件七态 | 全部 | — | — | — | 0 ~ 4 |
| G6 数据层 | 全部 | — | — | — | 0 ~ 4 |
| G7 响应式与平台差异 | 全部 | — | — | — | 0 ~ 4 |
| G8 前端安全与隐私边界 | 全部 | FR-003、FR-039、FR-040 | US-08 | 观测性脱敏 | 0 ~ 4 |
| G9 红线禁止项 | 全部 | FR-003、FR-013、FR-039 | US-03、US-08 | — | 0 ~ 4 |
| G10 固定文案红线 | SCR-10 ~ 12 | FR-021、FR-022、FR-024、FR-026 | US-04 | — | 3 |
| G11 测试、CI 与 DoD | 全部 | — | — | NFR-006（幂等用例） | 0 ~ 4 |
| B0-1 工作区与共享包 | — | — | — | — | 0 |
| B0-2 路由壳与全局错误页 | 全部 | — | — | — | 0 |
| B0-3 CI 接入 | — | — | — | — | 0 |
| B1-1 落地页与样例演示 | SCR-01 | FR-027 | US-01、US-05 | — | 1 |
| B1-2 登录/注册 | SCR-02 | FR-027 | US-05 | — | 1 |
| B1-3 工作台 | SCR-03 | FR-029、FR-030 | US-05 | — | 1 |
| B1-4 创建面试项目 | SCR-04 | FR-001、FR-004、FR-006 | US-01 | — | 1 |
| B1-5 解析校对页 | SCR-05 | FR-002、FR-003、FR-005 | US-01 | NFR-015 | 1 |
| B1-6 面试计划页 | SCR-06 | FR-007 ~ FR-012 | US-02 | NFR-016 | 1 |
| B1-7 会前检查页 | SCR-07 | FR-015、FR-016、FR-031 | US-03 | NFR-007 | 1 |
| B2-1 实时面试房间布局 | SCR-08 | FR-013 ~ FR-019 | US-03 | NFR-005、NFR-012 | 2 |
| B2-2 暂停/重连/降级覆盖层 | SCR-09 | FR-020、FR-032、FR-033 | US-03 | NFR-006 | 2 |
| B3-1 轮次结果页 | SCR-10 | FR-021、FR-022 | US-04 | NFR-013 | 3 |
| B3-2 完整报告页 | SCR-11 | FR-023、FR-025、FR-026 | US-04 | NFR-014 | 3 |
| B3-3 练习页 | SCR-12 | FR-024 | US-04 | — | 3 |
| B3-4 资产与历史页 | SCR-13 | FR-029、FR-030 | US-05 | — | 3 |
| B3-5 账户与隐私页 | SCR-14 | FR-027、FR-028、FR-040 | US-05 | — | 3 |
| B3-6 购买与额度页 | SCR-15 | FR-031 ~ FR-033 | US-06 | NFR-006 | 3 |
| B4-1 机构端页面组 | SCR-16 | FR-034 ~ FR-036 | US-07 | — | 4 |
| B4-2 运营后台骨架 | SCR-17 | FR-037 ~ FR-040 | US-08 | — | 4 |

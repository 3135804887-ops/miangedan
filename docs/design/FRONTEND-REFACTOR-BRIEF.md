# 面个蛋前端重构简报（给 opus-5 的实现输入）

- 文档编号：DESIGN-FE-001
- 版本：0.1.0（2026-08-04）
- 追踪：SCR-01 ~ SCR-17；docs/design/DESIGN-SYSTEM.md；docs/design/ACCESSIBILITY.md；WCAG 2.2 AA（axe 门禁）
- 一致性锚点：`apps/web`（Next.js App Router）、`packages/ui`（@mgd/ui）、`packages/design-tokens`、`packages/i18n`、`apps/web/src/lib/api-fetch.ts`

## 1. 技术基线（重构必须保持）

- Next.js App Router + React，TypeScript strict，pnpm workspace，Tailwind 风格 className（现有 token 体系）。
- 设计令牌：`@mgd/design-tokens` 的 `--mgd-app-brand-ink/from/to`、`--mgd-app-surface-muted`、`--mgd-app-shadow-*` 等；视觉语言延续「深色品牌渐变 + 玻璃拟态 + 霓虹点缀」。
- UI 组件统一来自 `@mgd/ui`（Button、Tint、Icon* 等）；不引入新的 UI 库，除非评审批准。
- 文案一律走 `packages/i18n`（zh-CN / en-US 双语，`i18n:check` 门禁）；页面标签必须从 `labels` 或 locale 内联双语取。
- 数据请求统一 `apiFetch`（真实 API 未就绪时用 `src/mocks` 合成数据，标注 synthetic）。
- 门禁：`pnpm lint`、`pnpm typecheck`、`pnpm test`（含 axe 页面级扫描，WCAG 2.2 AA）、`pnpm i18n:check`、`pnpm api:check`。

## 2. 页面清单（全部 23 个路由，按 SCR 分组）

### 公开区（SCR-01/02）

- `/`（SCR-01）：落地页。核心区块：Hero（品牌 + 数字面试官形象 + 主 CTA）、三步流程、能力亮点、评测/合规背书、页脚。
- `/demo`（SCR-01）：产品演示页。核心区块：演示流程步骤、交互说明、跳转进入真实面试房间的 CTA。
- `/auth`（SCR-02）：登录/注册。核心区块：邮箱/微信/Apple 登录、同意条款、语言切换。

### 工作台（SCR-03/04/05/06）

- `/dashboard`（SCR-03）：仪表盘。核心区块：欢迎区、最近项目卡片、进行中面试、关键指标（完成率/待复核）、快捷入口。
- `/projects/new`（SCR-04）：新建项目。核心区块：多步表单（岗位信息/面试轮次/覆盖点/便利设置）、进度指示、草稿保存。
- `/projects/[id]/review`（SCR-05）：项目复核/人工审阅。核心区块：候选人列表、待复核队列、证据时间线、评分修正与批注。
- `/projects/[id]/plan`（SCR-06）：面试计划。核心区块：轮次时间线、每轮题目与覆盖点、评分维度权重、生成计划/再生成。

### 面试流程（SCR-07/08/09/10/11/12）

- `/projects/[id]/precheck`（SCR-07）：会前检查。核心区块：摄像头/麦克风/网络/扬声器/数字人五检、便利设置冻结、额度预留说明。
- `/sessions/[id]`（SCR-08/09）：实时面试房间（核心页面）。核心区块：顶部状态栏（状态/计时/网络/字幕开关/退出）、左侧问题与候选人转写/修订、右侧数字人视频（静态形象 + 音频）、工具面板（code_editor/whiteboard/case/portfolio）、打断与降级覆盖层。已实现功能：播放问候语、转写演示（edge-tts→FunASR）、打断演示（VAD）。
- `/projects/[id]/rounds/[n]/result`（SCR-10）：单轮结果。核心区块：本轮维度得分、答案与转写、证据引用、可修订说明。
- `/projects/[id]/report`（SCR-11）：项目报告。核心区块：总体评分与结论、六维评分卡（进度条/雷达图）、逐轮明细、证据时间线、导出。
- `/projects/[id]/practice/[pid]`（SCR-12）：练习会话。核心区块：练习题目、输入模式、即时反馈、与正式面试的区别提示。

### 素材与账户（SCR-13/14/15）

- `/library`（SCR-13）：题库/素材库。核心区块：题目分类筛选、覆盖点标签、收藏/自定义、批量导入。
- `/settings`（SCR-14）：设置。核心区块：个人资料、隐私与授权中心、证据删除/导出、通知、语言。
- `/billing`（SCR-15）：账单/套餐。核心区块：当前套餐、用量、发票记录、支付方式、升降级。

### 机构与管理（SCR-16/17）

- `/org/[orgId]/*`（SCR-16，7 个子页）：aggregates（聚合分析）、assignments（任务分配/详情）、completion（完成情况）、members（成员）、permissions（权限）、shares（分享）。
- `/admin`（SCR-17）：平台管理后台。核心区块：租户/用量/审计、功能开关（feature flags）、状态页、运营配置。

## 3. 设计风格参考（按页面场景）

### 3.1 品牌基调

参考「AI 产品高级感」：OpenAI、Claude（Anthropic）、DeepL、Perplexity。
- 借鉴：深色渐变背景 + 大标题 + 单一发光 CTA；Hero 区放产品实拍/真人数字面试官形象；科技感强但克制，避免过度霓虹。
- 不照搬：不做赛博朋克化；保留可读性与专业面试氛围。

### 3.2 实时面试房间

参考「在线会议 + AI 面试」：Zoom、腾讯会议、Google Meet、HireVue、Paradox。
- 借鉴：顶部会话状态栏（连接状态/计时/网络/录制）、左侧主内容（问题/转写/工具）、右侧数字人视频与字幕；会前设备检测流程；打断/降级的明确反馈。
- 不照搬：会议室式顶部工具栏堆叠；保持候选人端注意力集中、信息密度适中。

### 3.3 仪表盘与项目管理

参考「HR SaaS / 现代工作台」：HireVue、TestGorilla、Mercor、飞书招聘。
- 借鉴：卡片式项目列表、状态徽章、进度条；「待复核」入口醒目；指标区简洁可扫读。
- 不照搬：企业后台常见的密集表格堆叠；首页保持 3 秒内看懂下一步动作。

### 3.4 报告与评分

参考「评分报告 SaaS」：TestGorilla、HireVue Report、Stripe/Linear 的数据呈现。
- 借鉴：六维评分条/雷达图、结论徽章（通过/待复核）、证据时间线（点击回看转写）、导出一键。
- 不照搬：纯表格报告；用视觉权重突出「结论 + 依据」。

### 3.5 后台/机构/设置

参考「开发者后台」：Linear、Stripe Dashboard、Vercel。
- 借鉴：左侧导航 + 内容区、紧凑表格、权限矩阵清晰、功能开关（feature flags）带状态与审计提示。
- 不照搬：过于工具化的深色 UI；保持与品牌一致但更克制。

## 4. 全局设计原则

- 一致性：所有页面使用同一套 design tokens 与 @mgd/ui 组件；间距遵循 8pt 网格。
- 层次：页面主操作唯一化（每屏一个主 CTA）；信息按「结论 → 依据 → 操作」排列。
- 密度：工作台可信息密集，面试房间保持宽松、低干扰。
- 无障碍：对比度、键盘焦点、`aria-live`（字幕/状态）、`prefers-reduced-motion` 全部满足 WCAG 2.2 AA（axe 门禁会拦）。
- 动效：150~250ms ease-out；状态切换用淡入/位移动效；尊重系统减少动态。
- 响应式：桌面优先；移动端保证核心流程（进入房间、看报告）可用。
- 双语：zh-CN / en-US 同时可用，文案长度差异预留（中文短、英文长）。

## 5. 硬约束（opus-5 实现时必须遵守）

- 不改路由结构与 SCR 映射；不删除既有 i18n keys（新增 key 需成对加入两种语言）。
- 不换 UI 库；组件优先复用 `@mgd/ui`；新增基础组件放入 `packages/ui` 并补齐测试。
- 保留房间页已有功能按钮：播放问候语、转写演示、打断演示、字幕开关、退出/暂停/降级覆盖层。
- 数据仍走 `apiFetch` + mock（`src/mocks`），不硬编码后端地址；`NEXT_PUBLIC_SELFHOST_TTS_URL` 等环境变量沿用。
- 每次交付必须通过：`pnpm lint`、`pnpm typecheck`、`pnpm test`、`pnpm i18n:check`。
- 动效不得影响 axe 与交互测试；避免装饰性元素造成对比度/焦点问题。

## 6. 交付方式

- 建议先出三张风格稿：落地页（`/`）、面试房间（`/sessions/[id]`）、项目报告（`/projects/[id]/report`），确认视觉方向后铺开到其余页面。
- 可按 SCR 分批交付（公开区 → 工作台 → 面试流程 → 报告 → 账户 → 机构/后台），每批一个 PR，由项目侧融合与验收。

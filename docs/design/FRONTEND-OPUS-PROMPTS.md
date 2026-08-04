# 面个蛋前端持续工作提示词（opus-5 版）

- 文档编号：DESIGN-FE-002
- 版本：0.1.0（2026-08-04）
- 配套：docs/design/FRONTEND-REFACTOR-BRIEF.md（页面清单/风格参考/硬约束，必须先读）
- 追踪：SCR-01 ~ SCR-17；IMPLEMENTATION_PLAN.md；PR 融合由主窗口执行

## 0. 使用方式

- 每个新窗口开头：粘贴「主提示词」+ 当前「批次提示词」，然后开始实现。
- 每批完成后：粘贴「完成报告」回主窗口（本窗口对话或其他协作方式）。
- 每批只做该批页面；未列出的页面/模块一律不动。
- 所有门禁命令必须在提交前本地跑通；PR 只创建不合并，由主窗口 review 后合并。

## 1. 主提示词（Master Prompt，每窗口必贴）

```text
你是面个蛋（MianGeDan，AI 数字面试官产品）的前端重构实现者。仓库在
C:\Users\Administrator\Documents\Codex\2026-08-02\3135804887-ops-miangedan-https-github-com\work\miangedan
（git 分支基于 main 创建，禁止动其他目录）。

开工前必须完整阅读：
1) docs/design/FRONTEND-REFACTOR-BRIEF.md —— 全部 23 个页面清单、设计风格参考、硬约束。
2) docs/design/DESIGN-SYSTEM.md 与 docs/design/ACCESSIBILITY.md —— 令牌体系与无障碍基线。
3) apps/web 现有页面与 src/mocks、src/lib/api-fetch.ts、packages/i18n 的既有用法。

技术基线（不得破坏）：
- Next.js App Router + React + TypeScript strict + pnpm workspace + Tailwind 风格 className。
- 设计令牌 @mgd/design-tokens（--mgd-app-brand-ink/from/to、--mgd-app-surface-muted 等），
  视觉语言为「深色品牌渐变 + 玻璃拟态 + 霓虹点缀」。
- UI 组件统一复用 @mgd/ui；不引入新 UI 库；新增基础组件放 packages/ui 并补测试。
- 文案走 packages/i18n（zh-CN/en-US 双语成对），数据走 apiFetch + src/mocks（标注 synthetic）。

硬约束：
- 不改路由结构与 SCR 映射；不删除既有 i18n keys；保留房间页的播放问候语/转写演示/
  打断演示/字幕开关/暂停/退出/降级覆盖层等功能。
- 不动后端、AI 服务、评测与 infra 目录；只允许改动 apps/web、packages/ui、
  packages/design-tokens、packages/i18n（按需）。
- 动效 150~250ms ease-out，尊重 prefers-reduced-motion；满足 WCAG 2.2 AA（axe 门禁）。
- 提交前必须全部通过：pnpm lint / pnpm typecheck / pnpm test / pnpm i18n:check / pnpm api:check。

工作协议：
- 每个批次只做指定页面；先读该批提示词。
- 分支命名 feat/fe-refactor-{batch}；PR 标题含批次名；PR 只创建不合并。
- 每批交付附「完成报告」：改动文件清单、关键视觉/交互决策、门禁结果、遗留问题。
- 全程用中文回复；遇到与约束冲突的选择，先说明再做最小改动。
```

## 2. 批次提示词

### 批次 0：风格稿（必做，先定方向）

```text
批次 0：视觉风格稿。实现并完善三页：/（落地页，SCR-01）、/sessions/[id]
（实时面试房间，SCR-08/09）、/projects/[id]/report（项目报告，SCR-11）。

要求：
- 落地页参考 OpenAI/Claude/DeepL 的 AI 产品高级感：深色渐变 Hero + 数字面试官形象 +
  单一发光 CTA + 三步流程 + 能力与合规背书。
- 房间参考 Zoom/HireVue：顶部状态栏（状态/计时/网络/字幕/退出）、左侧问题+转写+工具、
  右侧数字人视频区（保留播放问候语/转写演示/打断演示按钮）、降级覆盖层。
- 报告参考 TestGorilla/Stripe：总体结论徽章 + 六维评分条/雷达图 + 证据时间线。
- 输出三页截图或视觉说明：色彩、布局、动效决策。
验收：三页门禁全绿；保留所有既有交互功能；无 mock 回归。
```

### 批次 1：公开区（SCR-01/02）

```text
批次 1：公开区。页面：/（落地页）、/demo（演示页）、/auth（登录注册）。
视觉：延续批次 0 风格稿的落地页方向；demo 页突出「真实面试房间」预览；auth 页简洁居中卡片。
验收：双语文案成对；门禁全绿；不破坏批次 0 页面。
```

### 批次 2：工作台（SCR-03/04/05/06）

```text
批次 2：工作台。页面：/dashboard、/projects/new、/projects/[id]/review、/projects/[id]/plan。
视觉：参考 HireVue/TestGorilla/飞书招聘；卡片式项目列表、状态徽章、进度条；
复核页用证据时间线 + 评分修正；计划页用轮次时间线。
验收：新建项目多步表单可走通（mock）；门禁全绿。
```

### 批次 3：面试房间深化（SCR-08/09，核心）

```text
批次 3：实时面试房间深化。/sessions/[id]。
重点：顶部状态栏（状态/计时/网络/字幕开关/退出）、左侧问题/候选人转写/修订、
右侧数字人视频区（静态形象 + 音频）、工具面板、暂停/重连/降级/认证覆盖层。
必须保留：播放问候语、转写演示、打断演示三个按钮及其逻辑；aria-live 字幕；键盘可达。
验收：房间页 3 个既有单测与 axe 页面扫描通过；门禁全绿。
```

### 批次 4：结果与报告（SCR-10/11/12）

```text
批次 4：结果与报告。页面：/projects/[id]/rounds/[n]/result、/projects/[id]/report、
/projects/[id]/practice/[pid]。
视觉：评分条/雷达图/证据时间线（批次 0 报告风格稿延伸）；练习页简洁带进度感。
验收：门禁全绿；报告导出入口保留。
```

### 批次 5：账户（SCR-13/14/15）

```text
批次 5：账户。页面：/library、/settings、/billing。
视觉：参考 Linear/Stripe 的后台克制风；题库页筛选/标签；设置页分区（资料/隐私/导出/通知/语言）；
账单页套餐与用量卡片。
验收：门禁全绿；设置页隐私与删除导出入口保留。
```

### 批次 6：机构与管理（SCR-16/17）

```text
批次 6：机构与管理。/org/[orgId]/*（aggregates/assignments/assignment detail/completion/
members/permissions/shares）与 /admin。
视觉：参考 Linear/Stripe Dashboard；左侧导航 + 紧凑表格 + 权限矩阵 + feature flags 开关。
验收：门禁全绿；admin 的 feature flags 状态展示与审计提示保留。
```

## 3. 融合协议（主窗口执行）

- 每个批次 PR 合并前，主窗口跑一遍门禁并抽查：SCR 映射、保留功能、无障碍、i18n、mock 契约。
- 冲突或回归：退回该批修复；通过后 squash 合并到 main。
- 每批完成后更新本文「批次状态」与 IMPLEMENTATION_PLAN 对应 SCR 状态。

## 4. 批次状态跟踪

- 批次 0 风格稿：待开始
- 批次 1 公开区：待开始
- 批次 2 工作台：待开始
- 批次 3 面试房间：待开始
- 批次 4 结果与报告：待开始
- 批次 5 账户：待开始
- 批次 6 机构与管理：待开始

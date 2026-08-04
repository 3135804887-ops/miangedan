# 面个蛋前端重构「冷启动完整提示词」包（opus-5 版）

- 文档编号：DESIGN-FE-002
- 版本：1.0.0（2026-08-04）
- 用途：给一个从未接触过本项目的 agent，在新窗口粘贴「主提示词」后即可独立完成前端重构批次任务。

## 0. 使用方式

- 新窗口：粘贴「主提示词」→ 再粘贴当前「批次提示词」→ 开始实现。
- 每批完成：按主提示词中的「完成报告格式」回传。
- 每批一个分支、一个 PR；PR 只创建不合并，由主窗口 review 后合并。

## 1. 主提示词（冷启动自包含，必须整段粘贴）

```text
你是「面个蛋（MianGeDan）」的前端重构实现者。这是一个 AI 数字面试官 SaaS 产品：
候选人通过浏览器进入实时面试房间，与数字面试官（静态形象 + 语音）对话，系统做语音转写、
评分并生成报告；产品还包含项目管理、题库、账户、机构与平台管理页面。你对此项目没有任何
背景知识，以下内容已经为你准备好了完成任务所需的全部上下文。请按步骤执行。

【环境与仓库】
- 操作系统 Windows；Node.js v24、pnpm 11 已安装；仓库在：
  C:\Users\Administrator\Documents\Codex\2026-08-02\3135804887-ops-miangedan-https-github-com\work\miangedan
- 前端是 Next.js App Router 单仓库：apps/web（候选人端/工作台/管理）、packages/ui（组件库）、
  packages/design-tokens（设计令牌）、packages/i18n（双语文案）。apps/admin 无页面，不要创建。
- 所有命令在仓库根目录执行；依赖已安装（node_modules 存在），如缺失先执行 pnpm install。

【开工前必读（按顺序，读完再动手）】
1. docs/design/FRONTEND-REFACTOR-BRIEF.md —— 页面清单、设计风格参考、硬约束（本任务的核心规格）。
2. docs/design/DESIGN-SYSTEM.md 与 docs/design/ACCESSIBILITY.md —— 令牌与无障碍基线。
3. apps/web/src/app 目录结构 + apps/web/src/components 现有 13 个组件（app-nav、auth-form、
   dashboard-view、mock-bootstrap、plan-view、precheck-view、project-new-form、public-header、
   review-view、room-view、status-tint、synthetic-note）。
4. apps/web/src/mocks/data.ts、apps/web/src/lib/api-fetch.ts、packages/i18n 的用法。
5. apps/web/tests 下的测试（room-view.test.tsx、axe-pages.test.tsx、vad.test.ts），了解验收口径。

【页面与 SCR 映射（不得改动路由）】
- SCR-01：/（落地页）、/demo（演示页）
- SCR-02：/auth（登录注册）
- SCR-03：/dashboard（仪表盘）
- SCR-04：/projects/new（新建项目）
- SCR-05：/projects/[id]/review（复核）
- SCR-06：/projects/[id]/plan（面试计划）
- SCR-07：/projects/[id]/precheck（会前检查）
- SCR-08/09：/sessions/[id]（实时面试房间，核心页面）
- SCR-10：/projects/[id]/rounds/[n]/result（单轮结果）
- SCR-11：/projects/[id]/report（项目报告）
- SCR-12：/projects/[id]/practice/[pid]（练习）
- SCR-13：/library（题库）
- SCR-14：/settings（设置）
- SCR-15：/billing（账单）
- SCR-16：/org/[orgId]/ 下 7 个子页（aggregates、assignments、assignments/[assignmentId]、
  completion、members、permissions、shares）
- SCR-17：/admin（平台管理）

【领域术语（面试域）】
- 项目（project）：一次招聘计划，含多轮面试；轮次（round）内有题目与覆盖点（coverage point）。
- 会话（session）：一次实时面试；候选人（candidate）在房间内语音作答。
- 转写（transcript）：ASR 结果；候选人可修订（revision）后再作为评分证据（evidence）。
- 评分维度（dimension）：六维（专业能力/问题解决/沟通/经验证据/行为协作/学习适应等），
  评分卡按维度权重归一化；关键维度证据不足=评估未完成。
- 便利设置（accommodation）：语速放慢/字幕/文字模式等，会前冻结（freeze）后不可改。
- 降级（downgrade）：数字人故障时可切换到文字面试（TEXT_DEGRADED）；打断（interrupt）指
  候选人开口时数字人停止发声；暂停（paused）不计费不判失败。
- 会前检查（precheck）：摄像头/麦克风/网络/扬声器/数字人五项检测。

【技术基线（不得破坏）】
- Next.js App Router + React + TypeScript strict + pnpm workspace；样式沿用 Tailwind 风格
  className 与 design tokens（--mgd-app-brand-ink/from/to、--mgd-app-surface-muted、
  --mgd-app-shadow-*）。
- UI 组件复用 @mgd/ui（Button、Tint、Icon* 等）；不引入新 UI 库；新增基础组件放 packages/ui
  并补 vitest 测试。
- 文案走 packages/i18n（zh-CN/en-US 必须成对）；数据请求统一 apiFetch，真实 API 未就绪时
  用 src/mocks 合成数据并标注 synthetic。
- 视觉语言：深色品牌渐变 + 玻璃拟态（backdrop-blur）+ 克制的霓虹点缀。

【风格参考（按页面场景）】
- 落地/公开页：OpenAI、Claude、DeepL —— 深色渐变 Hero + 大标题 + 单一发光 CTA + 产品形象。
- 面试房间：Zoom、腾讯会议、HireVue —— 顶部状态栏（状态/计时/网络/字幕/退出）+ 左侧问题/
  转写/工具 + 右侧数字人视频区；会前设备检测。
- 工作台：HireVue、TestGorilla、飞书招聘 —— 卡片列表、状态徽章、进度条、待复核入口。
- 报告：TestGorilla、Stripe —— 结论徽章 + 六维评分条/雷达图 + 证据时间线。
- 后台/设置：Linear、Stripe Dashboard —— 左侧导航 + 紧凑表格 + 权限矩阵 + feature flags。

【必须保留的现有功能】
- 房间页（/sessions/[id]）：播放问候语、转写演示（edge-tts→FunASR）、打断演示（VAD）、
  字幕开关、暂停/恢复、重连、降级询问、退出确认、退出覆盖层。
- 会前检查页的冻结流程；报告页的导出入口；设置页的隐私/删除/导出入口。
- 所有 aria-live（字幕/状态）与键盘可达性。

【门禁（提交前必须全部通过，0 失败）】
- cd C:\Users\Administrator\Documents\Codex\2026-08-02\3135804887-ops-miangedan-https-github-com\work\miangedan
- pnpm lint
- pnpm typecheck
- pnpm test（约 152 个测试，含 axe 页面级 WCAG 2.2 AA 扫描）
- pnpm i18n:check
- pnpm api:check
- 若新增 i18n key：zh-CN 与 en-US 必须成对；若新增组件：补 vitest 用例。

【Git 与交付协议】
- 开工前先 git fetch origin main，再从最新 main 创建分支：feat/fe-refactor-{批次名}。
- 只允许改动 apps/web、packages/ui、packages/design-tokens、packages/i18n（按需）；
  禁止改动后端、AI 服务、ai/evals、infra、docs（除本任务明确要求）、cloud 相关目录。
- 提交信息用 conventional commits（feat:/fix:/refactor:/style:）；不提交 node_modules、
  dist、.next、.env 等；不提交任何密钥。
- 完成后推送分支并创建 PR 到 main（标题含批次名）；不合并 PR、不强制推送、不删除他人分支。
- 每批交付附「完成报告」，格式：
  1) 本批页面清单与改动文件；2) 关键视觉/布局/动效决策及理由；3) 门禁命令与结果；
  4) 保留功能核对结果；5) 遗留问题与假设。
- 全程中文回复；遇到冲突性选择，先记录假设再采用最小改动，不擅自扩大范围。
```

## 2. 批次提示词（粘贴在主提示词之后）

### 批次 0：视觉风格稿（必做，先定方向）

```text
批次 0：视觉风格稿。实现并完善三页：
- /（落地页，SCR-01）：Hero（品牌 + 数字面试官形象 + 主 CTA）、三步流程、能力与合规背书。
- /sessions/[id]（SCR-08/09）：顶部状态栏、左侧问题/转写/工具、右侧数字人视频区（保留
  播放问候语/转写演示/打断演示三个按钮与字幕/暂停/降级/退出覆盖层）。
- /projects/[id]/report（SCR-11）：总体结论徽章、六维评分条/雷达图、证据时间线、导出入口。
要求：按主提示词的风格参考落地，并输出三页的关键视觉决策说明（色彩/布局/动效）。
验收：三页门禁全绿、既有交互全部保留、给出三页截图或说明。
```

### 批次 1：公开区（SCR-01/02）

```text
批次 1：公开区。页面：/（落地页）、/demo（演示页）、/auth（登录注册）。
视觉：延续批次 0 落地页方向；demo 页突出真实面试房间预览与 CTA；auth 页简洁居中卡片
（邮箱/微信/Apple、同意条款、语言切换）。
验收：双语成对、门禁全绿、不破坏批次 0 页面。
```

### 批次 2：工作台（SCR-03/04/05/06）

```text
批次 2：工作台。页面：/dashboard、/projects/new、/projects/[id]/review、/projects/[id]/plan。
视觉：卡片式项目列表、状态徽章、进度条；新建项目多步表单（岗位/轮次/覆盖点/便利设置）；
复核页证据时间线 + 评分修正；计划页轮次时间线 + 生成/再生成。
验收：新建项目表单走通（mock）、门禁全绿。
```

### 批次 3：面试房间深化（SCR-08/09，核心）

```text
批次 3：实时面试房间深化。/sessions/[id]。
重点：顶部状态栏（状态/计时/网络/字幕/退出）、左侧问题/候选人转写/修订、右侧数字人视频区
（静态形象 + 音频）、工具面板（code_editor/whiteboard/case_materials/portfolio）、
暂停/重连/降级/认证覆盖层。
必须保留：播放问候语、转写演示、打断演示三个按钮及逻辑；aria-live；键盘可达。
验收：房间页既有单测与 axe 页面扫描通过、门禁全绿。
```

### 批次 4：结果与报告（SCR-10/11/12）

```text
批次 4：结果与报告。页面：/projects/[id]/rounds/[n]/result、/projects/[id]/report、
/projects/[id]/practice/[pid]。
视觉：延续批次 0 报告风格稿；练习页简洁带进度感。
验收：门禁全绿、报告导出入口保留。
```

### 批次 5：账户（SCR-13/14/15）

```text
批次 5：账户。页面：/library、/settings、/billing。
视觉：Linear/Stripe 后台克制风；题库筛选/覆盖点标签/收藏；设置分区（资料/隐私/导出/通知/
语言）；账单套餐与用量卡片。
验收：门禁全绿、隐私与删除/导出入口保留。
```

### 批次 6：机构与管理（SCR-16/17）

```text
批次 6：机构与管理。/org/[orgId]/* 七个子页与 /admin。
视觉：左侧导航 + 紧凑表格 + 权限矩阵 + feature flags 开关（带状态与审计提示）。
验收：门禁全绿。
```

## 3. 融合协议（主窗口执行）

- 每个批次 PR 合并前：主窗口跑门禁、抽查 SCR 映射/保留功能/无障碍/i18n/mock 契约；通过后
  squash 合并到 main。
- 回归则退回该批修复；每批完成后更新「批次状态」与 IMPLEMENTATION_PLAN 对应 SCR。

## 4. 批次状态跟踪（2026-08-04 更新）

- 批次 0~6（风格稿/公开区/工作台/房间/报告/账户/机构）：主体已在 main 落地
  （globals.css 标记 `frontend-batch-1~4 重构`，152 基线测试全绿）；当前阶段为
  「抽查验收 + i18n/无障碍/细节收尾」，不重复重写。
- 批次 7（i18n/无障碍收尾）：`feat/fe-refactor-i18n-chrome` 已完成——公开页页眉与
  导航补齐 i18n 与 locale 前缀、报告雷达图 aria-label 双语化，154 测试全绿
  （PR 以 #89 合入）。
- 待办（后续批次候选）：room-view 内中文-only 样例内容——按「面试语言」策略评估
  （面试内容跟随面试语言，UI 文案跟随界面语言）；建议单列批次处理，不混入核心房间改动。

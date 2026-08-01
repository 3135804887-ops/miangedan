# 前端工作区（frontend-global-pages）现状与续接指南

| 字段 | 内容 |
|---|---|
| 状态 | **批次 0 开发中断**（kiro 未继续）：代码已写、未提交、未完成安装与 CI 接入 |
| 规格 | `docs/frontend/spec/`（自 `.kiro/specs/frontend-global-pages/` 迁入并纳入版本控制；requirements.md / design.md / tasks.md） |
| 追踪 | PRD US-01~08；SCREEN-SPEC SCR-01~17；FR-001~040；前端批次见 `spec/tasks.md` |

## 一、审读结论（2026-08-01）

批次 0（工作区与 `apps/web` 脚手架）的**代码主体已写出但未完成**：

- 已写：根 pnpm 工作区（`package.json`/`pnpm-workspace.yaml`/`tsconfig.base.json`/`eslint.config.mjs`/
  `vitest.config.ts`/`.npmrc`）；`apps/web` Next.js App Router 骨架（`[locale]` 路由、middleware、
  `(app)` 布局与约 25 个页面路由、room 布局、i18n 三件、lib 四件、msw mock）；`packages/`
  （design-tokens 令牌+生成器、domain-states 状态枚举+契约断言、eslint-plugin-mgd 状态字面量禁令、
  i18n 运行时+文案、ui 基础层）；前端工具脚本 `tools/*.mjs`。
- 缺口：
  1. **无 `pnpm-lock.yaml`**，且 `node_modules` 仅剩 `.pnpm`（半装状态，顶层包与 `.bin` 缺失）——
     `pnpm run` 因此挂起，**当前不可构建/不可测试**。
  2. `contracts/ts` 只有 `package.json`，未运行 `pnpm api:generate` 生成 `openapi.d.ts`。
  3. `tools/validate_docs.py` 仅加了 node_modules 跳过（jsonl/fences），批次 0 的 CI 接入（FE-0.10）未完成。
  4. `spec/tasks.md` 复选框全部未勾选（进度未回写）。
  5. `apps/admin` 仅 README（批次 4 待办）；axe 无障碍、`scan:bundle` 等门禁未验证。
- 红线核对：实现为静态壳 + msw mock（合成数据），未接真实后端；未发现真实 PII/密钥入库。

## 二、文件地图

```text
docs/frontend/spec/        # 需求/设计/任务三件（批次 0~4 的唯一规格源）
apps/web/                  # 求职者端 Next.js（SCR-01~16）
apps/admin/                # 运营后台（SCR-17，批次 4）
packages/design-tokens/    # 设计令牌（JSON → CSS/TS 生成，含对比度检查）
packages/domain-states/    # 项目/会话/便利设置状态枚举 + 契约断言
packages/eslint-plugin-mgd/# 禁止状态字面量（防状态机漂移）
packages/i18n/             # zh-CN/en-US 运行时与文案
packages/ui/               # 基础组件层（primitive/pattern/a11y）
contracts/ts/              # openapi.yaml 生成的只读类型（未生成，待补）
tools/*.mjs                # api:generate/stamp/check、bundle 密钥扫描
```

## 三、续接步骤（恢复批次 0）

1. 修复安装：删除半装 `node_modules`，`pnpm install` 生成并**提交 `pnpm-lock.yaml`**。
2. `pnpm api:generate` 生成 `contracts/ts/openapi.d.ts` 并提交。
3. 跑 `pnpm lint` / `pnpm typecheck` / `pnpm test` / `pnpm build` / `pnpm scan:bundle` 全绿。
4. 完成 FE-0.10：前端 lint/typecheck/test/build 接入 CI 阶段 2/6（ci.yml 预留挂接点）。
5. 回写 `spec/tasks.md` 勾选进度；按批次 0 合入任务提交 PR（分支
   `task/frontend-batch-0-web-scaffold`）。
6. 批次 1~4 按 `spec/tasks.md` 顺序推进（SCR-01~17）。

## 四、红线

- 静态壳 + mock（合成数据），真实联调在对应后端服务合入后替换；契约文件只读。
- 禁止真实简历/手机号/邮箱/证件/音视频入库；`scan:bundle` 保证产物零密钥。
- 页面状态一律使用 `packages/domain-states` 枚举（eslint 禁令），不得自创状态名。

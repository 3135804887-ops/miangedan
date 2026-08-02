# 前端工作区（frontend-global-pages）现状与续接指南

| 字段 | 内容 |
|---|---|
| 状态 | **批次 0 收尾中**：安装、契约生成、共享包、CI 与 A+B 全局视觉基线已完成；全仓复验与 PR 待闭环 |
| 规格 | `docs/frontend/spec/`（自 `.kiro/specs/frontend-global-pages/` 迁入并纳入版本控制；requirements.md / design.md / tasks.md） |
| 追踪 | PRD US-01~08；SCREEN-SPEC SCR-01~17；FR-001~040；前端批次见 `spec/tasks.md` |

## 一、续接进度（2026-08-02）

批次 0（工作区与 `apps/web` 脚手架）已从中断点恢复，当前事实如下：

- 已在 `task/frontend-batch-0-web-scaffold` 合并最新 `main`；OpenAPI、PRD、默认流程与已接受 ADR
  相对中断点无规格漂移，后端 TASK-020/021/022/030 的最新实现仅作为后续批次契约背景。
- 已生成 pnpm 11.18.0 单一锁文件；`openapi-typescript@7.13.0` 与 TypeScript 5.9.3 的 peer 契约
  已校验，`pnpm install --frozen-lockfile` 与 `pnpm peers check` 均通过。
- `contracts/ts/openapi.d.ts` 与 `SOURCE.md` 已由只读 OpenAPI 契约生成，`api:check` 校验零漂移。
- UI 缺失原语、axe/交互/状态/隐私/幂等测试、i18n 源码键解析与打包扫描已补齐；当前
  17 个测试文件、91 个测试通过。
- CI 保持原 job 与 `needs` 链不变，只在阶段 2/3/6 挂接前端静态检查、测试、构建与产物扫描。
- 全部前端门禁、文档 17 套校验与 20 个 Go 模块构建已通过；FE-0.1 ~ FE-0.10 已据实回写。
- 已按 `claude-design` 三向对比收敛为 A+B 视觉基线：纯文字品牌、编辑网格与工具型状态反馈；
  1280px / 388px 浏览器检查无横向溢出，移动端最小可见交互目标 56px，控制台零警告/错误。
- 待办仅剩：复跑全门禁并经 PR/CI 合入（FE-0.11）。
- 红线核对：仍为静态壳 + MSW 合成 mock；未接真实后端或媒体供应商，未写入真实 PII、密钥或媒体。

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
contracts/ts/              # openapi.yaml 生成的只读类型 + SOURCE 来源标记
tools/*.mjs                # api:generate/stamp/check、bundle 密钥扫描
```

## 三、剩余步骤（完成批次 0）

1. 复跑全门禁，同步 `CHANGELOG.md` 与 `IMPLEMENTATION_PLAN.md`。
2. 推送分支、创建批次 0 PR，等待必需检查全绿后 squash 合入 `main`。
3. 从最新 `main` 依次推进批次 1 ~ 4（SCR-01 ~ 17）。

## 四、红线

- 静态壳 + mock（合成数据），真实联调在对应后端服务合入后替换；契约文件只读。
- 禁止真实简历/手机号/邮箱/证件/音视频入库；`scan:bundle` 保证产物零密钥。
- 页面状态一律使用 `packages/domain-states` 枚举（eslint 禁令），不得自创状态名。

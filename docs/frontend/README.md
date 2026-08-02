# 前端工作区（frontend-global-pages）现状与续接指南

| 字段 | 内容 |
|---|---|
| 状态 | **批次 0、1 已完成**：SCR-01 ~ SCR-07 核心页面已闭环；下一步为批次 2 房间外壳 |
| 规格 | `docs/frontend/spec/`（自 `.kiro/specs/frontend-global-pages/` 迁入并纳入版本控制；requirements.md / design.md / tasks.md） |
| 追踪 | PRD US-01~08；SCREEN-SPEC SCR-01~17；FR-001~040；前端批次见 `spec/tasks.md` |

## 一、批次 0 完成记录

- 根工作区使用 Node `>=20.9`、pnpm `11.18.0` 与唯一 `pnpm-lock.yaml`；依赖均为精确版本。
- `contracts/ts/openapi.d.ts` 已由只读 OpenAPI 契约生成，`SOURCE.md` 记录源提交与内容散列；
  `api:check` 在 CI 校验生成物零漂移。
- `apps/web` 已提供双语语言前缀、SCR-01 ~ SCR-16 路由壳、全局错误/404/加载边界；
  `packages/design-tokens`、`domain-states`、`eslint-plugin-mgd`、`i18n`、`ui` 均有契约与测试。
- CI 阶段 2 执行 lint/typecheck/i18n/令牌/API 检查，阶段 3 执行 Vitest + axe + 隐私/幂等测试，
  阶段 6 执行生产构建与 bundle 密钥扫描，原 1→2→3→4→5→6 依赖链不变。
- `openapi-typescript@7.13.0` 的 peer 契约为 TypeScript `^5.x`，因此工具链锁定 TypeScript 5.9.3；
  `strict: true` 基线不变，类型生成与应用类型检查共用同一受支持编译器。
- 红线核对：当前仍为静态壳 + MSW 合成 mock；未接真实后端/媒体供应商，未引入真实 PII、密钥或媒体。

## 二、批次 1 完成记录

- SCR-01 ~ SCR-07 已提供双语响应式实现，并统一支持正常、空、加载、错误、权限不足与恢复视图。
- 工作台严格使用 15 项项目状态枚举并将五类筛选写入 URL；解析页仅展示被排除的敏感字段类别名；
  计划页的统一评分、60 分门槛、解锁和交接规则保持只读，不展示正式问题或答案。
- 九项便利设置默认关闭且独立；会前五项检查支持单项幂等重试，摄像头/麦克风关闭继续不影响评分，
  额度不足在开始前阻止并给出购买路径。
- 浏览器复核覆盖 1440px 桌面与 390px 移动视口，无水平溢出；中英文 SCR-01 ~ SCR-07 的
  axe serious/critical 违规为 0，异常/保留/幂等路径进入 Vitest。

## 三、文件地图

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

## 四、续接步骤

1. 从最新 `main` 创建 `task/frontend-batch-2-room-shell`。
2. 按 `spec/tasks.md` 完成 FE-2.1 ~ FE-2.4（SCR-08 / SCR-09），保持既有门禁全绿。
3. 房间只实现布局、媒体桩和覆盖层；不得使用静态头像或预录内容冒充实时数字人/ASR。
4. 后续依次推进批次 3、4；不得在页面层自创状态、接真实媒体或放宽无障碍门槛。

## 五、红线

- 静态壳 + mock（合成数据），真实联调在对应后端服务合入后替换；契约文件只读。
- 禁止真实简历/手机号/邮箱/证件/音视频入库；`scan:bundle` 保证产物零密钥。
- 页面状态一律使用 `packages/domain-states` 枚举（eslint 禁令），不得自创状态名。

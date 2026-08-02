# 任务清单：全局前端页面（frontend-global-pages）

| 字段 | 内容 |
|---|---|
| 特性名 | frontend-global-pages |
| 需求文档 | `.kiro/specs/frontend-global-pages/requirements.md` |
| 设计文档 | `.kiro/specs/frontend-global-pages/design.md` |
| 实现语言 | TypeScript（Next.js App Router + React，`strict: true`） |
| 批次数 | 5（frontend-batch-0 ~ frontend-batch-4） |

任务编号规则：`FE-{批次}.{序号}`。每个任务标注需求追踪（requirements.md 的 G*/B* 编号）与设计锚点（design.md 章节号）。每个批次末尾是该批次的合入任务（分支、PR、CHANGELOG、追踪同步）。

---

## 批次 0：工作区与 `apps/web` 脚手架

分支 `task/frontend-batch-0-web-scaffold`。出口：`pnpm install` 后 lint、typecheck、test、build 全绿；语言前缀路由壳与全局错误页可访问；CI 三处接入生效。

- [x] FE-0.1 建立 pnpm 工作区骨架
  - 根 `package.json`（`private: true`，脚本 `lint`/`typecheck`/`test`/`build`/`api:generate`/`api:check`/`i18n:check`/`tokens:check-names`/`tokens:check-contrast`/`scan:bundle`）、`pnpm-workspace.yaml`（`apps/*`、`packages/*`、`contracts/ts`）、`tsconfig.base.json`（strict 基线与路径别名）
  - 全部依赖写精确版本；生成并提交 `pnpm-lock.yaml`
  - 需求：B0-1（1、3、4、7） | 设计：2.1

- [x] FE-0.2 生成 `contracts/ts` 只读类型包
  - `openapi-typescript` 生成 `contracts/ts/openapi.d.ts`；`contracts/ts/package.json` 暴露为 `@mgd/api-types`；`tools/stamp-contract-source.mjs` 写 `contracts/ts/SOURCE.md`（记录 `docs/api/openapi.yaml` 的 commit hash）
  - `pnpm api:check` 实现重新生成后 diff 校验与失败文案
  - 需求：G6（1、2） | 设计：8.1

- [x] FE-0.3 实现 `packages/design-tokens`
  - `tokens/{color,typography,space,breakpoint}.json` 采用 DESIGN-SYSTEM 第 5 ~ 7 节中性占位值；`src/build-css.ts` 输出 `dist/tokens.css`；`tailwind-preset.ts` 全量指向 CSS 变量
  - `tokens/NAMES.lock` 与 `pnpm tokens:check-names`；`tokens/contrast-pairs.json` 与 `pnpm tokens:check-contrast`
  - 单元测试：对比度算法（已知色对期望值）、名称集合 diff 失败路径
  - 需求：G5（1、2、3、9）、G4（4） | 设计：3

- [x] FE-0.4 实现 `packages/domain-states` 与契约断言
  - `PROJECT_STATUSES`（15）、`SESSION_STATUSES`（10）、`ACCOMMODATION_KEYS`（9）、`org.ts` 的机构可见性投影常量
  - `assert-contract.ts` 编译期类型等价断言（`components['schemas']['ProjectStatus']`、`components['schemas']['Session']['room_status']`）
  - 测试：读 `ai/schemas/turn-evidence.schema.json` 断言便利设置集合；解析 `docs/domain/INTERVIEW-STATE-MACHINE.md` 5.1/6.2 表格断言两组状态集合
  - 需求：G1（1、3） | 设计：5.1 ~ 5.3

- [x] FE-0.5 实现状态名字面量禁令
  - `packages/eslint-plugin-mgd` 的 `no-domain-state-literal` 规则（含白名单与报错文案）
  - `tests/no-invented-states.test.ts` 源码扫描兜底，失败输出 `文件:行 命中值`
  - 规则自身的正例/反例单测
  - 需求：G1（3） | 设计：5.4

- [x] FE-0.6 实现 `packages/i18n`
  - `config.ts`（`SUPPORTED_LOCALES`、`DEFAULT_LOCALE`、`FALLBACK_LOCALE`）、`request.ts`、`format.ts`（`formatMoney` 处理最小货币单位）、`nfr-expectations.ts`
  - `messages/{zh-CN,en-US}/{common,error}.json`；`common.redline.*` 四条固定文案双语
  - `pnpm i18n:check`：键对称差、源码键存在性、ICU 占位符一致性、两语言同值检测
  - 需求：G3（1、3、4、5、6、8）、G10（1 ~ 4） | 设计：7

- [x] FE-0.7 实现 `packages/ui` 基础层
  - primitives：`Button`/`IconButton`/`Switch`/`Field`/`Skeleton`/`AlertDialog`；patterns：`StateView`/`ErrorPanel`/`StatusBadge`/`DisclosureNote`
  - `state-contract.ts` 七态 props 契约；`a11y/focus-trap.ts`、`a11y/target-size.css`；`testing/control-registry.ts`（`data-mgd-control`）
  - 组件测试：每个交互组件七态断言（含 `disabled` 不可聚焦且有原因、`loading` 的 `aria-busy`、`error` 的图标+文字、焦点环不可移除）
  - 需求：G5（4 ~ 8）、G4（1、2、3、6）、G1（4、5） | 设计：4

- [x] FE-0.8 搭建 `apps/web` 应用壳
  - `next.config.ts`、`tsconfig.json`、`eslint.config.mjs`（含 import zone 与硬编码色值/字号禁令）、`vitest.config.ts`、`tailwind.config.ts`（`presets: [mgdPreset]`）
  - `proxy.ts`（Next.js 16 约定）语言前缀重定向与不支持 locale 回退；根布局输出 `<html lang>`、skip-link、导航与页脚
  - `app/[locale]/` 下 SCR-01 ~ SCR-16 路由壳（`(app)` 与 `(room)` 两个路由组）
  - `app/{error,not-found,loading}.tsx` 全局边界
  - 需求：B0-1（2、4、5、6）、B0-2（1 ~ 6）、G3（1、2、7）、G7（1、5） | 设计：2.1、2.3、16

- [x] FE-0.9 搭建数据层与安全封装
  - `src/lib/api-fetch.ts`（写操作幂等键类型约束、`Error` 信封解析）、`src/lib/error-presenter.ts`（21 项 `ErrorCode` → 五要素映射）
  - `src/lib/telemetry.ts` 白名单出口 + ESLint 禁止直接 `console.error` 上报
  - `src/lib/region-context.ts` 只读数据区
  - `src/mocks/{browser,server}.ts` 与 `handlers/index.ts` 骨架；`vitest.setup.ts` 启用 MSW
  - 测试：`presentError` 逐码覆盖、遥测脱敏、`NEXT_PUBLIC_*` 键白名单
  - 需求：G6（3、4、5）、G2（4）、G8（1 ~ 4） | 设计：6.2、8.2、8.3、11

- [x] FE-0.10 接入 CI 并加固文档校验脚本
  - `.github/actions/setup-frontend/action.yml`（node 22 + pnpm 11.18.0 + 缓存 + `--frozen-lockfile`）
  - `ci.yml` 三处追加：`stage2-lint`（lint、typecheck、i18n:check、tokens:check-names、tokens:check-contrast、api:check）、`stage3-unit-tests`（`pnpm test --run`）、`stage6-build`（`pnpm build`、`scan:bundle`）；不新增 job、不改 `needs`
  - `tools/validate_docs.py` 最小加固：`check_fences` 与 JSONL 分支补 `node_modules` 排除（与既有套件一致）
  - 需求：B0-3（1 ~ 5）、G11（3、4） | 设计：13、12

- [ ] FE-0.11 批次 0 合入
  - 先合并最新 `main`；提交标题 `feat(web-batch-0): 建立 Web 工作区、设计令牌与全局壳（SCR-01~17, FR-028）`
  - PR 正文：页面→FR 映射、偏离 1（语言前缀路由）与偏离 2（Storybook 等价物）披露、`validate_docs.py` 加固说明
  - `CHANGELOG.md` 的 `[Unreleased] / Added` 记录；`IMPLEMENTATION_PLAN.md` EPIC-02 补批次追踪行
  - 本地验证：`python tools/validate_docs.py` 全绿 + 逐 Go 模块 `go build ./...`
  - 需求：G11（5 ~ 8） | 设计：14

---

## 批次 1：核心页面 SCR-01 ~ SCR-07

分支 `task/frontend-batch-1-core-pages`。出口：七个页面组五态齐备、双语文案、axe serious+critical = 0。

- [ ] FE-1.1 SCR-01 落地页与样例演示
  - `/{locale}` 与 `/{locale}/demo`；样例数据标识；未登录上传入口跳转 `auth` 并回跳；演示资源失败的五要素文案；16 岁使用范围说明
  - 测试：正常渲染、演示加载失败、刷新回默认态、文案不含录用预测/投递承诺
  - 需求：B1-1（1 ~ 7） | 设计：2.3、6

- [ ] FE-1.2 SCR-02 登录/注册
  - 四种登录方式、注册协议三项（录制/机构共享/模型训练不在注册捆绑）、验证码有效期与重发冷却、年龄声明、登录后回跳上下文
  - 测试：验证码错误保留邮箱、第三方失败回落邮箱路径、无手机号必填字段
  - 需求：B1-2（1 ~ 8） | 设计：6、8.3

- [ ] FE-1.3 SCR-03 工作台
  - 项目卡片（岗位/轮次/状态徽标/下一动作）、五类筛选写入 URL、骨架屏、空态与筛选空态、其他设备"活动中"标识与安全转移入口
  - 测试：15 项状态徽标渲染、筛选 URL 恢复、列表失败五要素（数据保留、不影响评分与额度）
  - 需求：B1-3（1 ~ 8）、G1（2、4） | 设计：5、6

- [ ] FE-1.4 SCR-04 创建面试项目
  - 双栏输入、上传限制（PDF/DOC/DOCX ≤10MB）、删除替换、样例材料一键填充、草稿恢复、未登录权限不足态
  - 测试：文件被拒时 JD 文本不丢失、请求失败保留原始输入
  - 需求：B1-4（1 ~ 8） | 设计：6

- [ ] FE-1.5 SCR-05 解析校对页
  - `PARSING`/`MATERIAL_REVIEW`/`PARSE_FAILED` 三视图、NFR-015 预期时长、逐字段增改删、低置信度高亮（非纯颜色）、"已排除类别"仅渲染类别名、缺失影响弹窗需显式同意、失败双入口、确认前离开可恢复、未确认时生成计划 disabled 带原因
  - 测试：三视图、敏感字段无值渲染、缺失同意门槛、解析失败保留输入
  - 需求：B1-5（1 ~ 9）、G8（5、6） | 设计：6.3、11.3

- [ ] FE-1.6 SCR-06 面试计划页
  - 四视图（生成中/待确认/失败/已确认只读）、NFR-016 预期时长、来源可信度与非官方标识、轮次 1~5 增删重排与六类编辑项、工具启停、9 项便利设置独立开关（默认关闭、无联动）、便利设置说明（不进入评分、文字模式口语未评估不记零）、只读说明区（60 分线/算法/解锁/交接）、未就绪轮阻止确认、部分失败仅重试失败模块、确认后冻结与报价摘要、不渲染正式问题与答案
  - 测试：便利设置默认值与独立性、未就绪轮 disabled 原因、只读区无编辑控件
  - 需求：B1-6（1 ~ 12） | 设计：4、6

- [ ] FE-1.7 SCR-07 会前检查页
  - 五项检查三态、摄像头/麦克风"关闭并继续"与不影响分数说明、NFR-007 预期、失败具体原因与替代建议、便利设置确认冻结（例外说明）、跨轮继承可调、额度不足阻止开始并给购买入口、其他设备权限不足态、单项重试不重复请求
  - 测试：额度不足 disabled 原因、重复触发同一检查项幂等
  - 需求：B1-7（1 ~ 11） | 设计：6、9.5

- [ ] FE-1.8 批次 1 合入
  - 提交标题 `feat(web-batch-1): 落地页至会前检查七个页面组（SCR-01~07, FR-001~FR-012、FR-027~FR-031）`
  - PR 正文页面→FR 映射；`CHANGELOG.md` 与 `IMPLEMENTATION_PLAN.md` 同步
  - 需求：G11（5 ~ 8）

---

## 批次 2：实时面试房间外壳 SCR-08 / SCR-09

分支 `task/frontend-batch-2-room-shell`。出口：房间布局与四类覆盖层可由 mock 事件驱动；媒体全部为桩；幂等与键盘用例通过。

- [ ] FE-2.1 房间布局与会话态视图模型
  - `RoomShell` 五区组件树；`session-view-model.ts` 的 `Record<SessionStatus, RoomViewModel>`（顶部文案、计时、计费提示、可用控件）
  - 字幕区 `aria-live="polite"`；问题固定显示可回看；无手动暂停控件；退出二次确认含"将标记为评估未完成"
  - 测试：10 个会话态逐一渲染断言、无暂停控件、退出确认文案
  - 需求：B2-1（1、2、5、8、9） | 设计：9.1、9.2

- [ ] FE-2.2 媒体桩与数字人区域
  - `AvatarMediaPort` 接口与桩实现（`hasRenderableTrack` 恒 false）；`AvatarPlaceholderFrame` 技术占位框与状态文案；候选人摄像头/麦克风可关且提示不影响分数
  - 测试：房间子树无带 `src` 的 `img`/`video`、`room.avatar.*` 文案不含模拟发言、依赖清单无媒体供应商包
  - 需求：B2-1（3、4）、G9（5） | 设计：9.3

- [ ] FE-2.3 回合修订与冻结
  - `TurnUiState` 与 `useTurnFreeze`；修订窗口三点说明；下一主问题后修订控件 disabled 带冻结说明
  - 测试：窗口内可修订、推进后冻结、修订文本保留展示
  - 需求：B2-1（6、7） | 设计：9.4

- [ ] FE-2.4 工具区与设备/响应式规则
  - 仅渲染计划已启用工具（`ToolPane` 角色化）；第二设备阻止进入与安全转移提示；<768px 对代码/白板给桌面平板建议与替代路径；房间页全宽与工具区内部滚动
  - 测试：未启用工具不出现、第二设备阻止态、移动断点提示
  - 需求：B2-1（10、11、13）、G7（4、5）、G4（5） | 设计：2.3、9.1

- [ ] FE-2.5 SCR-09 四类覆盖层与幂等
  - `useOverlayCoordinator`（优先级单实例、`eventId` 去重、5 秒窗口去重）、`useIdempotentAction`（实例级幂等键、`inFlight` disabled）
  - 四变体文案：系统故障暂停、3 分钟重连倒计时（含耗尽后评估未完成与重试入口）、降级询问（同意→文字继续/额度与口语说明；拒绝→评估未完成非失败 + 全额返还）、令牌重认证
  - `role="alertdialog"` + `aria-modal`、`role="alert"` 提示段、焦点圈定与返回
  - 测试：四变体渲染、5 秒内 3 次重复事件仅一个实例且数值不变、重复点击仅一次请求、双语关键文案快照
  - 需求：B2-2（1 ~ 11） | 设计：9.5、4.1

- [ ] FE-2.6 房间键盘与日志边界
  - Tab 序前段包含停止数字人与提交；全部房间控制键盘可达；字幕与回答文本不进入遥测
  - 测试：`tabbable` 顺序断言、遥测载荷不含字幕/回答样本
  - 需求：B2-1（12、14）、G4（1、3）、G8（1、2） | 设计：9.1、11.1

- [ ] FE-2.7 批次 2 合入
  - 提交标题 `feat(web-batch-2): 实时面试房间外壳与故障覆盖层（SCR-08、SCR-09, FR-013~FR-020）`
  - PR 正文说明媒体全部留桩、无冒充数字人输出；`CHANGELOG.md` 与 `IMPLEMENTATION_PLAN.md` 同步
  - 需求：G11（5 ~ 8）

---

## 批次 3：结果、报告与账户 SCR-10 ~ SCR-15

分支 `task/frontend-batch-3-results-report`。出口：六个页面组五态齐备；图表均有文字与表格等价版本；四类红线文案双语快照通过。

- [ ] FE-3.1 SCR-10 轮次结果页
  - 四状态视图；`ROUND_PASSED` 首屏第一条渲染固定祝贺语 + 总分/60 分线/关键维度/优势注意点/流程进度/下一轮信息 + 两个动作；`ROUND_FAILED` 累计纪要与四动作、进入下一轮 disabled 带原因；`EVALUATION_INCOMPLETE` 三类原因、"这不是失败"、系统责任返还说明；NFR-013 预期；不渲染后续轮标准答案；复核入口不在本页
  - 测试：三态图标+文字非纯颜色、祝贺语位置、评分失败自动重算文案
  - 需求：B3-1（1 ~ 8）、G10（1、2）、G1（5） | 设计：6、12

- [ ] FE-3.2 SCR-11 完整报告页
  - 十个模块；`ChartWithTextEquivalent` 落地雷达/趋势/匹配三类图；无 JD 不显百分比；必备/加分分列；沟通分析标注输入模式与证据限制（文字模式口语"未评估"不显 0）；NFR-014 预期；部分报告标注缺失模块并单模块重试；五个动作与复核一次限制；复核前后对比与原因；导出标记
  - 测试：图表文字与表格等价可访问、模块级失败其他模块可读、导出标记快照、非所有人权限不足不揭示存在性
  - 需求：B3-2（1 ~ 12）、G4（8）、G10（4） | 设计：4.1、6、16

- [ ] FE-3.3 SCR-12 练习页
  - 两视图；固定标识"练习不改变正式分数与解锁状态"常显；原题/变体入口与提示框架示例；暂停与结束；结束后逐步反馈与"不产生正式证据"说明
  - 测试：固定标识常显快照、生成失败五要素中评分影响为不影响
  - 需求：B3-3（1 ~ 6）、G10（3） | 设计：6、12

- [ ] FE-3.4 SCR-13 资产与历史页
  - 四分区独立空/失败态；简历解析记录、编辑生成新版本、替换、删除与版本标识；岗位编辑与版本；面试记录筛选与跨设备继续；活动中阻止进入与安全转移；删除活动项目确认文案；删除任务分项真实进度；训练进度薄弱点与正式重试轨迹（练习与正式分区标识）
  - 测试：活动中阻止、删除进度未完成不显完成
  - 需求：B3-4（1 ~ 8） | 设计：6

- [ ] FE-3.5 SCR-14 账户与隐私页
  - 五分区；身份绑定/解绑与双侧验证说明、冲突不合并 + 人工支持；界面语言与面试语言两个独立控件；六类独立授权开关 + 撤回即时生效、无批量开启；导出与删除的重新验证、级联删除或匿名化与财务记录说明；任务真实进度与失败重试不伪造完成
  - 测试：六类开关独立无联动、无批量开启控件、失败不显完成
  - 需求：B3-5（1 ~ 8）、G3（7）、G9（2、3） | 设计：7、4.1

- [ ] FE-3.6 SCR-15 购买与额度页
  - 四分区；报价五项与货币本地化（最小货币单位换算）；四类权益；自动续费独立未勾选 + 扣款前提醒；流水六字段；订单处理中禁止重复支付并给原因；支付失败保留方案；重复扣款自动退回与通知；发票/取消续费/申诉三入口；不收费项与付费不改评分说明；重复提交同一幂等键至多一次
  - 测试：处理中 disabled 原因、重复提交幂等、货币格式双语
  - 需求：B3-6（1 ~ 11）、G9（4、7） | 设计：7、8.2、9.5

- [ ] FE-3.7 批次 3 合入
  - 提交标题 `feat(web-batch-3): 结果、报告、练习与账户账单页面组（SCR-10~15, FR-021~FR-033、FR-040）`
  - PR 正文页面→FR 映射与四类红线文案清单；`CHANGELOG.md` 与 `IMPLEMENTATION_PLAN.md` 同步
  - 需求：G11（5 ~ 8）

---

## 批次 4：机构端与运营后台 SCR-16 / SCR-17

分支 `task/frontend-batch-4-org-admin`。出口：机构端默认最小可见与小样本保护用例通过；`apps/admin` 可构建、七页存在、控件清册无改分类控件。

- [ ] FE-4.1 机构端最小可见类型层
  - `OrgCompletionRow`（无分数/失败字段）、`Authorized<T>` 品牌类型与 `authorizeShare`、`AuthorizedShareView`
  - `packages/domain-states/src/org.ts` 可见性投影常量并纳入状态字面量白名单
  - 测试：类型层用例（未授权对象无法传入授权视图）、未授权渲染"已完成但未共享结果"且无失败/分数
  - 需求：B4-1（2、3）、G1（3） | 设计：10.1

- [ ] FE-4.2 机构端七个页面
  - 任务列表、任务配置（可配置项白名单，六项治理规则不出现）、完成情况、成员与邀请（加入前四项告知）、聚合趋势（小样本保护）、权限与审计（六角色矩阵与访问记录）、授权结果视图（范围与有效期、过期失效）
  - 退出机构后数据入口移除；未授权请求渲染权限不足态不揭示他人数据
  - 测试：9 人隐藏/10 人展示边界、无排行榜/排名/搜索控件、六项治理规则零出现
  - 需求：B4-1（1、4 ~ 12） | 设计：10.1、10.2

- [ ] FE-4.3 `apps/admin` 工程与七个骨架页
  - `next.config.ts`（`basePath: '/admin'`）、复用四个共享包、公网路径 `/admin/{locale}/<page>`
  - 七页：区域监控（仅匿名会话编号与技术指标）、供应商与模型、版本治理（发布门槛只读、无跳过控件）、来源与内容安全、客服工单（默认字段限制与授权/双人审批说明）、财务与补偿、审计日志（追加式、无删改控件）
  - 未实现操作用 disabled + 原因或只读说明块；双语与无障碍基线；未授权渲染权限不足与所需角色
  - 需求：B4-2（1 ~ 3、5 ~ 11） | 设计：2.4、10.3

- [ ] FE-4.4 「0 个改分控件」控件清册校验
  - `control-inventory.test.ts`：渲染 admin 七页与 web `org/*` 页，收集 `data-mgd-control` 清册、断言禁用词表零命中、写入快照
  - 源码级扫描：`apps/admin/src/**` 无指向评分/解锁/证据写操作的 `apiFetch` 调用
  - 页面与遥测载荷不含姓名、简历正文、回答正文与媒体引用
  - 需求：B4-2（4、11）、G9（1）、G8（1、2） | 设计：10.4、11.1

- [ ] FE-4.5 全工作区构建与批次 4 合入
  - `stage6-build` 的 `pnpm build` 过滤参数改为覆盖 `apps/web` 与 `apps/admin`（1 行）
  - 提交标题 `feat(web-batch-4): 机构端页面组与运营后台骨架（SCR-16、SCR-17, FR-034~FR-040）`
  - PR 正文说明 admin 公网路径澄清（`/admin/{locale}/…`）与 0 改分控件校验证据；`CHANGELOG.md` 与 `IMPLEMENTATION_PLAN.md` EPIC-08/EPIC-09 追踪同步
  - 需求：B4-2（1）、G11（3、5 ~ 8） | 设计：13.3、14

---

## 收尾任务

- [ ] FE-9.1 全批次覆盖清单汇报
  - 汇总 17 个页面组的路由、状态覆盖、双语与 axe 结果；列出 5 个 PR 链接与各自 FR 映射
  - 核对 IMPLEMENTATION_PLAN 第 6 节 DoD 六项在每批次均已满足
  - 需求：G11（8） | 设计：14

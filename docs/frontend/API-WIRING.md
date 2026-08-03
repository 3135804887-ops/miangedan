# 前端真实 API 接线与全栈联调（任务 6）

追踪：IMPLEMENTATION_PLAN.md 前端交付追踪；`docs/api/openapi.yaml`；
`apps/web/src/lib/api-fetch.ts`；`apps/web/src/mocks/handlers/index.ts`。

## 1. 17 页数据来源盘点（改造前 → 接线后）

| 页面 | 改造前数据源 | 接线后端点（apiFetch，契约路径） | 占位说明 |
|---|---|---|---|
| SCR-01 落地页/演示 | 静态内容 | 无（纯内容页） | — |
| SCR-02 登录/注册 | apiFetch | `POST /v1/identity/email/challenges`、`.../verify`、`/v1/identity/oauth/{provider}/verify` | 已接线 |
| SCR-03 工作台 | apiFetch | `GET/POST /v1/projects`、`PATCH /v1/projects/{id}`、`duplicate`、`DELETE` | 已接线 |
| SCR-04 创建项目 | 内联表单 | `POST /v1/projects`（上传/解析端点已登记：`/v1/uploads/resumes`、`/v1/parsing/resumes`、`/v1/jobs`） | 501 时保持占位文案 |
| SCR-05 解析校对 | 内联 mock 字段 | `GET /v1/projects/{id}` → `GET /v1/resumes/{resumeId}/versions/{version}`、`/v1/jobs/{jobId}/versions/{version}`、`POST .../confirm` | 无材料引用时展示合成占位 |
| SCR-06 计划 | 内联 `mockPlan` | `GET /v1/projects/{id}/plan`、`POST plan:generate`、`PATCH plan`、`POST plan:confirm` | 501 占位提示 |
| SCR-07 会前检查 | 内联检测 | `POST /v1/projects/{id}/rounds/{seq}/session`、`GET /v1/sessions/{id}/precheck`、`POST precheck/freeze` | API 失败回退合成检测并标注 |
| SCR-08 实时房间 | 内联演示 | `GET /v1/sessions/{id}`、`transcripts`、`revisions`、`turns/{i}/freeze`、`timer/pause/resume`、`downgrade/offer/accept/decline`、`end`、`reconnect`、`tools/*` | 媒体/WebRTC 仍为合成（供应商选型后接真实） |
| SCR-09 故障覆盖层 | （SCR-08 内） | 同上（故障动作走契约端点） | 同上 |
| SCR-10 轮次结果 | 内联常量 | `GET /v1/projects/{id}/rounds/{seq}/result` | 501 占位提示 |
| SCR-11 完整报告 | 内联常量 | `GET /v1/projects/{id}/report`、`POST report/export`、`POST .../review`、`POST /v1/deletion-tasks` | 501 占位提示 |
| SCR-12 练习 | 内联内容 | `POST /v1/projects/{id}/practice`、`POST /v1/practice/{id}/end` | 501 占位提示 |
| SCR-13 资产 | 内联列表 | `GET/DELETE /v1/library/resumes`、`GET/DELETE /v1/library/jobs` | 面试记录/训练进度无契约端点，保留合成并标注 |
| SCR-14 账户与隐私 | 内联 | `GET/PUT /v1/me/preferences`、`GET /v1/consent/grants`、`POST .../withdrawals`、`GET /v1/me/export`、`POST /v1/me/deletion` | 501 占位提示 |
| SCR-15 购买与额度 | 内联 | `GET /v1/entitlements`、`POST /v1/quotes`、`POST /v1/orders`、`GET /v1/pricing/{region}`、`GET /v1/usage-ledger`、`GET /v1/subscription`、`PUT auto-renew`、`POST cancel` | 订单列表无契约 GET，保留占位说明 |
| SCR-16 机构端（7 页） | 内联 | members/audits/aggregates/shares/assignments 全部契约端点（详见页面） | 任务列表 GET 缺失→占位；创建/发布/关闭走 POST |
| SCR-17 运营后台 | 内联 | `GET /v1/admin/regions`、`providers`、`audit-logs`、`data-rights`；`PUT providers/{id}/status`、`POST break-glass`、`tickets` 等 | 无列表端点的分区保留占位说明 |

## 2. Mock_Layer 语义

- `NEXT_PUBLIC_MGD_MOCKS=on`：浏览器加载 MSW Service Worker，`/api/v1/*` 全部由
  `apps/web/src/mocks/handlers/index.ts` 返回 `synthetic: true` 合成数据；未登记的端点
  统一返回 501，避免测试静默通过。
- 关闭 MSW（生产/联调）：同一份页面代码经 `api-fetch.ts` 直接请求同源 `/api/v1/*`；
  `api-fetch` 的读/写、幂等键、错误信封语义未改动。
- 测试（vitest）：`apps/web/vitest.setup.ts` 启动 MSW Node server，`onUnhandledRequest:
  'error'` 保证任何遗漏端点立即失败。

## 3. 全栈本地联调

### 3.1 Go 服务启动参数（区域 fail-closed）

每个 Go 服务模块（`services/*/cmd/*`）启动前必须提供：

```bash
export DATA_REGION=cn            # cn | eu | intl
export INFRA_REGION=cn           # 必须与 DATA_REGION 一致，不一致拒绝启动（TASK-002）
export SERVICE_ENV=dev
```

示例（本地起 identity + project 两个控制面服务）：

```bash
DATA_REGION=cn INFRA_REGION=cn SERVICE_ENV=dev go run ./services/identity/cmd/identity
DATA_REGION=cn INFRA_REGION=cn SERVICE_ENV=dev go run ./services/project/cmd/project
```

真实本地全栈（22 个 Go 模块 + 5 个 Python 服务 + PostgreSQL/Redis/Temporal/对象存储）
依赖 EPIC-01 基础设施；无基础设施时可只起“无外部依赖”的控制面服务用于契约联调。

### 3.2 前端 → 网关路径

- 本地单机：设置 `MGD_API_ORIGIN=http://127.0.0.1:8080`（服务端专用、非 NEXT_PUBLIC
  白名单键，网关地址），
  `apps/web/next.config.ts` 的 env 门控 rewrite 将 `/api/:path*` 代理到网关；未设置时
  保持同源 `/api`。
- 生产：边缘身份/API 网关（SYSTEM-ARCHITECTURE 5.2 GW）按账户 `data_region` 路由，
  前端与网关同域，`/api/v1/*` 由网关鉴权、限流并转发对应区服务；区域不匹配拒绝
  （DEPLOYMENT 第 6 节，不自动跨区转发）。

## 4. 门禁

- `pnpm lint`、`pnpm test`（vitest 全量）、`pnpm typecheck`、
  `pnpm --filter @mgd/web run build`、`pnpm i18n:check`、`pnpm tokens:check-*`、
  `pnpm api:check` 全部为 PR 硬门禁（CI 阶段2/3/6）。

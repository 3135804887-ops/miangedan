# billing 服务（权益/订单/账本）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go（控制面服务，每服务一个模块，经根 `go.work` 统一工作区） |
| 拥有任务 | EPIC-07（TASK-060 ~ TASK-065） |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 当前状态

TASK-001 工程骨架 + TASK-002 区域自检：最小入口 `cmd/billing`；启动时校验
`DATA_REGION` 与 `INFRA_REGION` 一致且 `SERVICE_ENV` 合法（共享包 `services/region`，
正常/异常单测已配）。

## 已实现（TASK-060 首次业务实现，FR-031）

- **权益模型**：免费 60 分钟（首次登录，幂等每人一份）、单项目包（覆盖已确认
  轮次 + 每失败轮一次正式重试）、Pro 订阅（月额度、结转 ≤1 账期、总余额
  ≤2×月额度）、时长加油包；`Balance` / `CanReserve`（余额校验只在轮次开始前，
  已开始轮次不中断）。
- **报价引擎**：`CreateQuote`（轮次/时长/重试/税费/有效期；区域化合成定价，
  OD-02 未决前确定性可测）→ `PresentQuote` → `AcceptQuote`（QUOTE_ACCEPTED +
  计费版本冻结）；开始前计划修改 → `RecalculateQuote`（版本递增）；开始后冻结
  拒绝重新报价（ErrQuoteFrozen）。
- 迁移 `0060_billing_entitlements.sql`；DATA-MODEL 增加 billing_freezes。

## 规划（后续任务）

- TASK-061 秒级 UsageLedger；TASK-062 支付集成；TASK-063 退款；
  TASK-064 Pro 订阅；TASK-065 发票/收据。

红线：付费永不影响评分；重复扣费为 0；系统故障自动全额返还（TASK-061/063 落地）。

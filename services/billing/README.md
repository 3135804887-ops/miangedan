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
- **秒级 UsageLedger**（TASK-061，FR-032）：`Reserve`（每轮开始前预留，不足
  阻止开始；消费顺序 免费→项目包（限本项目）→Pro→加油包）、`StartMetering`/
  `StopMetering`（只计 LIVE 秒；故障/等待/重连/认证暂停与降级后不计）、
  `Settle`（按实际扣减 + 冲正释放未使用预留；用户主动退出同规则）、
  `RefundFull`（系统责任自动全额返还本轮预留，冲正条目）；
  账本追加式（reserve/consume/reversal），幂等键去重，逐笔可查。
- **区域化支付集成**（TASK-062，FR-033，US-06 场景 4）：Order 状态机
  （PAYMENT_PENDING → PAID / PAYMENT_FAILED / PAYMENT_TIMEOUT），创建订单幂等键
  去重；支付回调 HMAC-SHA256 验签 + ±5 分钟重放窗口 + payment_event_id 去重；
  支付成功未到账保持 PAYMENT_PENDING，对账任务按 provider_txn_id 收敛；
  同一订单只记一次权益和一次扣款；重复扣款自动识别原路退回（写 Incident）；
  支付状态不明禁止重复发起扣款。迁移 `0062_payment_orders.sql`。
- **退款与补偿**（TASK-063，FR-033，US-06 场景 3）：小额用户退款自动执行；
  大额（≥¥500 等值）与人工补偿双人审批（同一审批人去重、不可自批、并发原子）；
  系统故障自动全额执行；拒绝说明原因、可申诉；执行幂等，账本追加 refund
  冲正条目记录原因；退款不影响评分/复核/解锁。

## 规划（后续任务）

- TASK-064 Pro 订阅生命周期；TASK-065 发票/收据与区域税费。

红线：付费永不影响评分；重复扣费为 0；系统故障自动全额返还（TASK-061/063 落地）。

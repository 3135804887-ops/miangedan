# adminapi 服务（运营治理后台 BFF）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go（控制面服务，每服务一个模块，经根 `go.work` 统一工作区） |
| 拥有任务 | EPIC-09（TASK-080 ~ TASK-085） |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## 当前状态

TASK-001 工程骨架 + TASK-002 区域自检：最小入口 `cmd/adminapi`；启动时校验
`DATA_REGION` 与 `INFRA_REGION` 一致且 `SERVICE_ENV` 合法（共享包 `services/region`，
正常/异常单测已配），无业务实现。
业务实现按拥有任务推进；开工前必读 AGENTS.md 及该任务对应的契约文档（领域、API、数据与安全）。

## 已实现（TASK-080 首次业务实现，FR-037）

- **运营监控**：区域在线房间/排队/容量/供应商健康/SLO/错误预算；房间快照仅
  匿名会话编号与技术状态（无姓名/简历/回答/媒体）。
- **供应商状态变更**：active/ramping/disabled；停用必须记录原因并写审计。
- **运营红线**：OperatorSessionGuard 拒绝加入/旁听/代答并写审计；
  后台角色与跨区访问校验。
- 迁移 `0068_admin_ops.sql`（ops_providers/ops_room_snapshots/ops_region_status）。

## 已实现（TASK-081，FR-038）

- **版本治理**：模型/提示词/量表/工作流版本注册（offline→shadow→canary→full
  逐级推进）；灰度门槛=结构兼容+安全测试、放量门槛=指标通过；
- **冻结与回滚**：项目固定开始版本（活跃正式面试不可中途改变）；回滚仅限
  无进行中会话（新会话回稳定版）；
- **量表停用**：产品/面试专业/安全公平三方审批；不批量改写历史分数。
- 迁移 `0069_version_registry.sql`（artifact_versions/version_pins）。

## 已实现（TASK-082，FR-039）

- **禁止改分系统级约束**：编辑分数/解锁/改证据一律拒绝并写审计；正式复核唯一
  入口；无分数修改存储路径（与前端 control-registry 红线呼应）。
- **破窗访问**：限重大安全/法律事件、理由+时长 ≤8h、72h 内事后复核、不可自审、
  到期自动 expired、敏感访问记录可查；break_glass 与审计存储仅 SELECT/INSERT。
- 迁移 `0070_break_glass.sql`。

## 已实现（TASK-083，FR-040）

- **数据权利请求**：delete/export/correct/withdraw 工单化（幂等）；复用
  export 删除编排骨架：六层真实进度 + 级联删除逐项可追踪；法定财务记录保留
  但解除内容关联；失败如实 FAILED 且可重试。
- 迁移 `0071_data_rights.sql`。

## 已实现（TASK-084，FR-037/FR-040）

- **追加式审计**：管理员不可删除；存储仅 SELECT/INSERT（反射断言无修改路径）；
  分页查询。
- **抗钓鱼 MFA**：设备公钥绑定、随机 nonce 挑战（5 分钟一次性）、HMAC 验证、
  挑战不可重放；高风险操作 15 分钟窗口重新验证。
- 迁移 `0072_admin_mfa.sql`。

## 规划（后续任务）

- TASK-085 客服工单。

# services/provider — 供应商中立适配层（TASK-030）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（控制面共享包，仅依赖 services/region，零外部依赖） |
| 拥有任务 | TASK-030（EPIC-04）；各能力适配器实现随对应实时/AI 任务接入 |
| 追踪 | docs/ai/PROVIDER-ADAPTERS.md（唯一契约）；ADR-0003；PRD 架构决策 4；NFR-007 ~ NFR-012 |

## 职责

- **统一能力接口**：LLM / ASR / TTS / Avatar / Search 五类能力（`Capability`）与供应商注册条目
  （`Info`：provider_id、能力、数据区、语言、角色、**固定版本**）。
- **注册表**：按数据区隔离（ADR-0005）；`provider_id` 形态 `{capability}_{region}_{role}`
  （如 `llm_cn_primary`）；重复注册拒绝；`SetStatus(disabled)` 支持紧急停用（US-08 场景 5）。
- **健康检查**：`Registry.Health` 低频合成探针（§6；未配置探针以被动指标为准，§11）。
- **熔断**：每（区 × 供应商 × 能力）`CircuitBreaker`（§7.2：closed → open → half_open → closed；
  注入时钟可测；参数随被动指标校准）。
- **区域路由**：`Router.Route` 新会话按（区域、能力、语言）选择已验证健康供应商，
  主 open 切 secondary；主备均不可用拒绝新会话（§5：无已验证供应商 = 不创建新会话，不静默降级）。
- **会话钉扎**：`Pin` / `Resolve` 固定活跃正式面试的供应商与版本（§7.3、§9）；
  被停用/版本变化返回 `ErrPinnedUnavailable`——**活跃正式会话不静默切换**，故障走状态机路径。

## 红线（PROVIDER-ADAPTERS §10）

1. 业务代码只依赖本层接口语义，禁止直连厂商 SDK。
2. 无已验证供应商 = 拒绝新会话；跨区不得回退。
3. 活跃正式面试固定版本，不中途无记录切换。
4. 凭证按区隔离、密钥系统管理、日志零输出。

# services/temporal — Temporal 区域契约共享包

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（控制面共享包） |
| 拥有任务 | TASK-004（EPIC-01）；业务工作流实现随 TASK-017 |
| 追踪 | ADR-0001、ADR-0005；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节 |

## 职责

- 区域命名空间契约：`mgd-{region}-{env}-temporal`，与 `DATA_REGION`/`SERVICE_ENV` 严格绑定。
- 七域任务队列契约：`ingestion` / `plan` / `interview` / `scoring` / `report` /
  `billing` / `deletion`，集合齐全、无重复、无未知队列。
- 配置校验 `ValidateConfig`：命名空间与任务队列任一不符即失败（fail-closed）。

## 用法

```go
namespace, err := temporal.Namespace("cn", "production")
err = temporal.ValidateConfig("cn", "production", namespace, temporal.AllTaskQueues)
```

## 红线

1. 区域间无共享命名空间、无跨区工作流引用（ADR-0005）。
2. 任务队列只允许七域固定集合；新增队列必须先在模块契约登记。
3. 工作流跨 AZ 故障可恢复由集群拓扑（`infra/modules/temporal`）保证。

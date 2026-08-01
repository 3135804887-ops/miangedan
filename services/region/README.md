# services/region — 三数据区共享包

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（控制面共享包） |
| 拥有任务 | TASK-002（EPIC-01）；网关接线随 EPIC-02 边缘任务 |
| 追踪 | ADR-0005；docs/architecture/DEPLOYMENT.md 第 9 节；SEC-051 |

## 职责

- 数据区枚举 `cn / eu / intl`（OD-09）与严格解析。
- 启动自检：`DATA_REGION` 与 `INFRA_REGION` 一致、`SERVICE_ENV` 合法，不一致拒绝启动（fail-closed）。
- 区域路由：账户归属区域与请求入口区域一致才放行；不匹配返回 `region_mismatch`（与 OpenAPI 错误码一致）。

## 用法

```go
err := region.CheckStartup(os.Getenv("DATA_REGION"), os.Getenv("INFRA_REGION"), os.Getenv("SERVICE_ENV"))
decision, err := region.Route(accountRegion, requestRegion)
```

## 红线

1. 区域代码只允许 `cn / eu / intl`，大小写与空白不宽容。
2. 任何服务不得在未通过启动自检的情况下连接区域基础设施。
3. 跨区请求拒绝而非转发（ADR-0005）。

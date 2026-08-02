# services/evidence — 追加式证据账本写入管道（TASK-026）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（共享包，仅依赖 services/region；生产由实时链路消费） |
| 拥有任务 | TASK-026（EPIC-03） |
| 追踪 | NFR-005；ADR-0004；docs/data/DATA-MODEL.md §5.3；docs/domain/DOMAIN-MODEL.md §6.11 |

## 职责

- 问题实际播放内容、回答、修订、工具事件四类证据的**只追加**写入管道；
- `event_id` 幂等去重（NFR-006）；`content_hash` 与载荷一致性校验（fail-closed）；
- 无更新/删除路径（ADR-0004）；列表返回只读副本。

## 用法

```go
store := evidence.NewMemoryStore()
svc, _ := evidence.NewService(store)
entry, err := svc.AppendVerified(ctx, "cn", evidence.AppendInput{
	SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1,
	Kind: evidence.KindQuestionPlayed, EventID: "ev-1",
	PayloadJSON: payload,
}, evidence.HashPayload(payload))
```

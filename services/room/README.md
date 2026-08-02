# services/room — 实时会话房间（TASK-020）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（控制面共享服务，依赖 services/project 与 services/region） |
| 拥有任务 | TASK-020（EPIC-03）；媒体链路（数字人/ASR/字幕）随 TASK-021~027 |
| 追踪 | FR-013、NFR-007；SEC-003；docs/domain/INTERVIEW-STATE-MACHINE.md 6.2；docs/api/realtime-events.md |

## 职责

- **会话房间**：按 openapi 契约创建/查询/结束会话（`POST /v1/projects/{projectId}/rounds/{sequence}/session`、
  `/v1/sessions/*`），前置校验：项目 READY、本轮量表与覆盖方案就绪（FR-011）、单活动设备（TASK-018）；
  交接包（TASK-034）与额度预留（TASK-061）为后续挂接点。
- **短期媒体令牌**（SEC-003）：HMAC-SHA256 签名、分钟级 TTL、一次性（nonce 消费）、按 nonce 吊销；
  与业务令牌**相互隔离**（独立签名密钥经 `*_REF` 注入、claims 仅媒体面用途，不含业务身份）。
- **会话状态**：ROOM_CREATED → … → ENDED（INTERVIEW-STATE-MACHINE 6.2 枚举）；
  reconnect 3 分钟窗口（超窗 `reconnect_expired`）、设备安全转移（原设备令牌立即失效）。
- **房间提供方适配**：`Provider` 供应商中立契约 + 合成桩（ADR-0003；LiveKit 为技术基线，
  真实接入随供应商选型）。

## 用法

```go
store := room.NewMemoryStore()
tokens, _ := room.NewMediaTokenManager(room.TokenConfig{
	SigningKey: os.Getenv("MEDIA_TOKEN_SIGNING_KEY"), // *_REF 注入，>=32 字符，与业务密钥隔离
}, store)
svc, _ := room.NewService(store, store, tokens, room.StubRoomProvider{}, projects)
```

## 红线

1. 媒体令牌只用于媒体面；浏览器不持有任何供应商密钥（PRD API 原则）。
2. 令牌一次性、可吊销、短期；签发/校验密钥必须与业务令牌隔离（SEC-003）。
3. 媒体令牌不携带业务正文/令牌；日志与追踪只含匿名技术标识（SEC-008、SEC-032）。

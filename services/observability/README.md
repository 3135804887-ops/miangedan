# services/observability — 观测基线共享包（OpenTelemetry）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（控制面共享包）；OpenTelemetry v1.44.0 |
| 拥有任务 | TASK-005（EPIC-01）；业务指标与追踪埋点随各服务任务落地 |
| 追踪 | PRD Observability and Operations；SEC-032、SEC-033；docs/observability/LOGGING-POLICY.md |

## 职责

- **SDK 级日志脱敏**（SEC-032）：`NewLogger` 返回结构化 JSON 日志器，strict 模式自动套用
  脱敏处理器——敏感键（token/secret/password/credential/authorization/cookie/otp/key/
  resume/transcript/answer/raw）整值替换；消息与普通属性中的 JWT、Bearer、`sk-`、超长不透明串
  按值模式替换；生产环境强制 strict。
- **OpenTelemetry 装配**：`Setup` 按配置创建 OTLP HTTP 指标/追踪导出器与 SDK Provider，
  资源携带 `service.name` / `service.version` / `deployment.environment` / `data_region`；
  任一导出器为 `none` 时跳过对应 Provider，观测故障不影响业务链路。
- **指标/追踪属性白名单**：`ValidateAttributes` 只放行 `docs/observability/LOGGING-POLICY.md`
  列出的匿名技术标签（data_region/language/input_mode/provider/job_family/version 等），
  白名单外、敏感键或疑似敏感值一律拒绝；强制携带 `data_region`（ADR-0005）。

## 用法

```go
cfg := observability.Defaults("identity", "cn", "production")
cfg.OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
logger, shutdown, err := observability.Setup(context.Background(), cfg)
defer shutdown(context.Background())
logger.Info("启动完成", slog.String("data_region", cfg.DataRegion))
```

## 红线

1. 生产环境 `REDACTION_MODE` 只能是 `strict`（SEC-032）。
2. 禁止把简历正文、完整回答、令牌、原始媒体写入日志或作为标签/追踪属性。
3. 指标与追踪属性必须通过 `ValidateAttributes` 白名单，且必须携带 `data_region`。
4. 脱敏管道故障时宁可丢弃日志，不得泄露正文（SYSTEM-ARCHITECTURE 第 10 节）。

# 区域事件流模块

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-003；docs/data/DATA-MODEL.md 第 8 节 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.event_stream` |

## 主题

`parse.jobs`、`scoring.requests`、`report.jobs`、`notification.outbox`、
`deletion.tasks`、`compensation.jobs` 六类主题按区域独立创建。

## 载荷规则

- 按 `session_id` / `project_id` 分区保序。
- 载荷不含简历正文、完整回答、原始媒体；事件名与 `docs/api/realtime-events.md` 一致。

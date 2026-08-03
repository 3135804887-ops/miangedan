# services/export（数据导出与删除编排，TASK-055）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26，仅依赖 `services/region`；gofmt + go vet + golangci |
| 追踪 | docs/data/RETENTION-MATRIX.md；FR-040；US-05 场景 5；DOMAIN-MODEL |

## 已实现

- **导出任务**：`CreateExport` / `ExecuteExport` / `GetExportTask`
  - 异步任务真实进度可查（queued → running → succeeded）；
  - 导出物必带训练用途标记"模拟训练结果，不代表真实企业录用结论"；
  - account / project 两种范围；项目导出必须携带 project_id；幂等。
- **删除任务**：`CreateDeletionTask` / `ExecuteDeletion` / `RetryDeletionTask`
  - 目标类型 project / resume / job / account（级联语义由编排层按
    RETENTION-MATRIX §6 执行）；
  - 六层真实进度（database/cache/search_index/object_storage/backups/
    third_party_processors），任一层失败 → FAILED、仅失败任务可重试；
  - 法定财务记录保留但解除内容关联（legal_retention_note）；幂等。
- **到期提醒**：`ExpiringItems` 按 30/7 天窗口扫描（RETENTION-MATRIX §5）。
- **HTTP**：`GET /v1/me/export`、`POST /v1/deletion-tasks`、
  `GET /v1/deletion-tasks/{taskId}`、`POST /v1/projects/{projectId}/report/export`。

红线：删除必须是真实删除或不可逆匿名化；进度对用户可见、失败可重试；
不伪造完成状态；导出物不含后续轮次正式考点与原始媒体。

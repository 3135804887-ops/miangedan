# services/migrate — 数据平台迁移工具

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26 + pgx v5（PostgreSQL 专用迁移 CLI） |
| 拥有任务 | TASK-003（EPIC-01）；业务表迁移随各自拥有任务追加 |
| 追踪 | docs/data/DATA-MODEL.md；ADR-0004；NFR-005、NFR-006 |

## 职责

- 幂等执行 PostgreSQL 迁移：`schema_migrations` 记录版本与 SHA-256 校验和，重复执行自动跳过。
- 已应用迁移的 SQL 校验和变化即失败（fail-closed），禁止修改历史迁移。
- 基线迁移 `0001_ledger_baseline.sql` 落地四张追加式账本表
  （`evidence_items`、`score_versions`、`usage_ledger`、`access_audits`）及数据库层
  `REVOKE UPDATE, DELETE` 约束（ADR-0004）。
- `0010_identity_accounts.sql` 落地 TASK-010 用户/身份、短期验证、会话轮换、身份冲突恢复与
  幂等结果引用；强制区域内主体唯一且不保存邮箱、验证码、OAuth 授权码、证明或令牌明文。
- `0012_resume_uploads.sql` 落地 TASK-012 的 `resume_uploads` / `upload_scan_attempts`，
  强制 10 MiB 上限、所属区域 uploads 桶、上传与扫描重试两级幂等唯一键。
- `0013_resume_parsing.sql` 落地 TASK-013 的 `resumes` / `resume_parse_attempts` /
  `resume_versions`，强制追加式版本、操作幂等、敏感根键二次阻断与低置信度清零后确认。

## 用法

```bash
cd services/migrate
DATA_REGION=cn INFRA_REGION=cn SERVICE_ENV=dev DATABASE_URL=postgres://... \
  go run ./cmd/migrate up
DATA_REGION=cn INFRA_REGION=cn SERVICE_ENV=dev DATABASE_URL=postgres://... \
  go run ./cmd/migrate status
```

`DATA_REGION` 与 `INFRA_REGION` 不一致时工具拒绝启动（TASK-002 区域自检）。

## 红线

1. 迁移文件命名 `NNNN_name.sql`；版本全局唯一，禁止改写已应用迁移。
2. 追加式账本表的 `UPDATE`/`DELETE` 只能授权给删除编排专用角色。
3. Redis 不作为证据唯一存储；证据持久化由 PostgreSQL/事件流保证（NFR-005）。

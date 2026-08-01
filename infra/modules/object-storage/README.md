# 对象存储模块（S3 兼容，区域化）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-003；docs/data/DATA-MODEL.md 第 6 节；RETENTION-MATRIX 第 4 节 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.object_storage` |

## 桶隔离

- `uploads`：原始简历/JD，restricted，仅签名 URL。
- `exports`：用户导出物，restricted，短时效签名 URL。
- `media`：明确授权的原始音视频，restricted，**默认为空**，30 天生命周期自动过期。

媒体桶 30 天生命周期由对象存储生命周期规则强制，应用层另有校验任务（RETENTION-MATRIX）。

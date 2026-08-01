# 对象存储模块（S3 兼容，区域化）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-003；docs/data/DATA-MODEL.md 第 6 节；RETENTION-MATRIX 第 4 节 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.object_storage` |

## 桶隔离

- `uploads`：原始简历/JD，restricted，仅签名 URL。`quarantine/` 仅供 TASK-012 无网络一次性沙箱扫描；
  安全通过后移动到 `accepted/`，后续解析器只读 accepted；拒绝文件删除，扫描暂时失败原件保留以供幂等重试。
- `exports`：用户导出物，restricted，短时效签名 URL。
- `media`：明确授权的原始音视频，restricted，**默认为空**，30 天生命周期自动过期。

媒体桶 30 天生命周期由对象存储生命周期规则强制，应用层另有校验任务（RETENTION-MATRIX）。

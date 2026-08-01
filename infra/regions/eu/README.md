# 数据区：eu

| 字段 | 内容 |
|---|---|
| 区域代码 | `eu`（OD-09） |
| 状态 | TASK-002 已落地：`envs/{dev,staging,production}.yaml` 三环境拓扑实例 |
| 追踪 | ADR-0005；docs/architecture/DEPLOYMENT.md；docs/security/PRIVACY-DATA-MAP.md |

## 本区独立持有的资源

网络、PostgreSQL、Redis（非证据存储）、对象存储三桶（uploads / exports / media）、区域事件流、SFU 节点、Temporal 集群与命名空间、密钥引用（`*_REF` 模式）、供应商白名单、邮件/通知通道。

## 隔离红线

1. 与其他数据区（cn、intl）无 VPC 对等、无数据复制、无共享密钥。
2. 本区用户内容不因容灾、成本或便利原因出境。
3. `[REGION-SCOPED]` 环境变量按本区密钥管理系统独立配置（见 `.env.example`）。

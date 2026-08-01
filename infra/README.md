# infra — 基础设施即代码（供应商中立）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-001（目录骨架）；TASK-002 ~ TASK-008（实例化）；ADR-0005 |
| 约束 | 供应商中立抽象（OD-01 未决）；区域间零默认通路；fail-closed |

## 结构

- `modules/`：可复用 IaC 模块（网络、数据库、对象存储、事件流、SFU、Temporal）；
  TASK-003 已落地 `database/`、`object-storage/`、`event-stream/`，TASK-004 落地 `temporal/` 模块契约。
- `regions/`：cn / eu / intl 三数据区实例化配置；每区 `envs/{dev,staging,production}.yaml`
  共 9 个环境拓扑实例（TASK-002）。
- 区域拓扑由 `python tools/validate_docs.py --suites regions` 自动校验（区域/环境/3 AZ/
  资源命名/无跨区引用），配置错误即 CI 失败，防止静默跨区。

## 红线

1. 无跨区 VPC 对等、无跨区数据复制、无共享密钥（ADR-0005）。
2. 区域配置错误必须导致部署/启动失败，禁止静默跨区（fail-closed）。
3. 所有环境只有合成数据；真实用户数据不进入 dev/staging。
4. 密钥只进密钥管理系统；仓库、CI 日志、后台零明文（`*_REF` 引用模式，见 `.env.example`）。

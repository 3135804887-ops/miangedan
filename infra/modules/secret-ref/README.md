# 密钥管理模块（每区 KMS + REF 引用）

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-006；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；SECURITY-REQUIREMENTS SEC-012、4.7 |
| 实例化 | `infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.secrets` |

## 规则

- 每数据区独立 KMS（`kms_name`），密钥按区域隔离；跨区引用视为配置错误（ADR-0005）。
- 仓库内只允许 `*_REF` 引用名与普通环境变量名（`services/secretref` 校验 fail-closed）；
  真实值只存在于密钥管理系统，后台任何界面不展示完整密钥（SEC-012）。
- 轮换周期与责任人以 `SECURITY-REQUIREMENTS.md` 4.7 周期表为准；演练不中断服务，
  演练脚本 `tools/secret-rotation/rotation_drill.py` 已接入 CI 阶段 1。

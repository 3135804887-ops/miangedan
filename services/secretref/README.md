# services/secretref — 密钥引用契约共享包

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（控制面共享包，零外部依赖） |
| 拥有任务 | TASK-006（EPIC-01）；KMS 供应商接入随基础设施落地 |
| 追踪 | SECURITY-REQUIREMENTS SEC-012、4.7 密钥轮换周期表；docs/operations/KEY-ROTATION-RUNBOOK.md |

## 职责

- **引用契约**：密钥只允许以 `*_REF` 引用变量注入（`ValidateRefName`）；区域拓扑
  `secrets.refs` 值必须是合法环境变量名且不得内联真实密钥（`ValidateRefs` fail-closed，SEC-012）。
- **展示脱敏**：`MaskSecret` 只保留末 4 位（后台/日志不得展示完整密钥）。
- **轮换支持**：轮换流程、责任人周期与演练见 `docs/operations/KEY-ROTATION-RUNBOOK.md`
  与 `tools/secret-rotation/rotation_drill.py`（CI 阶段 1 执行）。

## 红线

1. 仓库、配置、文档、测试零明文密钥；真实值只进密钥管理系统。
2. 引用值出现长随机串/JWT/`sk-` 等形态一律拒绝（fail-closed）。
3. 后台任何界面不展示完整密钥（SEC-012）。

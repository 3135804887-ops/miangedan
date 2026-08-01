# tools/ci — CI 辅助脚本与本地检查

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-001；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.3 节 |
| 门禁定义 | `.github/workflows/ci.yml`（以流水线定义为准，本目录提供本地复现手段） |

## 本地检查

- 一键执行阶段 1~4 + 6（含阶段 5 本地等价项）：`bash tools/ci/run_local_checks.sh`
- 全量规范校验：`python tools/validate_docs.py`（默认全部 14 套件）
- 分套件示例：`python tools/validate_docs.py --suites yaml,json,schema,openapi,regions,data-platform,temporal`（套件键见 `tools/validate_docs.py` 的 `SUITES`；`regions` 为三数据区拓扑校验，`data-platform` 为数据平台契约校验，`temporal` 为 Temporal 命名空间/任务队列契约校验）

## 说明

- 阶段 5 的 gitleaks、govulncheck、pip-audit、SBOM 在 CI 平台执行；本地等价物为仓内密钥模式扫描（`--suites secrets`）。依赖漏洞门禁使用 govulncheck + pip-audit（GitHub 原生 dependency-review-action 在私有仓库需 GHAS 许可，转公开或购 GHAS 后可换回）。
- SAST 基线 = golangci-lint（gosec）+ ruff S 规则集；平台级 SAST（如 CodeQL）如需引入，按变更流程评估后扩展阶段 5。
- 主分支追加阶段（集成测试 / 评测集回归 / dev 部署）分别挂接 TASK-004、TASK-036、TASK-002。
- Python 工具链统一经 `tools/requirements-dev.txt` 安装。

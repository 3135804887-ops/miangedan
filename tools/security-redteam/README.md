# 安全红队自动化套件（TASK-093）

追踪：IMPLEMENTATION_PLAN.md TASK-093；docs/security/THREAT-MODEL.md；
SECURITY-REQUIREMENTS（零容忍：注入、越权、跨租户、重复扣费、证据丢失）。

## 覆盖范围

六类攻击，每类包含正常用例（正常路径仍可用）与攻击用例（断言命中即阻断）：

| 类 | 威胁编号 | 攻击用例示例 |
|---|---|---|
| 提示注入 | TM-01 | 中英文/编码混淆/工具诱导/JD 注入/练习回答注入均按数据处理，零泄露 |
| 越权（IDOR） | TM-03 | 未授权访问、跨用户资源查询、缺 Bearer、角色/权限矩阵越权 100% 拒绝 |
| 跨租户/跨区泄露 | TM-04/TM-05 | 个人排名面不存在、审计最小可见、区域钉扎、区域路由拒绝 |
| 恶意文件 | TM-02 | 病毒/宏/压缩炸弹/伪装/超限/损坏/加密矩阵全部拒绝且原因具体 |
| 重放 | TM-16/TM-06 | 支付回调重放、MFA 挑战重放、转写/证据/刷新令牌幂等 |
| 重复扣费 | TM-06 | 回调去重、重复扣款自动退回、订单幂等、预留幂等 |

## 运行

```bash
# 本地执行全部选择器并生成报告（写入 ai/evals/reports/redteam.json）
python tools/security-redteam/run_redteam.py --write

# CI 门禁：执行并与入库报告比对，不一致或任一攻击用例失败即阻断
python tools/security-redteam/run_redteam.py --check
```

选择器清单位于 `tools/security-redteam/manifest.json`（工作目录 + 命令数组，指向
仓库真实 Go/Python 测试）。任何相关测试或红队用例变更后必须重新 `--write` 并入库，
否则 CI 阶段5 `--check` 失败（0 失败门禁）。

## 断言语义

- 攻击选择器通过 = 对应威胁被阻断（拒绝/隔离/去重/审计），与 THREAT-MODEL 验收一致；
- 正常选择器通过 = 对应正常路径未被误伤；
- 任一选择器失败 → 套件 `passed=false` → CI 阻断合并。

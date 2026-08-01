# apps/web — 用户端 Web / PWA

| 字段 | 内容 |
|---|---|
| 技术基线 | Next.js + React + TypeScript（AGENTS.md 第 3 节） |
| 拥有任务 | EPIC-02（账户/材料/计划界面）、EPIC-03（实时面试房间）、EPIC-06（报告/训练） |
| 追踪 | docs/design/SCREEN-SPEC.md（17 页面清单）、DESIGN-SYSTEM.md、ACCESSIBILITY.md |

## 当前状态

TASK-001 仅建立目录。首个前端任务落地时引入 package.json 与 lockfile，并将 TS 静态检查与构建接入 CI 阶段 2/6（流水线中已留挂接注释）。

开工前必读：AGENTS.md、`docs/design/` 三份规范、`docs/api/openapi.yaml`、`docs/api/realtime-events.md`。

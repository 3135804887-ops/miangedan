# apps/admin — 运营治理后台

| 字段 | 内容 |
|---|---|
| 技术基线 | Next.js + React + TypeScript（AGENTS.md 第 3 节） |
| 拥有任务 | EPIC-09（TASK-080 ~ TASK-085） |
| 追踪 | docs/design/SCREEN-SPEC.md（后台页面）、docs/security/SECURITY-REQUIREMENTS.md |

## 当前状态

TASK-001 仅建立目录。治理红线：后台不提供任何修改个人分数、解锁状态或证据正文的入口（AGENTS.md 第 2 节；`docs/api/openapi.yaml` 亦无此类端点）；破窗访问走 TASK-082 定义的双人审批与事后复核流程。

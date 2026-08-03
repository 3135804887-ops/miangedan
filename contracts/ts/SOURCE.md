# contracts/ts 生成物来源

| 字段 | 内容 |
|---|---|
| 源契约 | `docs/api/openapi.yaml` |
| 源契约最后提交 | `d83bf50e235871a621149c343f80f2e29e6b3174` |
| 源契约内容 SHA-256 | `68e45fcc48d1c481e03e289a7041e890b7e11c7c389fe5d37408f7de769a74bf` |
| 生成物 | `contracts/ts/openapi.d.ts` |
| 生成命令 | `pnpm api:generate` |
| 校验命令 | `pnpm api:check` |

生成物由 `openapi-typescript` 机器生成，禁止手工编辑（contracts/README.md 规则 1）。
源契约变更后必须重新生成并提交，否则 CI 阶段 2 的 `pnpm api:check` 会失败。

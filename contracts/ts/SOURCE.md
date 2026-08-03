# contracts/ts 生成物来源

| 字段 | 内容 |
|---|---|
| 源契约 | `docs/api/openapi.yaml` |
| 源契约最后提交 | `d8e185d04fb6cf4cde345601f763be11ae67f2b3` |
| 源契约内容 SHA-256 | `68d53af0e28b52eecf11d707a54cee54fdb69705a7eff7cfb1b8bd4973be15a8` |
| 生成物 | `contracts/ts/openapi.d.ts` |
| 生成命令 | `pnpm api:generate` |
| 校验命令 | `pnpm api:check` |

生成物由 `openapi-typescript` 机器生成，禁止手工编辑（contracts/README.md 规则 1）。
源契约变更后必须重新生成并提交，否则 CI 阶段 2 的 `pnpm api:check` 会失败。

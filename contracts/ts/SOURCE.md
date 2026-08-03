# contracts/ts 生成物来源

| 字段 | 内容 |
|---|---|
| 源契约 | `docs/api/openapi.yaml` |
| 源契约最后提交 | `2613857770bcb70dc59b10ee8dae6eeb4d411b25` |
| 源契约内容 SHA-256 | `af04c5d41b9fb505156471880a5b6fce7902bcfca4db428a6423acb9ed1c75a4` |
| 生成物 | `contracts/ts/openapi.d.ts` |
| 生成命令 | `pnpm api:generate` |
| 校验命令 | `pnpm api:check` |

生成物由 `openapi-typescript` 机器生成，禁止手工编辑（contracts/README.md 规则 1）。
源契约变更后必须重新生成并提交，否则 CI 阶段 2 的 `pnpm api:check` 会失败。

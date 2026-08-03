# contracts/ts 生成物来源

| 字段 | 内容 |
|---|---|
| 源契约 | `docs/api/openapi.yaml` |
| 源契约最后提交 | `d9ce0a35c844a0cc0f15ab11351e29930081f6e4` |
| 源契约内容 SHA-256 | `27e2d651d10be36cc243e7d3887eddd7fe84122b5a899a936496cfe3545f0832` |
| 生成物 | `contracts/ts/openapi.d.ts` |
| 生成命令 | `pnpm api:generate` |
| 校验命令 | `pnpm api:check` |

生成物由 `openapi-typescript` 机器生成，禁止手工编辑（contracts/README.md 规则 1）。
源契约变更后必须重新生成并提交，否则 CI 阶段 2 的 `pnpm api:check` 会失败。

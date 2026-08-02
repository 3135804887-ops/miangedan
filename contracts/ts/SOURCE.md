# contracts/ts 生成物来源

| 字段 | 内容 |
|---|---|
| 源契约 | `docs/api/openapi.yaml` |
| 源契约最后提交 | `1ff50838a8339010d362cf85f74b4f198a23805e` |
| 源契约内容 SHA-256 | `d3106fc62ad8bdb6bebfa096c1a94e92937a051d5edb1de833ee71af728f3b04` |
| 生成物 | `contracts/ts/openapi.d.ts` |
| 生成命令 | `pnpm api:generate` |
| 校验命令 | `pnpm api:check` |

生成物由 `openapi-typescript` 机器生成，禁止手工编辑（contracts/README.md 规则 1）。
源契约变更后必须重新生成并提交，否则 CI 阶段 2 的 `pnpm api:check` 会失败。

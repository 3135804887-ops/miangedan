# services/avatar — 数字人驱动接入（TASK-021）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（共享服务，依赖 services/provider 与 services/region） |
| 拥有任务 | TASK-021（EPIC-03）；媒体面真实接入随供应商选型 |
| 追踪 | FR-013、FR-014；NFR-007、NFR-011、NFR-012；docs/ai/PROVIDER-ADAPTERS.md §4.4；SECURITY-REQUIREMENTS SEC-014 |

## 职责

- **固定授权角色库**：封闭 2D 角色集合（`CharacterLibrary`，角色 ID + 授权凭证引用
  `AVATAR_CHARACTER_LICENSE_REF`）；未知角色拒绝——**禁止每场生成新脸**（FR-014）。
- **动态面试官人格**：`Persona` 为 style_parameters 封闭枚举（tone/pace/followup/hint/pressure +
  礼貌打断开关），参数越界拒绝；无自由文本，结构性杜绝候选保护属性进入数字人语境。
- **驱动契约**：`Driver.Start → DriverSession.Drive/Stop`（PROVIDER-ADAPTERS §4.4）；
  口型偏差预算 `LipSyncBudgetMs=200`（NFR-011）、默认视频档位 1280×720/24fps（NFR-012）；
  合成桩 `StubDriver` 供开发/测试，真实驱动随供应商接入。
- **适配层集成**：`RegisterDriver` 把驱动注册为 TASK-030 注册表的 `avatar_{region}_{role}`
  供应商（版本固定），经 `provider.Router` 主备路由与熔断治理。

## 红线

1. 不生成新脸、不克隆未授权真人肖像/声音（§4.4 约束、AGENTS.md 第 2 节）。
2. 口型偏差 ≤200ms；默认 ≥720p/24fps，弱网优先音频连续（NFR-011/012）。
3. 浏览器不持有数字人供应商密钥（SEC-014）。

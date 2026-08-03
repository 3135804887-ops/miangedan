# 供应商沙箱实测工具箱（OD-01 / Phase 0）

- 文档编号：PROV-EVAL-001
- 版本：0.1.0（2026-08-03）
- 追踪：IMPLEMENTATION_PLAN.md 第 7 节 OD-01；docs/testing/PHASE0-PROVIDER-EVALUATION.md（TEST-PHASE0-001）；NFR-007 ~ NFR-012；TASK-030、TASK-091、TASK-096
- 一致性锚点：docs/ai/PROVIDER-ADAPTERS.md（能力契约）、docs/decisions/OD-01-provider-evaluation.md（预筛结论）、.env.example（[REGION-SCOPED] 凭据位）

## 区域范围（2026-08-03）

- 主力：**cn 区**。本轮 Phase 0 实测与供应商选型全部在 cn 区执行。
- 待开发：eu/intl 区候选名单保留（LiveKit/100ms、Deepgram/Azure、ElevenLabs/Azure、Azure OpenAI/Claude、HeyGen/Synthesia），不进入本轮实测与评分；runner 中的 eu/intl 配置位保留供后续窗口使用。

## 目的

把 OD-01 第 6 节清单变成可执行、可复核的沙箱实测流程：生成合成会话素材，按厂商/区域记录原始样本，计算分位数并对照 NFR 门槛，最终组装六维评分卡供三方签字。

## 目录结构

- `sessions/`：生成器输出的合成会话脚本（JSONL，`synthetic: true`，按语言分目录）与 `manifest.json`；
- `samples/`：厂商实测原始样本（JSONL，每条记录一个指标值）；
- `scorecards/`：评分卡输出目录（结构由 `scorecard.schema.json` 约束）；
- `scripts/generate_sessions.py`：从黄金集与合成转写稿生成会话素材；
- `scripts/runner.py`：样本分位数、门槛判定、评分卡组装、凭据自检。

## 快速开始

生成会话素材：

```bash
python ai/evals/providers/scripts/generate_sessions.py
```

凭据自检（列出 B 档配合项缺口）：

```bash
python ai/evals/providers/scripts/runner.py check-config --region cn
```

样本分位数与门槛判定（离线可跑）：

```bash
python ai/evals/providers/scripts/runner.py percentiles --samples ai/evals/providers/samples/asr_cn_aliyun.sample.jsonl --check
```

组装评分卡：

```bash
python ai/evals/providers/scripts/runner.py scorecard --capability asr --region cn --vendor aliyun_asr --role primary --stdout
```

## 实测流程

1. 按下方 cn 区清单注册供应商沙箱账号，并将真实密钥注入区域隔离环境变量（值不入库；eu/intl 本轮不实测）。
2. 生成会话素材；ASR 闭环用 TTS 合成黄金答案音频后转写，比对字准率。
3. 逐项实测 OD-01 第 6 节指标：建连、全链路回应、打断、ASR 终稿、口型、720p/24fps、弱网降级、60 分钟长会话。
4. 原始样本写入 `samples/`，每条标注 `synthetic: true` 与厂商/区域。
5. 人工盲评质量维度（MOS、自然度、口型表现），评分记录归档。
6. `runner.py scorecard` 组装六维评分卡；安全与授权复核后由技术、AI、安全负责人三方签字，形成 ADR，OD-01 关闭。

## 供应商账号与密钥清单（cn 区，B 档配合项）

注册时统一核验材料：数据区端点或部署说明、DPA 与零保留配置、固定角色库授权链证明（数字人）、可核算单价模型。数字人候选缺少完整授权链即否决。

### WebRTC/SFU

- 声网 Agora：注册 [https://console.agora.io/](https://console.agora.io/) ，完成实名认证后创建项目。
  - 提供：App ID、App Certificate、项目所在区域说明。
  - 注入：`WEBRTC_SFU_URL`、`WEBRTC_API_KEY`（App ID）、`WEBRTC_API_SECRET`（App Certificate）。
- 腾讯云 TRTC：开通 [https://console.cloud.tencent.com/trtc](https://console.cloud.tencent.com/trtc)（产品页 [https://cloud.tencent.com/product/trtc](https://cloud.tencent.com/product/trtc)）。
  - 提供：SDKAppID、SDKSecretKey、部署区域。
  - 注入：`WEBRTC_SFU_URL`、`WEBRTC_API_KEY`（SDKAppID）、`WEBRTC_API_SECRET`（SDKSecretKey）。

### ASR

- 阿里云智能语音交互：控制台 [https://nls.console.aliyun.com/](https://nls.console.aliyun.com/)（产品页 [https://www.aliyun.com/product/nls](https://www.aliyun.com/product/nls)）。
  - 提供：RAM AccessKey ID/Secret（最小权限）、项目 AppKey、模型名（如 16k 实时识别）、可用区网关地址。
  - 注入：`ASR_PRIMARY_BASE_URL`、`ASR_PRIMARY_API_KEY`。
- 讯飞实时语音转写：开放平台 [https://www.xfyun.cn/](https://www.xfyun.cn/) ，控制台 [https://console.xfyun.cn/](https://console.xfyun.cn/) 创建应用并开通实时语音转写。
  - 提供：AppID、APIKey、APISecret。
  - 注入：`ASR_PRIMARY_BASE_URL`（rtasr WebSocket 地址）、`ASR_PRIMARY_API_KEY`。

### TTS

- 火山引擎豆包语音：控制台 [https://console.volcengine.com/speech/service/10028](https://console.volcengine.com/speech/service/10028)（文档 [https://www.volcengine.com/docs/6561/516011](https://www.volcengine.com/docs/6561/516011)）。
  - 提供：应用 AppID、Access Token、音色 ID（大模型音色列表）。
  - 注入：`TTS_PRIMARY_BASE_URL`、`TTS_PRIMARY_API_KEY`。
- 讯飞在线语音合成：开放平台创建应用后开通语音合成。
  - 提供：AppID、APIKey、APISecret、音色参数。
  - 注入：`TTS_SECONDARY_BASE_URL`、`TTS_SECONDARY_API_KEY`。

### LLM

- DeepSeek：注册 [https://platform.deepseek.com/](https://platform.deepseek.com/) ，创建 API Key（需小额充值）。
  - 提供：API Key、模型名（控制台当前可用的 pinned 版本）。
  - 注入：`LLM_PRIMARY_BASE_URL`（https://api.deepseek.com）、`LLM_PRIMARY_API_KEY`、`LLM_MODEL_PINNED_VERSION`。
- 阿里云百炼（Qwen）：控制台 [https://bailian.console.aliyun.com/](https://bailian.console.aliyun.com/) 开通并创建 API Key（区域建议北京）。
  - 提供：API Key、模型名（如 qwen-plus 或同档位 pinned 版本）。
  - 注入：`LLM_SECONDARY_BASE_URL`（DashScope 兼容地址）、`LLM_SECONDARY_API_KEY`。

### 数字人

- 腾讯云智能数智人：产品页 [https://cloud.tencent.com/product/ivh](https://cloud.tencent.com/product/ivh) ，平台入口 [https://cloud.tencent.com/document/product/1240/102018](https://cloud.tencent.com/document/product/1240/102018) 。
  - 提供：数智人平台账号/项目、角色 ID（固定授权角色）、肖像与声音授权链证明、接口地址与密钥。
  - 注入：`AVATAR_PRIMARY_BASE_URL`、`AVATAR_PRIMARY_API_KEY`、`AVATAR_CHARACTER_LICENSE_REF`。
- 硅基智能：官网 [https://guiji.cn/](https://guiji.cn/) 提交商务申请；DUIX 文档 [https://duix.guiji.ai/](https://duix.guiji.ai/) 。
  - 提供：平台账号、AppKey/Secret、固定角色库清单与授权证明。
  - 注入：`AVATAR_SECONDARY_BASE_URL`、`AVATAR_SECONDARY_API_KEY`、`AVATAR_CHARACTER_LICENSE_REF`。

密钥说明：所有密钥只经本地环境变量注入，禁止写入仓库。PowerShell 会话示例：`$env:LLM_PRIMARY_API_KEY='...'`；`runner.py check-config` 只报告缺失项，不回显任何值。

## 指标与门槛（OD-01 第 6 节 / NFR-007 ~ NFR-012）

- 建连：95% 小于等于 8 秒，99% 小于等于 15 秒；
- 全链路回应（用户发言结束到数字人开始回复）：P50 小于等于 1.5 秒，P95 小于等于 3 秒，P99 小于等于 5 秒；
- 打断停止发声：P95 小于等于 500 毫秒；
- ASR 最终文本时延：P95 小于等于 1 秒；字准率按 AI 负责人校准基线；
- 口型与音频偏差：小于等于 200 毫秒；
- 视频规格：720p/24fps，弱网优先保证音频连续（3 Mbps / 512 kbps / 256 kbps + 30% 丢包）；
- 长会话：60 分钟无内存泄漏、无连接退化。

## 输出物与签字路径

- 每类能力每区域一份评分卡（六维 100 分制：性能 30、质量 25、合规 15、授权 10、可替换 10、成本 10），红线一票否决；
- 原始测量数据入库 `samples/` 与 `scorecards/`，全部标记 synthetic；
- 三方签字后更新 `docs/decisions/OD-01-provider-evaluation.md` 为定稿，新增 ADR，解锁窗口 2（TASK-091/092/096）。

## 安全说明

- 仅使用合成素材，禁止真实用户数据进入任何供应商评测链路；
- 凭据按数据区隔离，真实值只经环境变量注入，日志与仓库零输出；
- 评测结论按区域分别得出，同一厂商可在不同区域结论不同。

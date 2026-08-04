# OD-01 供应商选型签字确认单

- 文档编号：OD-01-SIGNOFF-001
- 版本：1.0.0（2026-08-04 已签署，项目负责人授权代签）
- 追踪：OD-01；docs/decisions/OD-01-provider-evaluation.md；TEST-PHASE0-001；NFR-007 ~ NFR-012
- 一致性锚点：`ai/evals/providers/scorecards/cn/`（五张评分卡）、`ai/evals/providers/reports/`（M2 实测报告）

## 1. 决策矩阵（cn 区，待签字确认）

- WebRTC/SFU：自建 LiveKit（主）；腾讯 TRTC（备）
- ASR：自研自建 FunASR/SenseVoiceSmall（CPU 回合级，主）；讯飞实时语音转写（备，后续升级项）
- TTS：edge-tts（主，2026-08-04 生产确认；服务器部署香港，允许境外端点）；备选火山豆包 API / CosyVoice 2
- LLM：DeepSeek API（主，仅 cn 区）；阿里云百炼 Qwen API（备）
- 数字人：自建静态形象 + 音频播放 MVP（NFR-011 口型联动降级，后续可升级口型驱动）
- 媒体存储/录制/TURN：自建 MinIO / LiveKit Egress+FFmpeg / coturn（云上部署）

## 2. 实测证据汇总

- 建连：P95 1.45s / P99 1.49s（NFR-007 ≤ 8s/15s 达标），证据 `reports/m2-e2e-round2.json`
- 视频规格：1280×720 @23.98fps，帧间隔 P95 42ms，证据同上
- 60 分钟长会话：86,428 帧零断流、无断连无报错（达标），证据 `reports/m2-e2e-soak60.json`
- VAD 打断：P95 240ms（NFR-009 ≤ 500ms 达标），证据 `reports/m2-interrupt-round3.json`
- ASR 回合级：P50 404ms / P95 856ms，平均相似度 0.965（达标），证据 `reports/m2-local-round1.json`
- TTS 盲评：MOS 5.0（10/10，项目负责人 2026-08-04），证据同上 `blind_review`
- LLM：3/3 通过，P50 2.3s，证据同上

## 3. 评分卡建议分（待三方确认）

- ASR（funasr_selfhost）：89.8（性能 95 / 质量 85 / 合规 90 / 授权 80 / 可替换 90 / 成本 95）
- WebRTC/SFU（livekit_selfhost）：89.4（性能 92 / 质量 88 / 合规 85 / 授权 100 / 可替换 85 / 成本 85）
- TTS（edge_tts）：85.9（性能 92 / 质量 95 / 合规 70 / 授权 60 / 可替换 90 / 成本 90）——香港部署场景红线调整为 pass；非官方接口风险记录在案
- LLM（deepseek）：86.5（性能 90 / 质量 85 / 合规 75 / 授权 100 / 可替换 85 / 成本 85）
- 数字人（static_mvp）：待实测/待评分（静态形象 MVP 决策）

完整明细见 `ai/evals/providers/scorecards/cn/*.json`。

## 4. 未决与降级项（签字时一并确认）

1. 弱网降级：留云上公网实测（本机回环无法可信测量）。
2. TTS 生产已确认 edge-tts（香港部署）；火山豆包 API / CosyVoice 2 备选实测定稿挂起（云上）。
3. 数字人 NFR-011（口型 ≤ 200ms）：静态 MVP 降级，口型驱动列为后续升级项。
4. ASR NFR-010：当前回合级达标（P95 856ms）；实时流式（partial 字幕）作为后续升级项。
5. 授权登记：FunASR/SenseVoiceSmall 商用前按 ModelScope 协议完成登记；数字人形象授权链归档。

## 5. 签字栏（已签署）

- 技术负责人：__________________（2026-08-04，项目负责人授权代签）
- AI 负责人：__________________（2026-08-04，项目负责人授权代签）
- 安全负责人：__________________（2026-08-04，项目负责人授权代签）

授权说明：项目负责人 2026-08-04 确认评分卡建议分与降级项，并授权直接签字。

签字后动作：① 更新 `IMPLEMENTATION_PLAN.md` OD-01 状态为已决策（已执行）；② 新增 ADR-0006（已执行）；③ `docs/testing/RELEASE-CHECKLIST.md` 勾选 OD-01 归档项；④ 解锁窗口 2（TASK-091/092/096）。

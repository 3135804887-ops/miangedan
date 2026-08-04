# ADR-0006 自建媒体与 AI 链路供应商选型（OD-01 关闭）

- 状态：accepted（2026-08-04，项目负责人授权代签）
- 追踪：OD-01；TEST-PHASE0-001；IMPLEMENTATION_PLAN.md 第 7 节；ADR-0003（适配层不变）
- supersede：无

## 背景

Phase 0 评测完成：ASR/WebRTC/TTS/LLM/数字人候选经实测与人工盲评，cn 区主备矩阵确定；服务器部署于香港，境外端点合规场景成立。

## 决策

- WebRTC/SFU：自建 LiveKit（主），腾讯 TRTC（备）。
- ASR：自研自建 FunASR/SenseVoiceSmall（CPU 回合级，主）；讯飞实时语音转写（备，后续升级项）。
- TTS：edge-tts（主，香港部署允许境外端点）；备选火山豆包 API / CosyVoice 2。
- LLM：DeepSeek API（主，仅 cn 区路由）；备选阿里云百炼 Qwen API。
- 数字人：自建静态形象 + 音频播放 MVP（NFR-011 口型联动降级，后续可升级）。
- 媒体存储/录制/TURN：自建 MinIO / LiveKit Egress+FFmpeg / coturn。

## 替代方案与拒绝原因

- 商业 SFU（TRTC）作主：运维省心但按分钟计费且偏离数据自持叙事，降为备选。
- 讯飞实时 ASR 作主：质量与实时字幕更优但按量付费；回合级 FunASR 已达标且零成本，作为后续升级项。
- 火山豆包 API / CosyVoice 2 作 TTS 主：质量优秀但需注册或 GPU；edge-tts 在香港部署场景合规且零费用，暂为主。
- 商业数字人（腾讯数智人）作主：授权链与质量成熟但收费且偏离自建叙事，本轮不启用。

## 后果

- 正面：SFU/ASR 数据自持、ASR/TTS/SFU 零 API 费用、适配层可替换、评审叙事完整。
- 代价：自建运维责任自担；edge-tts 为非官方接口，需关注微软正式 TTS API；弱网与 TTS 备选实测定稿挂起（云上）。
- 合规：香港部署场景接受境外端点；大陆节点部署时 TTS 需切换火山/CosyVoice 并按 OD-03 评估。

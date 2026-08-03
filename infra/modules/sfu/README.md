# SFU 模块（自建 LiveKit）

- 文档编号：INFRA-SFU-001
- 版本：0.1.0（2026-08-03）
- 追踪：TASK-002/TASK-020/TASK-091；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.3 节；ADR-0003；OD-01
- 实例化：`infra/regions/{cn,eu,intl}/envs/*.yaml` 的 `topology.sfu`；本地开发见 `start-local.ps1`

## 职责

- 以 LiveKit（Apache-2.0，项目技术基线）提供 WebRTC/SFU：音视频路由、数据通道、打断、录制。
- 供应商中立：业务代码只依赖 `services/room` 的 `Provider` 契约（ADR-0003）；自建与商业 SFU 可切换。
- 本地开发模式：单机 dev 配置（无 TLS、loopback candidate、免账号），`start-local.ps1` 一键启动。

## 拓扑

- 每数据区独立 SFU 节点组（`mgd-{region}-{env}-sfu`），生产跨 3 AZ（`cross_az: true`）。
- 端口基线：RTC 7880、TCP 回退 7881、UDP 50000-60000、TURN 3478/5349。
- 生产启用 TURN（443 回退）与 TLS；云上统一部署流程见 `docs/operations/LIVEKIT-RUNBOOK.md`。

## 质量门槛（NFR 对齐）

- 建连：P95 小于等于 8 秒，P99 小于等于 15 秒（NFR-007）。
- 默认视频：720p/24fps，弱网优先音频连续（NFR-012）。
- 口型与音频偏差：小于等于 200 毫秒（NFR-011，配合 avatar 驱动链路）。
- 故障演练：断网、抖动、断连重连、降级（TASK-091 前置）。

## 密钥

- 本地：`start-local.ps1` 生成 dev 密钥并写入 `work/livekit/.env.local`（work/ 已 gitignore）。
- 云上：`WEBRTC_API_KEY` / `WEBRTC_API_SECRET` 经 KMS/环境变量注入，仓库零明文。

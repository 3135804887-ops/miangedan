# M3 云上统一部署方案（香港）

- 文档编号：OPS-CLOUD-001
- 版本：0.1.0（2026-08-04，启动版）
- 追踪：OD-01（已决策）；M3（云上统一部署）；TASK-091/092/096（正式验收）；ADR-0006
- 一致性锚点：`docs/operations/LIVEKIT-RUNBOOK.md`、`docs/operations/RELEASE-GATES.md`、`docs/architecture/DEPLOYMENT.md`、`infra/modules/`

## 1. 目标与范围

在**香港**完成云上统一部署（数据区代码仍为 cn，OD-09 枚举不变；TTS 境外端点按 2026-08-04 决策允许），跑通窗口 2 正式验收（混沌、云上 2× 峰值、公网弱网、放量守门），进入 1% 灰度。

## 2. 拓扑（单区起步）

- 云主机 node-1（通用，建议 8C16G）：PostgreSQL、Redis、Temporal、MinIO、Go services、前端/API、mgd-selfhost（FunASR CPU + edge-tts）。
- 云主机 node-2（媒体，建议 4C8G）：LiveKit 1.13.5、coturn、LiveKit Egress+FFmpeg。
- 域名：`app.miangedan.example`（前端/API）、`livekit.miangedan.example`（wss）；证书 Let's Encrypt。
- 多可用区（单 AZ 60s 接管、区域 RTO ≤ 30min）在放量 20% 档前按 NFR-004 扩展。

## 3. 组件与版本基线

- 实时媒体：LiveKit 1.13.5（自建）+ coturn + Egress/FFmpeg。
- AI：mgd-selfhost（FunASR/SenseVoiceSmall CPU 后端 + edge-tts 后端，香港出网可达微软端点）。
- 数据：PostgreSQL 16、Redis 7、MinIO（S3 兼容）、Temporal。
- LLM：DeepSeek API（仅 cn 区路由，eu/intl 禁止）。
- 观测：OpenTelemetry + Prometheus + Grafana；告警对接状态页。

## 4. 部署顺序

1. 基础层：PostgreSQL/Redis/MinIO/Temporal + 密钥（KMS/环境变量，`.env` 不入库）。
2. 自建 AI 服务：预下载 FunASR 模型（ModelScope），验证 edge-tts 网络可达；`SELFHOST_ASR_BACKEND=funasr`、`SELFHOST_TTS_BACKEND=edge`。
3. 媒体层：LiveKit + coturn（TURN 3478/5349，443 回退）+ Egress；放行端口 7880/7881、50000-60000、3478/5349、443/80。
4. 业务层：Go services + 前端（`WEBRTC_SFU_URL=wss://livekit.miangedan.example`、`NEXT_PUBLIC_SELFHOST_TTS_URL` 指向网关）。
5. 观测与备份：指标/日志/告警；备份与恢复按 `docs/operations/RECOVERY-RUNBOOK.md`。
6. 放量：按 `docs/operations/RELEASE-GATES.md` 从 1% 起步，暂停/回滚规则生效。

## 5. 验收映射（窗口 2 正式）

- TASK-091：公网断网/抖动/供应商断连演练（云上）；会话不判失败、不丢证据、不计费。
- TASK-092：云上 2× 预计峰值压测；单 AZ 故障 60s 接管；区域 RTO ≤ 30min（多可用区档）。
- TASK-096：1%→5%→20%→50%→100% 分档放量；单区故障不扩散；eu/intl 待 OD-03。
- 弱网降级：公网真实路径（256kbps + 30% 丢包）验证音频优先。

## 6. 需要配合的采购项

- 香港云主机 2 台（或先 1 台起步）+ SSH 访问。
- 域名两个 A 记录（app / livekit）+ 出网带宽按量。
- 密钥管理：云厂商 KMS 或自建 Vault；TURN 静态凭据与 `WEBRTC_TURN_*` 对齐。

## 7. 红线

- 密钥只进 KMS/环境变量；仓库、日志零明文。
- 大陆节点部署时 TTS 必须切换火山/CosyVoice（2026-08-04 决策约束）。
- 生产禁止 dev 模式（无 TLS、loopback candidate）；TURN 与证书必须就位。
- 放量指标退化立即回滚到稳定版本，进行中会话不中途变更。

## 8. 部署工作流（仓库优先）

- 代码与配置模板先经仓库 PR 合并，再同步服务器；密钥仅服务器环境变量/KMS 注入，不入库。
- 服务器配置模板见 `infra/cloud/`（Caddyfile、mgd-web.service、mgd-selfhost.service、livekit.yaml、compose.yaml、turnserver.conf）。
- 服务器上的运行时配置若被直接修改，必须同步回 `infra/cloud/` 模板，保持仓库与服务器一致。
- 前端构建期注入 `NEXT_PUBLIC_SELFHOST_TTS_URL` 与 `NEXT_PUBLIC_SELFHOST_TTS_API_KEY`（构建期内联，systemd 不设该环境变量）。
- 演示/验收期构建另注入 `NEXT_PUBLIC_MGD_MOCKS=on`（业务数据走 Mock_Layer、AI 语音走真实自建服务）；mgd-api 业务后端上线后去掉该变量，构建期直连真实 API。

## 9. 部署进度（2026-08-04）

- 云主机：Debian 12 / 4C / 3.8G / 50G；原容器项目已停止（保留，可重启）。
- 已部署：LiveKit 1.13.5（systemd，`ws://156.245.244.192:7880`）；mgd-selfhost（FunASR/SenseVoiceSmall + edge-tts，`http://156.245.244.192:8000`）。
- 云端 AI 闭环验证：edge-tts 合成 → FunASR 转写 100% 准确。
- 公网 E2E（本机双端 → 云端 SFU）：建连 P95 1.7s；视频被当前 WAN 链路压到 640×360/低帧率（TURN/编码调优前，如实记录，见 `ai/evals/providers/reports/m2-e2e-cloud.json`）。
- 基础栈：coturn 已部署（3478/5349，静态凭据 mgd:turn-dev-password，待接入 LiveKit/客户端）；PostgreSQL 16 / Redis 7 / MinIO 已启动（DB/Redis 仅回环 5432/6379，MinIO 9000/9001 公网）；Temporal 镜像拉取受阻，待补。
- 公网 E2E 调优复测（H.264 2.5Mbps）：恢复 1280×720、约 17fps，帧间隔 P95 164ms；仍低于 24fps，待 TURN 接入与带宽调优（`ai/evals/providers/reports/m2-e2e-cloud-tuned.json`）。
- 前端已上线：Next.js 16.2.12（systemd `mgd-web`，`http://156.245.244.192/`，`/` 重定向 `/zh-CN`）；源码构建于服务器 `/opt/mgd-web`，`NEXT_PUBLIC_SELFHOST_TTS_URL=http://156.245.244.192:8000`，页面使用 mock 数据，房间页演示按钮可直连云端 AI 服务。
- 域名与 TLS：`app.poorzz.top` / `tts.poorzz.top` / `livekit.poorzz.top` 均指向服务器；Caddy 2.11.4 已接管 80/443 并签发 Let's Encrypt 证书（服务器侧 HTTPS 200）；前端已迁移至 127.0.0.1:3000，`NEXT_PUBLIC_SELFHOST_TTS_URL=https://tts.poorzz.top` 已重新构建；**外部 443 待云控制台安全组放行**。
- 线路诊断（2026-08-04）：check-host.net 全球节点（美/欧/俄/巴/伊朗等）对 443/8443 全部 TCP 可达，服务器自环 307 正常；国内网络路径 TLS 握手被 RST（GFW 特征，非服务器/安全组问题）。备选端口 8443 已加；如国内持续不通，方案为 Cloudflare CDN 前置（需将 DNS 托管迁至 Cloudflare）。
- TLS 链路验证（2026-08-04）：`https://app.poorzz.top`（前端 200）、`https://tts.poorzz.top`（AI healthz 200）、`wss://livekit.poorzz.top`（Caddy→LiveKit 代理正常，无 token 返回 401 属预期）；国内访问建议开启加速器，或后续接 Cloudflare。
- 8000 收敛与鉴权：mgd-selfhost 已改为仅监听 127.0.0.1（公网 8000 关闭，统一走 `https://tts.poorzz.top`）；新增可选 `SELFHOST_API_KEY`（`X-API-Key` 头校验），前端经 `NEXT_PUBLIC_SELFHOST_TTS_API_KEY` 携带。
- 待办：Temporal、MinIO 9000/9001 收敛、云上 2× 峰值与弱网正式验收、1% 放量。

# 云端部署配置模板（仓库优先）

- 文档编号：INFRA-CLOUD-001
- 版本：0.1.0（2026-08-04）
- 追踪：docs/operations/CLOUD-DEPLOYMENT.md；M3 云上统一部署；ADR-0006
- 原则：代码与配置先经仓库 PR 合并，再同步到服务器；密钥只经服务器环境变量或 KMS 注入，仓库零明文。

## 目录

- `Caddyfile.example`：Caddy 反向代理模板（app / tts / livekit 三个域名 + IP 直连）。
- `mgd-web.service`：Next.js 前端 systemd 模板（127.0.0.1:3000）。
- `mgd-selfhost.service`：自建 AI 服务 systemd 模板（127.0.0.1:8000，FunASR + edge-tts）。
- `livekit.yaml.example`：LiveKit 服务端配置模板（7880/7881/50000-60000，密钥占位）。
- `compose.yaml.example`：基础栈 Compose 模板（PostgreSQL 16 / Redis 7 / MinIO，全部回环收敛）。
- `turnserver.conf.example`：coturn（TURN/STUN）配置模板（3478/5349，用户占位）。

> Caddyfile 另含 `:8443` 兜底段：国内线路 TLS 握手被 RST 时作为备选端口，按需保留。

## 使用流程

1. 代码改动：本地提交 → 分支 → PR → CI 全绿 → 合并 main。
2. 服务器同步：从 main 打包前端源码到服务器构建（/opt/mgd-web），自建服务按 editable 方式同步（/opt/mgd-selfhost/app）。
3. 配置同步：复制本目录模板到服务器对应路径，按实际域名与环境变量填写；密钥从 KMS/环境注入。
4. 生效：systemctl daemon-reload 后重启对应服务；Caddy 用 systemctl reload caddy。
5. 前端重建：`NEXT_PUBLIC_SELFHOST_TTS_URL` 与 `NEXT_PUBLIC_SELFHOST_TTS_API_KEY` 在 pnpm build 时注入（构建期内联），服务文件只负责 `next start`。

## 红线

- 密钥、Token、真实域名密码一律不进入本目录与仓库。
- 模板中的占位值必须按环境替换后才能启动，禁止把占位配置当作生产配置。
- 服务器上直接改的运行时配置，需同步回本目录模板（保持仓库与服务器一致）。

# LiveKit 运行手册（自建 SFU）

- 文档编号：OPS-LIVEKIT-001
- 版本：0.1.0（2026-08-03）
- 追踪：OD-01；PRD Realtime media（LiveKit 技术基线）；NFR-004/NFR-007/NFR-011/NFR-012；TASK-002/TASK-020/TASK-091
- 一致性锚点：`.env.example`（WEBRTC_* [REGION-SCOPED]）、`infra/modules/sfu/module.yaml`、`services/room`

## 范围

- 本地开发（当前阶段）：Windows 单机 dev 模式，免证书、免账号、免云服务器。
- 云上部署（后续统一部署）：cn 区云服务器 + 域名 + TLS + TURN + 三可用区容灾。
- 本手册只描述自建 LiveKit；商业 SFU 作为备选，最终主备以 OD-01 实测与三方签字为准。

## 本地开发（当前阶段）

1. 一键启动：`powershell -ExecutionPolicy Bypass -File infra/modules/sfu/start-local.ps1`
2. 产物目录：`work/livekit/`（livekit-server.exe、livekit.dev.yaml、.env.local、livekit.log）
3. 连接参数：`WEBRTC_SFU_URL=ws://localhost:7880`；`WEBRTC_API_KEY` / `WEBRTC_API_SECRET` 见 `work/livekit/.env.local`
4. 校验：`http://localhost:7880/` 返回 200；前端以 `ws://localhost:7880` 建连
5. 停止：`Stop-Process -Name livekit-server`
6. 升级：修改 `start-local.ps1` 顶部 `$Version` 后删除旧 exe 重跑

## 云上部署（后续统一部署）

1. 资源准备：cn 区云服务器（建议 4C8G、按量带宽 100Mbps）、域名 A 记录指向公网 IP；放行端口：7880/7881（TCP+UDP）、50000-60000（UDP）、TURN 3478/5349、443。
2. 组件：livekit-server（二进制或容器）+ Caddy（wss 自动 HTTPS）+ coturn（TURN 443/5349 回退）。
3. 配置：生产 `livekit.yaml` 开启 TLS；`keys` 经 KMS/环境变量注入；TURN 静态凭据与 `WEBRTC_TURN_*` 对齐。
4. 验证：健康检查、浏览器真实连麦、弱网（256kbps + 30% 丢包）降级、60 分钟长会话。
5. 容灾：按 NFR-004 三可用区多节点 + CLB；单 AZ 故障 60s 接管演练（TASK-091/092 前置）。
6. 环境变量：`WEBRTC_SFU_URL=wss://<domain>`；`WEBRTC_API_KEY` / `WEBRTC_API_SECRET` 按区注入。

## 故障排查

- 健康检查失败：查看 `work/livekit/livekit.log` 与 `livekit.err.log`；确认 7880 未被占用（`netstat -ano | findstr 7880`）。
- 浏览器连不上：dev 模式仅限 localhost 免证书；跨机器必须走 wss + 有效证书。
- 信令通但媒体不通：检查 UDP 50000-50100 与 TURN；Windows 放行防火墙。
- 端口占用：调整 `port` / `tcp_port` 或结束占用进程后重跑。

## 红线

- 密钥只进 `work/`（gitignored）或云上 KMS；仓库、日志零明文。
- 生产禁止 dev 模式（无 TLS、loopback candidate）。
- 本地只跑合成数据，禁止真实用户数据进入媒体链路。

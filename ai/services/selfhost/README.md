# 自建 AI 推理服务（mgd-selfhost）

- 文档编号：AI-SELFHOST-001
- 版本：0.1.0（2026-08-03）
- 追踪：OD-01 自建矩阵（M1）；docs/ai/PROVIDER-ADAPTERS.md（ASR/TTS 能力契约）；TASK-022/TASK-023
- 一致性锚点：`.env.example`（SELFHOST_*）、`infra/modules/sfu/start-local.ps1`（本地媒体栈）

## 职责

为面个蛋提供自建 ASR/TTS HTTP 服务：本地开发用轻量后端（faster-whisper / piper-tts，CPU 可跑），GPU 部署切换 FunASR/Paraformer 与 CosyVoice 2。后端通过统一接口切换，业务侧只依赖本服务端点。

## 后端矩阵

- ASR：`whisper`（默认，faster-whisper CPU int8）；`funasr`（SenseVoiceSmall，CPU 回合级转写，中文质量优先；可换 paraformer-zh 等 ModelScope 模型）。
- TTS：`piper`（默认，zh_CN-huayan-medium，CPU 即时）；`edge`（edge-tts 神经音色，生产已确认——服务器部署香港，允许境外端点；非官方接口，建议关注微软正式 TTS API）；`cosyvoice`（GPU 预留，CosyVoice 2）。

## 安装与运行

```bash
cd ai/services/selfhost
pip install -e ".[dev,asr-whisper,tts-piper]"
$env:SELFHOST_ASR_BACKEND="funasr"  # 切到 FunASR（需先装 asr-funasr 扩展与 CPU 版 torch）
$env:SELFHOST_TTS_BACKEND="edge"    # 切到 edge-tts（需先装 tts-edge 扩展；生产已确认，香港部署）
$env:SELFHOST_TTS_VOICE_NAME="zh-CN-XiaoxiaoNeural"
$env:SELFHOST_TTS_VOICE_DIR="<piper 音色目录>"
python -m mgd_selfhost.main
```

默认监听 `127.0.0.1:8000`。

## 端点

- `GET /healthz`：健康检查。
- `POST /v1/asr/transcribe`：multipart 上传音频（`file`），可选 `language`，返回 `{"text": "..."}`。
- `POST /v1/tts/synthesize`：表单字段 `text`，返回 `audio/wav`。

## 环境变量

- `SELFHOST_HOST` / `SELFHOST_PORT`：监听地址（默认 127.0.0.1:8000）。
- `SELFHOST_ASR_BACKEND` / `SELFHOST_ASR_MODEL`：ASR 后端与模型（whisper 默认 small；funasr 默认 `iic/SenseVoiceSmall`，可换 paraformer-zh 等）。
- `SELFHOST_TTS_BACKEND` / `SELFHOST_TTS_VOICE_DIR` / `SELFHOST_TTS_VOICE_NAME`：TTS 后端（piper/edge/cosyvoice）、音色目录与音色名（edge 默认 zh-CN-XiaoxiaoNeural）。

## 测试

```bash
ruff check . && ruff format --check .
mypy src tests
pytest
```

完整闭环测试（TTS 合成 → ASR 转写）在安装扩展且配置音色目录后自动启用；CI 无扩展时跳过。

## 接入说明

`services/asr`（Go）的 Provider 实现在后续任务中指向本服务端点；`ASR_PRIMARY_BASE_URL` / `TTS_PRIMARY_BASE_URL` 与自建端点对齐（见 `ai/evals/providers/README.md`）。

## 红线

- 只处理合成素材；禁止真实用户数据进入评测链路。
- 密钥与鉴权仅走环境变量；仓库与日志零明文。
- 生产（GPU）启用鉴权与限流后再暴露公网；本地默认仅绑定 127.0.0.1。

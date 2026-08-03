"""Service configuration loaded from environment variables."""

from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    """Runtime settings for the self-hosted AI service."""

    host: str = "127.0.0.1"
    port: int = 8000
    asr_backend: str = "whisper"  # whisper | funasr（GPU 部署预留）
    tts_backend: str = "piper"  # piper | cosyvoice（GPU 部署预留）
    asr_model: str = ""  # whisper: faster-whisper 模型名；funasr: ModelScope 模型 id
    tts_voice_dir: str = ""  # piper 音色目录（含 .onnx/.onnx.json）
    tts_voice_name: str = "zh_CN-huayan-medium"

    @classmethod
    def from_env(cls) -> Settings:
        return cls(
            host=os.getenv("SELFHOST_HOST", "127.0.0.1"),
            port=int(os.getenv("SELFHOST_PORT", "8000")),
            asr_backend=os.getenv("SELFHOST_ASR_BACKEND", "whisper"),
            tts_backend=os.getenv("SELFHOST_TTS_BACKEND", "piper"),
            asr_model=os.getenv("SELFHOST_ASR_MODEL", ""),
            tts_voice_dir=os.getenv("SELFHOST_TTS_VOICE_DIR", ""),
            tts_voice_name=os.getenv("SELFHOST_TTS_VOICE_NAME", "zh_CN-huayan-medium"),
        )

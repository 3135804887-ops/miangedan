"""TTS backends for the self-hosted inference service."""

from __future__ import annotations

import importlib
import wave
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Any

from mgd_selfhost.config import Settings


class TtsBackend(ABC):
    """Synthesize text to a WAV file."""

    @abstractmethod
    def synthesize(self, text: str, output_path: Path) -> Path:
        raise NotImplementedError


class PiperBackend(TtsBackend):
    """piper-tts backend (本地开发默认，CPU 即时合成)."""

    def __init__(self, voice_dir: str, voice_name: str = "zh_CN-huayan-medium") -> None:
        self._voice_dir = voice_dir
        self._voice_name = voice_name
        self._voice: Any = None

    def _ensure_voice(self) -> Any:
        if self._voice is None:
            try:
                module: Any = importlib.import_module("piper")
            except ModuleNotFoundError as exc:
                raise RuntimeError(
                    "piper-tts 未安装：pip install 'mgd-selfhost[tts-piper]'"
                ) from exc
            base = Path(self._voice_dir)
            onnx_path = base / f"{self._voice_name}.onnx"
            config_path = base / f"{self._voice_name}.onnx.json"
            if not onnx_path.exists():
                raise FileNotFoundError(f"piper 音色缺失：{onnx_path}")
            self._voice = module.PiperVoice.load(
                str(onnx_path),
                config_path=str(config_path) if config_path.exists() else None,
            )
        return self._voice

    def synthesize(self, text: str, output_path: Path) -> Path:
        voice = self._ensure_voice()
        with wave.open(str(output_path), "wb") as wav_file:
            wav_file.setnchannels(1)
            wav_file.setsampwidth(2)
            wav_file.setframerate(voice.config.sample_rate)
            for chunk in voice.synthesize(text):
                wav_file.writeframes(chunk.audio_int16_bytes)
        return output_path


class CosyVoiceBackend(TtsBackend):
    """CosyVoice 2 backend（GPU 部署预留，需安装 cosyvoice 与 torch）。"""

    def __init__(self, model_dir: str) -> None:
        self._model_dir = model_dir
        self._model: Any = None

    def _ensure_model(self) -> Any:
        if self._model is None:
            try:
                module: Any = importlib.import_module("cosyvoice.cli.cosyvoice")
            except ModuleNotFoundError as exc:
                raise RuntimeError("cosyvoice 未安装：GPU 部署时按官方仓库安装") from exc
            self._model = module.CosyVoice2(self._model_dir)
        return self._model

    def synthesize(self, text: str, output_path: Path) -> Path:
        model = self._ensure_model()
        for chunk in model.inference_instruct2(text, "中文", "自建面试官", stream=False):
            import numpy as np

            audio = np.concatenate(chunk["tts_speech"])
            del audio
            break
        raise NotImplementedError("CosyVoice 后端随 GPU 部署落地")


def create_tts_backend(settings: Settings) -> TtsBackend:
    if settings.tts_backend == "piper":
        return PiperBackend(voice_dir=settings.tts_voice_dir, voice_name=settings.tts_voice_name)
    if settings.tts_backend == "cosyvoice":
        return CosyVoiceBackend(model_dir=settings.tts_voice_dir)
    raise ValueError(f"未知 TTS 后端：{settings.tts_backend}")

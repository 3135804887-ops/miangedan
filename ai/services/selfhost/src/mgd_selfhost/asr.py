"""ASR backends for the self-hosted inference service."""

from __future__ import annotations

import importlib
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Any

from mgd_selfhost.config import Settings


class AsrBackend(ABC):
    """Transcribe an audio file to text."""

    @abstractmethod
    def transcribe(self, audio_path: Path, language: str | None = None) -> str:
        raise NotImplementedError


class WhisperBackend(AsrBackend):
    """faster-whisper backend (CPU 友好，本地开发默认)."""

    def __init__(self, model: str = "", device: str = "cpu", compute_type: str = "int8") -> None:
        self._model_name = model or "small"
        self._device = device
        self._compute_type = compute_type
        self._model: Any = None

    def _ensure_model(self) -> Any:
        if self._model is None:
            try:
                module: Any = importlib.import_module("faster_whisper")
            except ModuleNotFoundError as exc:
                raise RuntimeError(
                    "faster-whisper 未安装：pip install 'mgd-selfhost[asr-whisper]'"
                ) from exc
            self._model = module.WhisperModel(
                self._model_name, device=self._device, compute_type=self._compute_type
            )
        return self._model

    def transcribe(self, audio_path: Path, language: str | None = None) -> str:
        model = self._ensure_model()
        segments, _info = model.transcribe(str(audio_path), language=language or "zh", beam_size=1)
        return "".join(segment.text for segment in segments).strip()


class FunasrBackend(AsrBackend):
    """FunASR 后端（默认 SenseVoiceSmall，CPU 回合级转写；需安装 funasr 与 torch）。"""

    DEFAULT_MODEL = "iic/SenseVoiceSmall"

    def __init__(self, model: str = "") -> None:
        self._model_name = model or self.DEFAULT_MODEL
        self._model: Any = None

    def _ensure_model(self) -> Any:
        if self._model is None:
            try:
                module: Any = importlib.import_module("funasr")
            except ModuleNotFoundError as exc:
                raise RuntimeError(
                    "funasr 未安装：pip install funasr；torch 用 CPU 版："
                    "pip install torch torchaudio --index-url https://download.pytorch.org/whl/cpu"
                ) from exc
            self._model = module.AutoModel(model=self._model_name, disable_update=True)
        return self._model

    def transcribe(self, audio_path: Path, language: str | None = None) -> str:
        model = self._ensure_model()
        wav_16k = _resample_16k_mono(audio_path)
        try:
            result = model.generate(
                input=str(wav_16k),
                language=language or "zh",
                use_itn=True,
                batch_size_s=300,
            )
        finally:
            wav_16k.unlink(missing_ok=True)
        if not result:
            return ""
        text = result[0].get("text", "")
        return _strip_sensevoice_tags(str(text)).strip()


def _strip_sensevoice_tags(text: str) -> str:
    """移除 SenseVoice 输出的事件/语言标签（如 <|zh|>）。"""
    import re

    return re.sub(r"<\|[^|]+\|>", "", text)


def _resample_16k_mono(audio_path: Path) -> Path:
    """将任意采样率音频重采样为 16k 单声道 WAV（FunASR 输入要求）。"""
    try:
        import librosa
        import soundfile as sf
    except ModuleNotFoundError as exc:
        raise RuntimeError("funasr 依赖缺失：pip install funasr（含 librosa/soundfile）") from exc
    y, _sr = librosa.load(str(audio_path), sr=16000, mono=True)
    out = audio_path.with_suffix(".funasr-16k.wav")
    sf.write(str(out), y, 16000)
    return out


def create_asr_backend(settings: Settings) -> AsrBackend:
    if settings.asr_backend == "whisper":
        return WhisperBackend(model=settings.asr_model)
    if settings.asr_backend == "funasr":
        return FunasrBackend(model=settings.asr_model)
    raise ValueError(f"未知 ASR 后端：{settings.asr_backend}")

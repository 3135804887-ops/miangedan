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

    def __init__(
        self, model: str = "small", device: str = "cpu", compute_type: str = "int8"
    ) -> None:
        self._model_name = model
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
    """FunASR/Paraformer backend（GPU 部署预留，需安装 funasr 与 torch）。"""

    def __init__(self, model: str = "paraformer-zh") -> None:
        self._model_name = model
        self._model: Any = None

    def _ensure_model(self) -> Any:
        if self._model is None:
            try:
                module: Any = importlib.import_module("funasr")
            except ModuleNotFoundError as exc:
                raise RuntimeError("funasr 未安装：GPU 部署时 pip install funasr") from exc
            self._model = module.AutoModel(model=self._model_name)
        return self._model

    def transcribe(self, audio_path: Path, language: str | None = None) -> str:
        model = self._ensure_model()
        result = model.generate(input=str(audio_path), language=language or "zh")
        if not result:
            return ""
        text = result[0].get("text", "")
        return str(text).strip()


def create_asr_backend(settings: Settings) -> AsrBackend:
    if settings.asr_backend == "whisper":
        return WhisperBackend(model=settings.asr_model)
    if settings.asr_backend == "funasr":
        return FunasrBackend()
    raise ValueError(f"未知 ASR 后端：{settings.asr_backend}")

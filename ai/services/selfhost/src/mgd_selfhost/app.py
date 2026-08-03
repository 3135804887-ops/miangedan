"""FastAPI application for self-hosted ASR/TTS."""

from __future__ import annotations

import os
import tempfile
from pathlib import Path

from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import Response

from mgd_selfhost.asr import AsrBackend, create_asr_backend
from mgd_selfhost.config import Settings
from mgd_selfhost.tts import TtsBackend, create_tts_backend


def create_app(settings: Settings | None = None) -> FastAPI:
    cfg = settings or Settings.from_env()
    asr_backend = create_asr_backend(cfg)
    tts_backend = create_tts_backend(cfg)
    app = FastAPI(title="mgd-selfhost", version="0.1.0")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=[
            "http://localhost:3000",
            "http://127.0.0.1:3000",
            "http://localhost:8765",
            "http://127.0.0.1:8765",
        ],
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.get("/healthz")
    def healthz() -> dict[str, str]:
        return {"status": "ok"}

    @app.post("/v1/asr/transcribe")
    def transcribe(file: UploadFile = File(...), language: str | None = None) -> dict[str, str]:
        suffix = os.path.splitext(file.filename or "audio.wav")[1] or ".wav"
        with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
            tmp.write(file.file.read())
            tmp_path = Path(tmp.name)
        text = _run_transcribe(asr_backend, tmp_path, language)
        return {"text": text}

    @app.post("/v1/tts/synthesize")
    def synthesize(text: str = Form(...)) -> Response:
        if not text.strip():
            raise HTTPException(status_code=400, detail="text 不能为空")
        data = _run_tts(tts_backend, text)
        return Response(content=data, media_type="audio/wav")

    return app


def _run_transcribe(backend: AsrBackend, audio_path: Path, language: str | None) -> str:
    """同步执行转写并清理临时文件（避免 async 内使用 pathlib 方法）。"""
    try:
        return backend.transcribe(audio_path, language=language)
    finally:
        audio_path.unlink(missing_ok=True)


def _run_tts(backend: TtsBackend, text: str) -> bytes:
    """同步执行合成并返回 WAV 字节（避免 async 内使用 pathlib 方法）。"""
    out_path = Path(tempfile.mkdtemp()) / "tts.wav"
    try:
        backend.synthesize(text, out_path)
        return out_path.read_bytes()
    finally:
        out_path.unlink(missing_ok=True)

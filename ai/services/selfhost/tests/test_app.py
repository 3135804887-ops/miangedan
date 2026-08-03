from __future__ import annotations

import importlib.util
import os
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

from mgd_selfhost.app import create_app
from mgd_selfhost.asr import create_asr_backend
from mgd_selfhost.config import Settings
from mgd_selfhost.tts import create_tts_backend

HAS_WHISPER = importlib.util.find_spec("faster_whisper") is not None
HAS_PIPER = importlib.util.find_spec("piper") is not None
VOICE_DIR = os.environ.get("SELFHOST_TTS_VOICE_DIR", "")
HAS_VOICE = bool(VOICE_DIR) and Path(VOICE_DIR, "zh_CN-huayan-medium.onnx").exists()


def client() -> TestClient:
    return TestClient(create_app(Settings()))


def test_healthz() -> None:
    resp = client().get("/healthz")
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


def test_cors_allows_local_frontend() -> None:
    resp = client().get("/healthz", headers={"Origin": "http://localhost:3000"})
    assert resp.status_code == 200
    assert resp.headers.get("access-control-allow-origin") == "http://localhost:3000"


def test_transcribe_requires_file() -> None:
    resp = client().post("/v1/asr/transcribe")
    assert resp.status_code == 422


def test_tts_rejects_empty_text() -> None:
    resp = client().post("/v1/tts/synthesize", data={"text": "  "})
    assert resp.status_code == 400


def test_unknown_asr_backend() -> None:
    with pytest.raises(ValueError):
        create_asr_backend(Settings(asr_backend="unknown"))


def test_unknown_tts_backend() -> None:
    with pytest.raises(ValueError):
        create_tts_backend(Settings(tts_backend="unknown"))


def test_funasr_default_model() -> None:
    from mgd_selfhost.asr import FunasrBackend

    backend = FunasrBackend()
    assert backend._model_name == FunasrBackend.DEFAULT_MODEL


def test_edge_tts_default_voice() -> None:
    from mgd_selfhost.tts import EdgeTtsBackend

    backend = EdgeTtsBackend()
    assert backend._voice_name == EdgeTtsBackend.DEFAULT_VOICE


@pytest.mark.skipif(
    not (HAS_WHISPER and HAS_PIPER and HAS_VOICE),
    reason="需要 asr-whisper/tts-piper 扩展与 piper 音色文件",
)
def test_tts_asr_closed_loop(tmp_path: Path) -> None:
    tts = create_tts_backend(Settings(tts_backend="piper", tts_voice_dir=VOICE_DIR))
    asr = create_asr_backend(Settings(asr_backend="whisper", asr_model="small"))
    audio = tmp_path / "loop.wav"
    tts.synthesize("今天是个好日子。", audio)
    text = asr.transcribe(audio, language="zh")
    assert text

from __future__ import annotations

import json
from typing import Any

import pytest

from mgd_orchestrator.llm_provider import Completion, DeepSeekProvider, LlmError


class FakeResponse:
    def __init__(self, payload: bytes) -> None:
        self._payload = payload

    def __enter__(self) -> FakeResponse:
        return self

    def __exit__(self, *args: object) -> None:
        return None

    def read(self) -> bytes:
        return self._payload


def _make_urlopen(payload: dict[str, Any]) -> tuple[Any, dict[str, Any]]:
    captured: dict[str, Any] = {}

    def urlopen(request: Any, timeout: float = 60.0) -> FakeResponse:
        captured["url"] = request.full_url
        captured["authorization"] = request.get_header("Authorization")
        captured["timeout"] = timeout
        return FakeResponse(json.dumps(payload).encode("utf-8"))

    return urlopen, captured


def test_deepseek_complete_success() -> None:
    urlopen, captured = _make_urlopen(
        {
            "choices": [
                {
                    "message": {"content": " 结构化回答 "},
                    "finish_reason": "stop",
                }
            ],
            "usage": {"prompt_tokens": 10, "completion_tokens": 20},
        }
    )
    provider = DeepSeekProvider(
        base_url="https://api.deepseek.com/",
        api_key="sk-test",
        model="deepseek-chat",
        urlopen=urlopen,
    )
    result = provider.complete([{"role": "user", "content": "介绍一下你自己"}], max_tokens=128)
    assert isinstance(result, Completion)
    assert result.text == "结构化回答"
    assert result.finish_reason == "stop"
    assert result.prompt_tokens == 10
    assert result.completion_tokens == 20
    assert captured["url"] == "https://api.deepseek.com/chat/completions"
    assert captured["authorization"] == "Bearer sk-test"


def test_deepseek_invalid_config() -> None:
    with pytest.raises(ValueError):
        DeepSeekProvider(base_url="", api_key="", model="")


def test_deepseek_empty_messages() -> None:
    urlopen, _captured = _make_urlopen({"choices": [], "usage": {}})
    provider = DeepSeekProvider(
        base_url="https://api.deepseek.com",
        api_key="sk-test",
        model="deepseek-chat",
        urlopen=urlopen,
    )
    with pytest.raises(ValueError):
        provider.complete([])


def test_deepseek_malformed_response() -> None:
    urlopen, _captured = _make_urlopen({"unexpected": True})
    provider = DeepSeekProvider(
        base_url="https://api.deepseek.com",
        api_key="sk-test",
        model="deepseek-chat",
        urlopen=urlopen,
    )
    with pytest.raises(LlmError):
        provider.complete([{"role": "user", "content": "hi"}])

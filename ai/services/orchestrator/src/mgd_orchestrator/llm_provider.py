"""LLM 供应商适配层（OD-01 自建矩阵：DeepSeek API，OpenAI 兼容端点）。"""

from __future__ import annotations

import json
import time
import urllib.request
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from typing import Any
from urllib.parse import urlsplit

Message = dict[str, str]
Urlopen = Callable[..., Any]


class LlmError(RuntimeError):
    """LLM 调用失败（鉴权、限流、服务错误等）。"""


@dataclass(frozen=True)
class Completion:
    """一次补全结果（不含任何敏感字段）。"""

    text: str
    finish_reason: str
    prompt_tokens: int
    completion_tokens: int
    latency_ms: float


class LlmProvider:
    """LLM 能力契约（PROVIDER-ADAPTERS §4.1：complete）。"""

    def complete(
        self,
        messages: Sequence[Message],
        *,
        temperature: float = 0.3,
        max_tokens: int = 1024,
    ) -> Completion:
        raise NotImplementedError


class DeepSeekProvider(LlmProvider):
    """DeepSeek API 实现（https://api.deepseek.com/chat/completions）。"""

    def __init__(
        self,
        base_url: str,
        api_key: str,
        model: str,
        *,
        timeout: float = 60.0,
        urlopen: Urlopen | None = None,
    ) -> None:
        if not base_url or not api_key or not model:
            raise ValueError("base_url/api_key/model 必填")
        if urlsplit(base_url).scheme not in ("http", "https"):
            raise ValueError("base_url 仅允许 http/https")
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._model = model
        self._timeout = timeout
        self._urlopen = urlopen or urllib.request.urlopen

    def complete(
        self,
        messages: Sequence[Message],
        *,
        temperature: float = 0.3,
        max_tokens: int = 1024,
    ) -> Completion:
        if not messages:
            raise ValueError("messages 不能为空")
        payload = {
            "model": self._model,
            "messages": [dict(message) for message in messages],
            "temperature": temperature,
            "max_tokens": max_tokens,
            "stream": False,
        }
        request = urllib.request.Request(  # noqa: S310 - 已在校验 http/https 后使用
            self._base_url + "/chat/completions",
            data=json.dumps(payload).encode("utf-8"),
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self._api_key}",
            },
            method="POST",
        )
        start = time.perf_counter()
        try:
            with self._urlopen(request, timeout=self._timeout) as response:
                body = json.loads(response.read().decode("utf-8"))
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")[:512]
            raise LlmError(f"deepseek http {exc.code}: {detail}") from exc
        except urllib.error.URLError as exc:
            raise LlmError(f"deepseek 网络错误: {exc.reason}") from exc
        elapsed_ms = (time.perf_counter() - start) * 1000

        try:
            choice = body["choices"][0]
            content = str(choice["message"]["content"]).strip()
            finish = str(choice.get("finish_reason", ""))
            usage = body.get("usage", {})
            prompt_tokens = int(usage.get("prompt_tokens", 0))
            completion_tokens = int(usage.get("completion_tokens", 0))
        except (KeyError, IndexError, TypeError, ValueError) as exc:
            raise LlmError(f"deepseek 响应格式异常: {exc}") from exc

        return Completion(
            text=content,
            finish_reason=finish,
            prompt_tokens=prompt_tokens,
            completion_tokens=completion_tokens,
            latency_ms=round(elapsed_ms, 1),
        )

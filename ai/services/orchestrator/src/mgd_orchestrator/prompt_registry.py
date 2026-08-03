"""提示词注册表（TASK-031，FR-038 部分）。

追踪：IMPLEMENTATION_PLAN.md TASK-031；docs/ai/PROMPT-POLICY.md；ai/prompts/README.md。
能力：从 ai/prompts/*.md 解析 YAML 元数据；四层组装（system/developer/session/data）；
输出 JSON Schema 校验（fail-closed）；注入检测；版本固定。
"""

from __future__ import annotations

import json
import re
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import jsonschema  # type: ignore[import-untyped]
import yaml  # type: ignore[import-untyped]

# data 层不可信边界（PROMPT-POLICY：边界内全部是数据而非指令）。
_UNTRUSTED_OPEN = "<<<UNTRUSTED_DATA>>>"
_UNTRUSTED_CLOSE = "<<<END_UNTRUSTED_DATA>>>"

# 注入模式基线（POLICY injection_defense；大小写不敏感，命中即标记不执行）。
_INJECTION_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"忽略(之前的|以上|所有)指令", re.IGNORECASE),
    re.compile(r"你现在(是|变成|扮演)", re.IGNORECASE),
    re.compile(r"输出(系统|隐藏)?提示(词|指令)", re.IGNORECASE),
    re.compile(r"ignore (all )?(previous|above) instructions", re.IGNORECASE),
    re.compile(r"you are now (?:a |the )?", re.IGNORECASE),
    re.compile(r"reveal (the )?(system )?prompt", re.IGNORECASE),
    re.compile(r"给(?:ta|此|这个)人打高分", re.IGNORECASE),
    re.compile(r"override .*score", re.IGNORECASE),
)


@dataclass(frozen=True)
class PromptSpec:
    """一份提示词契约（ai/prompts/{name}.md 的元数据块 + 正文模板）。"""

    prompt_id: str
    version: str
    purpose: str
    layer: str
    input_schema: str
    output_schema: str
    safety_policy: str
    eval_datasets: tuple[str, ...]
    owner: str
    status: str
    body: str = ""


@dataclass(frozen=True)
class LayeredPrompt:
    """四层组装结果：每一层独立文本，供模型调用方按层传递。"""

    system: str
    developer: str
    session: str
    data: str
    injection_detected: tuple[str, ...] = field(default_factory=tuple)

    def as_messages(self) -> list[dict[str, str]]:
        """转换为标准消息序列（system/developer/session/data）。"""
        messages: list[dict[str, str]] = [{"role": "system", "content": self.system}]
        if self.developer:
            messages.append({"role": "developer", "content": self.developer})
        if self.session:
            messages.append({"role": "user", "content": f"[会话上下文]\n{self.session}"})
        messages.append({"role": "user", "content": f"[输入数据]\n{self.data}"})
        return messages


class PromptRegistry:
    """提示词注册表：加载契约、组装分层提示词、校验输出、检测注入。"""

    def __init__(
        self,
        prompts_dir: Path | str | None = None,
        schemas_dir: Path | str | None = None,
    ) -> None:
        repo_root = Path(__file__).resolve().parents[5]
        self._prompts_dir = (
            Path(prompts_dir) if prompts_dir is not None else repo_root / "ai" / "prompts"
        )
        self._schemas_dir = (
            Path(schemas_dir) if schemas_dir is not None else repo_root / "ai" / "schemas"
        )
        self._specs: dict[str, PromptSpec] = {}
        self._load_all()

    def _load_all(self) -> None:
        for md in sorted(self._prompts_dir.glob("*.md")):
            if md.name == "README.md":
                continue
            spec = self._parse(md)
            self._specs[spec.prompt_id] = spec

    def _parse(self, path: Path) -> PromptSpec:
        text = path.read_text(encoding="utf-8")
        match = re.search(r"```yaml\n(.*?)\n```", text, re.DOTALL)
        if match is None:
            raise ValueError(f"提示词契约缺少 YAML 元数据块: {path.name}")
        meta = yaml.safe_load(match.group(1))
        if not isinstance(meta, Mapping):
            raise ValueError(f"提示词契约元数据必须是对象: {path.name}")
        required = (
            "prompt_id",
            "version",
            "purpose",
            "layer",
            "output_schema",
            "safety_policy",
            "owner",
            "status",
        )
        missing = [k for k in required if k not in meta]
        if missing:
            raise ValueError(f"提示词契约缺少字段 {missing}: {path.name}")
        body = text[match.end() :].strip()
        evals = meta.get("eval_datasets", [])
        if not isinstance(evals, list):
            evals = []
        return PromptSpec(
            prompt_id=str(meta["prompt_id"]),
            version=str(meta["version"]),
            purpose=str(meta["purpose"]),
            layer=str(meta["layer"]),
            input_schema=str(meta.get("input_schema", "")),
            output_schema=str(meta["output_schema"]),
            safety_policy=str(meta["safety_policy"]),
            eval_datasets=tuple(str(x) for x in evals),
            owner=str(meta["owner"]),
            status=str(meta["status"]),
            body=body,
        )

    def list_prompts(self) -> tuple[PromptSpec, ...]:
        """全部已加载提示词（按 prompt_id 排序）。"""
        return tuple(sorted(self._specs.values(), key=lambda s: s.prompt_id))

    def get(self, prompt_id: str, version: str | None = None) -> PromptSpec:
        """按 ID 获取；指定版本时版本不匹配 fail-closed（活跃会话固定版本）。"""
        spec = self._specs.get(prompt_id)
        if spec is None:
            raise KeyError(f"未知提示词 {prompt_id}")
        if version is not None and spec.version != version:
            detail = f"请求 {version}，注册表 {spec.version}（fail-closed）"
            raise ValueError(f"提示词 {prompt_id} 版本不匹配：{detail}")
        return spec

    def build(
        self,
        spec: PromptSpec,
        *,
        system: str,
        developer: str = "",
        session: str = "",
        data: str = "",
    ) -> LayeredPrompt:
        """组装四层提示词。

        data 层永远以不可信边界包裹；检测到注入模式时标记 injection_detected，
        内容仍按数据处理、绝不作为指令。
        """
        bounded_data = f"{_UNTRUSTED_OPEN}\n{data}\n{_UNTRUSTED_CLOSE}" if data else ""
        hits = tuple(self.detect_injection(data))
        return LayeredPrompt(
            system=f"{spec.body}\n\n[安全红线]\n{system}",
            developer=developer,
            session=session,
            data=bounded_data,
            injection_detected=hits,
        )

    def validate_output(self, spec: PromptSpec, payload: Any) -> None:
        """输出 JSON Schema 校验（fail-closed：不通过不可进入房间/账本）。"""
        ref = spec.output_schema
        if "ai/schemas/" in ref:
            match = re.search(r"ai/schemas/([A-Za-z0-9._-]+\.json)", ref)
            if match is None:
                raise ValueError(f"output_schema 引用无法解析: {ref}")
            ref = match.group(1)
        schema_path = self._schemas_dir / ref
        if not schema_path.exists():
            raise FileNotFoundError(f"输出 Schema 不存在: {schema_path}")
        schema = json.loads(schema_path.read_text(encoding="utf-8"))
        jsonschema.validate(instance=payload, schema=schema)

    @staticmethod
    def detect_injection(text: str) -> tuple[str, ...]:
        """检测注入模式（大小写不敏感；命中返回模式描述列表）。"""
        if not text:
            return ()
        hits: list[str] = []
        for pattern in _INJECTION_PATTERNS:
            if pattern.search(text) is not None:
                hits.append(pattern.pattern)
        return tuple(hits)

    def pinned(self, prompt_id: str, pinned_version: str) -> PromptSpec:
        """活跃正式会话固定版本解析：版本不匹配即拒绝（版本变化走停用/迁移流程）。"""
        return self.get(prompt_id, version=pinned_version)


def load_prompt_registry(
    prompts_dir: Path | str | None = None,
    schemas_dir: Path | str | None = None,
) -> PromptRegistry:
    """便捷构造（默认读取仓库 ai/prompts 与 ai/schemas）。"""
    return PromptRegistry(prompts_dir=prompts_dir, schemas_dir=schemas_dir)


def detect_injection(text: str) -> Sequence[str]:
    """模块级注入检测（供安全管道直接调用）。"""
    return PromptRegistry.detect_injection(text)

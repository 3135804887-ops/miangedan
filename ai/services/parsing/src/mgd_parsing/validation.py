"""resume-profile JSON Schema 校验适配器。"""

from __future__ import annotations

from typing import Protocol

from jsonschema import Draft202012Validator, FormatChecker  # type: ignore[import-untyped]

from .models import JsonObject


class ProfileSchemaError(ValueError):
    """供应商输出或人工编辑不符合安全画像 Schema。"""


class ProfileValidator(Protocol):
    """结构化画像写入前的 Schema 校验边界。"""

    def validate(self, profile: JsonObject) -> None:
        """校验失败时抛出 ProfileSchemaError。"""


class JsonSchemaProfileValidator:
    """基于 JSON Schema Draft 2020-12 的严格校验器。"""

    def __init__(self, schema: JsonObject) -> None:
        Draft202012Validator.check_schema(schema)
        self._validator = Draft202012Validator(schema, format_checker=FormatChecker())

    def validate(self, profile: JsonObject) -> None:
        """只返回字段路径和校验器名称，避免错误信息复述敏感值。"""
        errors = sorted(self._validator.iter_errors(profile), key=lambda item: list(item.path))
        if not errors:
            return
        first = errors[0]
        path = "/" + "/".join(str(part) for part in first.absolute_path)
        raise ProfileSchemaError(f"resume profile schema rejected {path}: {first.validator}")

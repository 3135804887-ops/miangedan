"""敏感字段脱敏与零泄露硬门槛（TASK-013、FR-003、SEC-040）。"""

from __future__ import annotations

import copy
import json
import re
from dataclasses import dataclass

from .models import JsonObject, JsonValue, SensitiveCategory

_EMAIL = re.compile(r"(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b")
_PHONE = re.compile(r"(?<![\w.])\+?\d(?:[ -]?\d){8,14}(?![\w.])")
_SENSITIVE_LINE_PATTERNS: tuple[tuple[SensitiveCategory, re.Pattern[str]], ...] = (
    (
        SensitiveCategory.PHONE,
        re.compile(r"(?im)^.*(?:电话|手机|联系电话|phone|mobile|telephone)\s*[:：].*$"),
    ),
    (
        SensitiveCategory.EMAIL,
        re.compile(r"(?im)^.*(?:邮箱|电子邮件|e-?mail)\s*[:：].*$"),
    ),
    (
        SensitiveCategory.ID_NUMBER,
        re.compile(
            r"(?im)^.*(?:身份证|证件号|护照号|社会安全号|id\s*(?:number|no)|passport|ssn)\s*[:：].*$"
        ),
    ),
    (
        SensitiveCategory.ADDRESS,
        re.compile(r"(?im)^.*(?:详细地址|住址|家庭地址|地址|address|street address)\s*[:：].*$"),
    ),
    (
        SensitiveCategory.PHOTO,
        re.compile(
            r"(?im)^.*(?:照片|头像|证件照|photo|portrait|headshot)\s*[:：].*$|^!\[[^]]*]\([^)]*\).*$"
        ),
    ),
    (
        SensitiveCategory.PROTECTED_ATTRIBUTE,
        re.compile(
            r"(?im)^.*(?:性别|年龄|出生日期|种族|民族|国籍|残障|残疾|婚育|婚姻|生育|宗教|"
            r"外貌|情绪|微表情|人格|gender|age|date of birth|race|ethnicity|nationality|"
            r"disability|marital|pregnan|religion|appearance|emotion|personality)\s*[:：].*$"
        ),
    ),
)

_KEY_CATEGORIES: tuple[tuple[SensitiveCategory, frozenset[str]], ...] = (
    (SensitiveCategory.PHONE, frozenset({"phone", "mobile", "telephone", "tel", "手机号", "电话"})),
    (SensitiveCategory.EMAIL, frozenset({"email", "e_mail", "mail", "邮箱", "电子邮件"})),
    (
        SensitiveCategory.ID_NUMBER,
        frozenset({"id_number", "identity_number", "passport", "ssn", "证件号", "身份证"}),
    ),
    (
        SensitiveCategory.ADDRESS,
        frozenset({"address", "home_address", "street_address", "详细地址", "住址", "地址"}),
    ),
    (
        SensitiveCategory.PHOTO,
        frozenset({"photo", "avatar", "portrait", "headshot", "照片", "头像"}),
    ),
    (
        SensitiveCategory.PROTECTED_ATTRIBUTE,
        frozenset(
            {
                "gender",
                "sex",
                "age",
                "birth_date",
                "date_of_birth",
                "race",
                "ethnicity",
                "nationality",
                "disability",
                "marital_status",
                "pregnancy",
                "religion",
                "appearance",
                "emotion",
                "micro_expression",
                "personality",
                "性别",
                "年龄",
                "出生日期",
                "种族",
                "民族",
                "国籍",
                "残障",
                "婚育",
                "婚姻",
                "宗教",
                "外貌",
                "情绪",
                "微表情",
                "人格",
            }
        ),
    ),
)


class SensitiveContentError(ValueError):
    """敏感值试图进入结构化版本、面试上下文或评分材料。"""


@dataclass(frozen=True, slots=True)
class RedactionResult:
    """脱敏文本及命中的类别集合，不保留任何命中值。"""

    text: str
    categories: frozenset[SensitiveCategory]


def classify_sensitive_key(key: str) -> SensitiveCategory | None:
    """按字段名识别禁止进入安全画像的敏感类别。"""
    normalized = re.sub(r"[^\w\u4e00-\u9fff]+", "_", key.strip().lower()).strip("_")
    for category, names in _KEY_CATEGORIES:
        if normalized in names:
            return category
    return None


def redact_resume_text(text: str) -> RedactionResult:
    """在调用解析适配层前按类别移除敏感行并替换游离联系方式。"""
    categories: set[SensitiveCategory] = set()
    redacted = text
    for category, pattern in _SENSITIVE_LINE_PATTERNS:
        redacted, count = pattern.subn(f"[REDACTED:{category.value}]", redacted)
        if count:
            categories.add(category)
    redacted, email_count = _EMAIL.subn("[REDACTED:email]", redacted)
    if email_count:
        categories.add(SensitiveCategory.EMAIL)
    redacted, phone_count = _PHONE.subn("[REDACTED:phone]", redacted)
    if phone_count:
        categories.add(SensitiveCategory.PHONE)
    return RedactionResult(text=redacted, categories=frozenset(categories))


def sanitize_profile(
    profile: JsonObject, initial_categories: frozenset[SensitiveCategory]
) -> tuple[JsonObject, frozenset[SensitiveCategory]]:
    """递归删除敏感键并脱敏字符串，返回可进入版本表的画像。"""
    categories = set(initial_categories)

    def sanitize(value: JsonValue) -> JsonValue:
        if isinstance(value, dict):
            safe: JsonObject = {}
            for key, child in value.items():
                category = classify_sensitive_key(key)
                if category is not None:
                    categories.add(category)
                    continue
                safe[key] = sanitize(child)
            return safe
        if isinstance(value, list):
            return [sanitize(item) for item in value]
        if isinstance(value, str):
            result = redact_resume_text(value)
            categories.update(result.categories)
            return result.text
        return value

    sanitized = sanitize(copy.deepcopy(profile))
    if not isinstance(sanitized, dict):
        raise TypeError("resume profile must be an object")
    sanitized["excluded_sensitive_fields"] = [category.value for category in sorted(categories)]
    return sanitized, frozenset(categories)


def find_sensitive_content(value: JsonValue, *, path: str = "") -> list[str]:
    """递归返回敏感命中路径；类别清单本身被允许且不含原值。"""
    findings: list[str] = []
    if isinstance(value, dict):
        for key, child in value.items():
            child_path = f"{path}/{key}"
            if key == "resume_id":
                continue
            category = classify_sensitive_key(key)
            if category is not None:
                findings.append(f"{child_path}:{category.value}")
                continue
            if key == "excluded_sensitive_fields":
                continue
            findings.extend(find_sensitive_content(child, path=child_path))
    elif isinstance(value, list):
        for index, child in enumerate(value):
            findings.extend(find_sensitive_content(child, path=f"{path}/{index}"))
    elif isinstance(value, str):
        if _EMAIL.search(value):
            findings.append(f"{path}:email")
        if _PHONE.search(value):
            findings.append(f"{path}:phone")
        for category, pattern in _SENSITIVE_LINE_PATTERNS:
            if pattern.search(value):
                findings.append(f"{path}:{category.value}")
    return findings


def assert_sensitive_content_absent(value: JsonValue, *, destination: str) -> None:
    """敏感命中非零时 fail-closed，错误只含类别与路径，不含原值。"""
    findings = find_sensitive_content(value)
    if findings:
        summary = ", ".join(sorted(set(findings)))
        raise SensitiveContentError(f"{destination} sensitive-content gate rejected: {summary}")


def leakage_count(value: JsonValue) -> int:
    """返回测试/评测使用的敏感命中数；SEC-040 门槛必须恒为 0。"""
    return len(find_sensitive_content(value))


def stable_json(value: JsonValue) -> str:
    """生成不含日志副作用的稳定 JSON，用于幂等指纹。"""
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))

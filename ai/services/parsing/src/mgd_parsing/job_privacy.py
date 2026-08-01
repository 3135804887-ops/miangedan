"""JD 排除字段硬门槛（TASK-014；FR-004）。"""

from __future__ import annotations

import re
from dataclasses import dataclass

from .models import JsonObject, JsonValue

_CATEGORY_PATTERNS: dict[str, tuple[re.Pattern[str], ...]] = {
    "salary_benefits": (
        re.compile(r"薪资|薪酬|福利|五险|奖金|compensation|salary|benefits?", re.IGNORECASE),
    ),
    "recruiter_contact": (
        re.compile(
            r"招聘联系人|联系人|招聘邮箱|recruiter\s+contact|contact\s+recruiter",
            re.IGNORECASE,
        ),
    ),
    "company_perks": (
        re.compile(
            r"公司福利|员工福利|团建|下午茶|company\s+perks?|employee\s+perks?",
            re.IGNORECASE,
        ),
    ),
}
_CONTACT_VALUE_PATTERNS = (
    re.compile(r"[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}", re.IGNORECASE),
    re.compile(r"(?<!\w)(?:\+\d[\d()\s-]{6,}\d)(?!\w)"),
)
_FORBIDDEN_KEYS = frozenset(
    {
        "salary",
        "compensation",
        "benefits",
        "perks",
        "recruiter",
        "recruiter_contact",
        "contact_email",
        "contact_phone",
    }
)


class ExcludedJobContentError(ValueError):
    """排除内容试图进入岗位画像或下游材料。"""


@dataclass(frozen=True, slots=True)
class JobRedactionResult:
    """预解析剔除结果；只暴露类别，不暴露原值。"""

    text: str
    categories: frozenset[str]


def redact_job_text(text: str) -> JobRedactionResult:
    """删除薪资福利、公司福利及招聘联系人整段，并兜底清除联系方式。"""
    categories: set[str] = set()
    output: list[str] = []
    excluded_section: str | None = None
    for line in text.splitlines():
        if line.lstrip().startswith("#"):
            excluded_section = _category_for_text(line)
            if excluded_section is not None:
                categories.add(excluded_section)
                continue
        if excluded_section is not None:
            continue
        line_category = _category_for_text(line)
        if line_category is not None:
            categories.add(line_category)
            continue
        sanitized = line
        for pattern in _CONTACT_VALUE_PATTERNS:
            if pattern.search(sanitized):
                categories.add("recruiter_contact")
                sanitized = pattern.sub("[EXCLUDED_RECRUITER_CONTACT]", sanitized)
        output.append(sanitized)
    return JobRedactionResult(text="\n".join(output), categories=frozenset(categories))


def sanitize_job_profile(
    profile: JsonObject, categories: frozenset[str]
) -> tuple[JsonObject, frozenset[str]]:
    """递归删除供应商误带回的排除键与内容。"""
    found = set(categories)

    def clean(value: JsonValue) -> JsonValue:
        if isinstance(value, dict):
            cleaned: dict[str, object] = {}
            for raw_key, child in value.items():
                key = str(raw_key)
                lowered = key.lower()
                if lowered in _FORBIDDEN_KEYS:
                    found.add(_category_for_key(lowered))
                    continue
                cleaned[key] = clean(child)
            return cleaned
        if isinstance(value, list):
            cleaned_items: list[object] = []
            for child in value:
                if isinstance(child, str) and _category_for_text(child) is not None:
                    found.add(_category_for_text(child) or "salary_benefits")
                    continue
                cleaned_items.append(clean(child))
            return cleaned_items
        if isinstance(value, str):
            category = _category_for_text(value)
            if category is not None or any(
                pattern.search(value) for pattern in _CONTACT_VALUE_PATTERNS
            ):
                found.add(category or "recruiter_contact")
                return "[EXCLUDED]"
        return value

    cleaned = clean(profile)
    if not isinstance(cleaned, dict):
        raise ExcludedJobContentError("job profile root must be an object")
    return cleaned, frozenset(found)


def job_leakage_count(value: JsonValue) -> int:
    """统计排除键、联系方式或排除内容的递归命中数。"""
    hits = 0
    if isinstance(value, dict):
        for raw_key, child in value.items():
            key = str(raw_key).lower()
            if key in _FORBIDDEN_KEYS:
                hits += 1
            if key != "excluded_from_scoring":
                hits += job_leakage_count(child)
    elif isinstance(value, list):
        hits += sum(job_leakage_count(child) for child in value)
    elif isinstance(value, str):
        if _category_for_text(value) is not None:
            hits += 1
        hits += sum(1 for pattern in _CONTACT_VALUE_PATTERNS if pattern.search(value))
    return hits


def assert_excluded_content_absent(value: JsonValue, *, destination: str) -> None:
    """在版本与下游材料边界 fail-closed。"""
    if job_leakage_count(value):
        raise ExcludedJobContentError(f"excluded JD content blocked before {destination}")


def _category_for_text(text: str) -> str | None:
    for category, patterns in _CATEGORY_PATTERNS.items():
        if any(pattern.search(text) for pattern in patterns):
            return category
    return None


def _category_for_key(key: str) -> str:
    if key in {"recruiter", "recruiter_contact", "contact_email", "contact_phone"}:
        return "recruiter_contact"
    if key == "perks":
        return "company_perks"
    return "salary_benefits"

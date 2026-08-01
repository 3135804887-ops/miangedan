"""简历解析领域模型（TASK-013；FR-002、FR-003、SEC-040）。"""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum
from typing import TypeAlias

JsonValue: TypeAlias = object
JsonObject: TypeAlias = dict[str, object]


class ParseStatus(StrEnum):
    """简历解析任务状态。"""

    PENDING = "PENDING"
    PARSING = "PARSING"
    AWAITING_CONFIRMATION = "AWAITING_CONFIRMATION"
    CONFIRMED = "CONFIRMED"
    RETRYABLE_FAILURE = "RETRYABLE_FAILURE"
    FAILED = "FAILED"


class SensitiveCategory(StrEnum):
    """只记录类别、不记录值的敏感字段分类。"""

    PHONE = "phone"
    EMAIL = "email"
    ID_NUMBER = "id_number"
    ADDRESS = "address"
    PHOTO = "photo"
    PROTECTED_ATTRIBUTE = "protected_attribute"
    OTHER_PII = "other_pii"


@dataclass(frozen=True, slots=True)
class PromptMessage:
    """按 PROMPT-POLICY 分层且不可混排的提示消息。"""

    layer: str
    role: str
    content: str


@dataclass(frozen=True, slots=True)
class ResumeProviderRequest:
    """供应商中立简历解析请求。"""

    data_region: str
    language: str
    messages: tuple[PromptMessage, ...]
    output_schema_id: str
    timeout_seconds: int
    trace_id: str


@dataclass(frozen=True, slots=True)
class ResumeProviderResult:
    """解析适配层返回的候选事实与逐字段置信度。"""

    profile_fields: JsonObject
    field_confidences: dict[str, float]
    provider_version: str
    injection_detected: bool = False


@dataclass(frozen=True, slots=True)
class ParseImpact:
    """解析异常对原件、重试、计费与评分的影响。"""

    original_input_retained: bool
    retryable: bool
    billable: bool = False
    scoring_affected: bool = False
    retry_action: str | None = None


@dataclass(frozen=True, slots=True)
class ParseTask:
    """一次幂等简历解析任务。"""

    task_id: str
    resume_id: str
    upload_id: str
    user_id: str
    data_region: str
    language: str
    idempotency_key: str
    status: ParseStatus
    input_fingerprint: str
    impact: ParseImpact
    version: int | None = None
    message: str | None = None


@dataclass(frozen=True, slots=True)
class ResumeVersion:
    """追加式、不可变的简历结构化版本。"""

    resume_id: str
    version: int
    data_region: str
    profile: JsonObject
    confirmed_by_user: bool
    reviewed_low_confidence_paths: frozenset[str] = field(default_factory=frozenset)


@dataclass(frozen=True, slots=True)
class StartParseRequest:
    """从已安全接受的上传启动解析。"""

    resume_id: str
    upload_id: str
    user_id: str
    data_region: str
    language: str
    idempotency_key: str
    trace_id: str


@dataclass(frozen=True, slots=True)
class FieldEditRequest:
    """单字段编辑或低置信度确认命令。"""

    base_version: int
    path: str
    operation: str
    idempotency_key: str
    value: JsonValue = None

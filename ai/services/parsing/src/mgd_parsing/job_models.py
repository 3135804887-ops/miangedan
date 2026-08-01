"""JD 解析与缺失材料降级领域模型（TASK-014；FR-004、FR-005）。"""

from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum

from .models import JsonObject, JsonValue, ParseImpact, PromptMessage


class JobStatus(StrEnum):
    """岗位及解析执行状态。"""

    CREATED = "CREATED"
    PARSING = "PARSING"
    AWAITING_CONFIRMATION = "AWAITING_CONFIRMATION"
    CONFIRMED = "CONFIRMED"
    RETRYABLE_FAILURE = "RETRYABLE_FAILURE"
    FAILED = "FAILED"


class MaterialMode(StrEnum):
    """PRD FR-005 固定的四种材料组合。"""

    FULL = "full"
    JD_ONLY = "jd_only"
    RESUME_ONLY = "resume_only"
    NEITHER = "neither"


@dataclass(frozen=True, slots=True)
class JobProviderRequest:
    """供应商中立 JD 解析请求。"""

    data_region: str
    language: str
    messages: tuple[PromptMessage, ...]
    output_schema_id: str
    timeout_seconds: int
    trace_id: str
    source_kind: str = "jd_text"


@dataclass(frozen=True, slots=True)
class JobProviderResult:
    """适配层返回的岗位事实、AI 推理和字段置信度。"""

    profile_fields: JsonObject
    field_confidences: dict[str, float]
    provider_version: str
    injection_detected: bool = False


@dataclass(frozen=True, slots=True)
class JobRecord:
    """岗位聚合根；JD 原文只通过受限原文存储读取。"""

    job_id: str
    user_id: str
    data_region: str
    language: str
    source_kind: str
    status: JobStatus
    source_resume_id: str | None = None
    source_resume_version: int | None = None
    current_version: int | None = None


@dataclass(frozen=True, slots=True)
class JobParseTask:
    """一次幂等解析尝试。"""

    task_id: str
    job_id: str
    user_id: str
    data_region: str
    idempotency_key: str
    status: JobStatus
    input_fingerprint: str
    impact: ParseImpact
    version: int | None = None
    message: str | None = None


@dataclass(frozen=True, slots=True)
class JobVersion:
    """追加式、不可变的岗位版本。"""

    job_id: str
    version: int
    data_region: str
    profile: JsonObject
    confirmed_by_user: bool


@dataclass(frozen=True, slots=True)
class CreateJobRequest:
    """粘贴 JD 并创建岗位。"""

    job_id: str
    user_id: str
    data_region: str
    language: str
    jd_text: str
    idempotency_key: str


@dataclass(frozen=True, slots=True)
class CreateInferredJobRequest:
    """从已确认的安全简历快照创建 AI 推导岗位。"""

    job_id: str
    resume_id: str
    resume_version: int
    user_id: str
    data_region: str
    language: str
    idempotency_key: str


@dataclass(frozen=True, slots=True)
class JobFieldEditRequest:
    """单字段人工校对命令。"""

    base_version: int
    path: str
    operation: str
    idempotency_key: str
    value: JsonValue = None


@dataclass(frozen=True, slots=True)
class MaterialReadiness:
    """展示给用户且写入同意快照的缺失材料影响。"""

    assessment_id: str
    user_id: str
    data_region: str
    mode: MaterialMode
    resume_id: str | None
    resume_version: int | None
    job_id: str | None
    job_version: int | None
    consent_required: bool
    modal_title: str
    modal_message: str
    effects: tuple[str, ...]
    allowed_scoring_dimensions: tuple[str, ...]
    input_fingerprint: str


@dataclass(frozen=True, slots=True)
class DegradedModeConsent:
    """追加式 ConsentGrant 语义记录；持久化由授权中心适配器负责。"""

    consent_grant_id: str
    assessment_id: str
    user_id: str
    data_region: str
    mode: MaterialMode
    accepted: bool
    effects: tuple[str, ...]
    idempotency_key: str

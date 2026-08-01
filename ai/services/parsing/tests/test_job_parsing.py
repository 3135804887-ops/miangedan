"""TASK-014 JD 解析、AI 推理校对、幂等重试和 FR-005 降级回归。"""

from __future__ import annotations

import json
import threading
from pathlib import Path
from typing import cast

import pytest

from mgd_parsing import (
    CreateInferredJobRequest,
    CreateJobRequest,
    ExplicitConsentRequiredError,
    InMemoryConfirmedMaterialReferenceValidator,
    InMemoryConfirmedResumeReader,
    InMemoryJobRawTextStore,
    InMemoryJobRepository,
    JobFieldEditRequest,
    JobNotConfirmedError,
    JobParsingService,
    JsonSchemaProfileValidator,
    MaterialReadinessService,
    RetryableProviderError,
    SyntheticJobParsingProvider,
    job_leakage_count,
)
from mgd_parsing.job_models import JobParseTask, JobProviderRequest, JobProviderResult, MaterialMode
from mgd_parsing.job_provider import JobParsingProvider
from mgd_parsing.job_service import JobFieldEditError
from mgd_parsing.models import JsonObject
from mgd_parsing.repository import IdempotencyConflictError

ROOT = Path(__file__).resolve().parents[4]
SCHEMA_PATH = ROOT / "ai" / "schemas" / "job-profile.schema.json"
FIXTURES = ROOT / "fixtures" / "synthetic" / "jobs"
EVAL_DATASET = ROOT / "ai" / "evals" / "datasets" / "job-parsing-governance.jsonl"
EVAL_EXPECTED = ROOT / "ai" / "evals" / "expected-results" / "job-parsing-governance.expected.json"


def schema() -> JsonObject:
    return cast(JsonObject, json.loads(SCHEMA_PATH.read_text(encoding="utf-8")))


def create_request(
    text: str,
    *,
    idem: str = "idem-job-create-0001",
    language: str = "zh-CN",
) -> CreateJobRequest:
    return CreateJobRequest(
        job_id="00000000-0000-7000-8000-000000000014",
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
        language=language,
        jd_text=text,
        idempotency_key=idem,
    )


def make_service(
    text: str,
    provider: JobParsingProvider | None = None,
    *,
    language: str = "zh-CN",
) -> tuple[JobParsingService, InMemoryJobRepository, InMemoryJobRawTextStore]:
    repository = InMemoryJobRepository()
    raw_store = InMemoryJobRawTextStore()
    counter = 0

    def new_id() -> str:
        nonlocal counter
        counter += 1
        return f"00000000-0000-7000-8000-{counter:012d}"

    service = JobParsingService(
        repository=repository,
        raw_text_store=raw_store,
        provider=provider or SyntheticJobParsingProvider(),
        validator=JsonSchemaProfileValidator(schema()),
        new_id=new_id,
    )
    service.create(create_request(text, language=language))
    return service, repository, raw_store


def parse_once(service: JobParsingService, *, idem: str = "idem-job-parse-0001") -> JobParseTask:
    return service.parse(
        job_id="00000000-0000-7000-8000-000000000014",
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
        idempotency_key=idem,
        trace_id="trace-task-014",
    )


def test_jd_normal_path_excludes_before_provider_and_marks_ai_inference() -> None:
    text = (FIXTURES / "jd-zh-01.md").read_text(encoding="utf-8")
    provider = SyntheticJobParsingProvider()
    service, repository, _ = make_service(text, provider)
    task = parse_once(service)
    assert task.status.value == "AWAITING_CONFIRMATION"
    version = service.get_version(task.job_id, 1)
    assert version.profile["excluded_from_scoring"] == [
        "recruiter_contact",
        "salary_benefits",
    ]
    focus = cast(list[dict[str, object]], version.profile["ai_inferred_interview_focus"])
    assert focus[0]["ai_inferred"] is True
    assert focus[0]["editable"] is True
    assert version.confirmed_by_user is False
    assert repository.version_count(version.job_id) == 1

    assert provider.last_request is not None
    data = next(
        message.content for message in provider.last_request.messages if message.layer == "data"
    )
    assert "hr@example.com" not in data
    assert "+86-21-0000-0000" not in data
    assert "薪资范围" not in data
    assert "<<<UNTRUSTED_JD_TEXT>>>" in data


def test_manual_edit_preserves_marker_and_confirmed_only_context_has_zero_leakage() -> None:
    service, _, _ = make_service((FIXTURES / "jd-zh-01.md").read_text(encoding="utf-8"))
    task = parse_once(service)
    job_id = task.job_id
    with pytest.raises(JobNotConfirmedError):
        service.build_interview_context(job_id=job_id, version=1)

    edited = service.edit_field(
        job_id=job_id,
        request=JobFieldEditRequest(
            base_version=1,
            path="/ai_inferred_interview_focus/0/focus",
            operation="replace",
            value="人工校对后的数据链路取舍",
            idempotency_key="idem-job-edit-0001",
        ),
    )
    focus = cast(list[dict[str, object]], edited.profile["ai_inferred_interview_focus"])[0]
    assert focus["ai_inferred"] is True
    assert focus["editable"] is True
    assert focus["edited_by_user"] is True
    duplicate_edit = service.edit_field(
        job_id=job_id,
        request=JobFieldEditRequest(
            base_version=1,
            path="/ai_inferred_interview_focus/0/focus",
            operation="replace",
            value="人工校对后的数据链路取舍",
            idempotency_key="idem-job-edit-0001",
        ),
    )
    assert duplicate_edit.version == edited.version
    with pytest.raises(JobFieldEditError, match="provenance markers are immutable"):
        service.edit_field(
            job_id=job_id,
            request=JobFieldEditRequest(
                base_version=edited.version,
                path="/ai_inferred_interview_focus/0/ai_inferred",
                operation="replace",
                value=False,
                idempotency_key="idem-job-edit-marker",
            ),
        )

    confirmed = service.confirm(
        job_id=job_id,
        base_version=edited.version,
        idempotency_key="idem-job-confirm-0001",
    )
    duplicate_confirm = service.confirm(
        job_id=job_id,
        base_version=edited.version,
        idempotency_key="idem-job-confirm-0001",
    )
    assert duplicate_confirm.version == confirmed.version
    context = service.build_interview_context(job_id=job_id, version=confirmed.version)
    scoring = service.build_scoring_material(job_id=job_id, version=confirmed.version)
    assert job_leakage_count(context) == 0
    assert job_leakage_count(scoring) == 0
    serialized = json.dumps({"context": context, "scoring": scoring}, ensure_ascii=False)
    assert "hr@example.com" not in serialized
    assert "薪资范围" not in serialized
    assert "excluded_from_scoring" not in context


class ThreeFailuresThenSuccess:
    """模拟一次内部重试耗尽后，外部步骤级重试成功。"""

    def __init__(self) -> None:
        self.calls = 0
        self.delegate = SyntheticJobParsingProvider()

    def parse_job(self, request: JobProviderRequest) -> JobProviderResult:
        self.calls += 1
        if self.calls <= 3:
            raise RetryableProviderError("synthetic timeout")
        return self.delegate.parse_job(request)


class LeakyJobProvider:
    """模拟适配器错误带回应排除的根字段。"""

    def __init__(self) -> None:
        self.delegate = SyntheticJobParsingProvider()

    def parse_job(self, request: JobProviderRequest) -> JobProviderResult:
        result = self.delegate.parse_job(request)
        fields = dict(result.profile_fields)
        fields["salary"] = "synthetic excluded salary value"
        fields["recruiter_contact"] = "synthetic-recruiter@example.invalid"
        return JobProviderResult(
            profile_fields=fields,
            field_confidences=result.field_confidences,
            provider_version="synthetic-leaky-job-parser/v1",
            injection_detected=result.injection_detected,
        )


def test_provider_output_is_sanitized_before_schema_and_version_write() -> None:
    service, _, _ = make_service(
        (FIXTURES / "jd-en-01.md").read_text(encoding="utf-8"),
        LeakyJobProvider(),
        language="en-US",
    )
    task = parse_once(service)
    profile = service.get_version(task.job_id, 1).profile
    assert "salary" not in profile
    assert "recruiter_contact" not in profile
    assert job_leakage_count(profile) == 0


def test_timeout_retains_original_and_retry_is_idempotent() -> None:
    provider = ThreeFailuresThenSuccess()
    service, repository, raw_store = make_service(
        (FIXTURES / "jd-zh-01.md").read_text(encoding="utf-8"), provider
    )
    failed = parse_once(service)
    assert failed.status.value == "RETRYABLE_FAILURE"
    assert failed.impact.original_input_retained is True
    assert failed.impact.retryable is True
    assert raw_store.read(
        job_id=failed.job_id,
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
    )

    completed = parse_once(service, idem="idem-job-parse-retry")
    duplicate = parse_once(service, idem="idem-job-parse-retry")
    assert completed.status.value == "AWAITING_CONFIRMATION"
    assert duplicate.task_id == completed.task_id
    assert provider.calls == 4
    assert repository.version_count(completed.job_id) == 1


def test_create_concurrent_idempotency_has_one_job_and_conflicting_payload_is_rejected() -> None:
    text = (FIXTURES / "jd-zh-01.md").read_text(encoding="utf-8")
    repository = InMemoryJobRepository()
    raw_store = InMemoryJobRawTextStore()
    service = JobParsingService(
        repository=repository,
        raw_text_store=raw_store,
        provider=SyntheticJobParsingProvider(),
        validator=JsonSchemaProfileValidator(schema()),
    )
    results: list[str] = []

    def create() -> None:
        results.append(service.create(create_request(text)).job_id)

    threads = [threading.Thread(target=create) for _ in range(8)]
    for thread in threads:
        thread.start()
    for thread in threads:
        thread.join()
    assert len(set(results)) == 1
    with pytest.raises(IdempotencyConflictError):
        service.create(create_request(text + "\nsynthetic change"))


def test_injection_is_data_and_governance_eval_has_zero_leakage() -> None:
    rows = [json.loads(line) for line in EVAL_DATASET.read_text(encoding="utf-8").splitlines()]
    expected = json.loads(EVAL_EXPECTED.read_text(encoding="utf-8"))
    passed = 0
    detected = 0
    for index, row in enumerate(rows):
        case_expected = expected[row["case_id"]]
        assert row["synthetic"] is True
        provider = SyntheticJobParsingProvider()
        service, _, _ = make_service(
            (ROOT / cast(str, row["fixture"])).read_text(encoding="utf-8"),
            provider,
            language=cast(str, row["language"]),
        )
        task = parse_once(service, idem=f"idem-eval-{index:04d}")
        profile = service.get_version(task.job_id, 1).profile
        assert job_leakage_count(profile) == case_expected["excluded_content_leakage_count"]
        assert set(cast(list[str], profile["excluded_from_scoring"])) >= set(
            case_expected["required_excluded_categories"]
        )
        focus = cast(list[dict[str, object]], profile["ai_inferred_interview_focus"])
        assert all(item["ai_inferred"] is True and item["editable"] is True for item in focus)
        parse_meta = cast(dict[str, object], profile["parse_meta"])
        if row.get("expect_injection_detected"):
            assert parse_meta["injection_detected"] is True
            detected += 1
        passed += 1
    assert passed == len(expected)
    assert detected == 1


def test_resume_only_generates_fully_marked_editable_job_profile() -> None:
    repository = InMemoryJobRepository()
    raw_store = InMemoryJobRawTextStore()
    resume_reader = InMemoryConfirmedResumeReader()
    resume_reader.add(
        resume_id="00000000-0000-7000-8000-000000000013",
        version=3,
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
        profile={"skills": [{"name": "Python"}, {"name": "SQL"}]},
    )
    service = JobParsingService(
        repository=repository,
        raw_text_store=raw_store,
        confirmed_resume_reader=resume_reader,
        provider=SyntheticJobParsingProvider(),
        validator=JsonSchemaProfileValidator(schema()),
    )
    job = service.create_from_resume(
        CreateInferredJobRequest(
            job_id="00000000-0000-7000-8000-000000000014",
            resume_id="00000000-0000-7000-8000-000000000013",
            resume_version=3,
            user_id="00000000-0000-7000-8000-000000000001",
            data_region="cn",
            language="zh-CN",
            idempotency_key="idem-inferred-create",
        )
    )
    task = service.parse(
        job_id=job.job_id,
        user_id=job.user_id,
        data_region=job.data_region,
        idempotency_key="idem-inferred-parse",
        trace_id="trace-inferred",
    )
    profile = service.get_version(task.job_id, 1).profile
    assert profile["source_kind"] == "resume_inference"
    derived = cast(list[str], profile["ai_derived_fields"])
    assert "/job_title" in derived
    requirements = cast(list[dict[str, object]], profile["requirements"])
    assert all(item["ai_inferred"] is True for item in requirements)
    edited = service.edit_field(
        job_id=job.job_id,
        request=JobFieldEditRequest(
            base_version=1,
            path="/job_title",
            operation="replace",
            value="人工确认的数据工程师",
            idempotency_key="idem-inferred-edit",
        ),
    )
    assert edited.profile["job_title"] == "人工确认的数据工程师"
    assert "/job_title" in cast(list[str], edited.profile["ai_derived_fields"])


def make_readiness_service() -> MaterialReadinessService:
    counter = 100

    def new_id() -> str:
        nonlocal counter
        counter += 1
        return f"00000000-0000-7000-8000-{counter:012d}"

    material_validator = InMemoryConfirmedMaterialReferenceValidator()
    material_validator.add_resume(
        resume_id="00000000-0000-7000-8000-000000000013",
        version=1,
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
    )
    material_validator.add_job(
        job_id="00000000-0000-7000-8000-000000000014",
        version=1,
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
    )
    return MaterialReadinessService(
        repository=InMemoryJobRepository(),
        material_validator=material_validator,
        new_id=new_id,
    )


@pytest.mark.parametrize(
    ("resume_version", "job_version", "mode", "consent_required"),
    [
        (1, 1, MaterialMode.FULL, False),
        (None, 1, MaterialMode.JD_ONLY, True),
        (1, None, MaterialMode.RESUME_ONLY, True),
        (None, None, MaterialMode.NEITHER, True),
    ],
)
def test_material_modes_return_exact_impact_modal(
    resume_version: int | None,
    job_version: int | None,
    mode: MaterialMode,
    consent_required: bool,
) -> None:
    service = make_readiness_service()
    assessment = service.assess(
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
        resume_id=("00000000-0000-7000-8000-000000000013" if resume_version else None),
        resume_version=resume_version,
        job_id=("00000000-0000-7000-8000-000000000014" if job_version else None),
        job_version=job_version,
        idempotency_key=f"idem-mode-{mode.value}",
    )
    assert assessment.mode is mode
    assert assessment.consent_required is consent_required
    assert assessment.modal_title
    assert assessment.modal_message
    if mode is MaterialMode.NEITHER:
        assert assessment.allowed_scoring_dimensions == (
            "expression",
            "logic",
            "communication",
            "adaptability",
        )
    if mode is MaterialMode.JD_ONLY:
        assert "不虚构候选人经历" in assessment.effects
        assert "不生成经历匹配评分" in assessment.effects
    if mode is MaterialMode.RESUME_ONLY:
        assert "岗位画像为 AI 推导" in assessment.effects
        assert "推理字段需人工校对确认" in assessment.effects


def test_degraded_mode_blocks_without_explicit_matching_consent_and_is_idempotent() -> None:
    service = make_readiness_service()
    assessment = service.assess(
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
        resume_id=None,
        resume_version=None,
        job_id="00000000-0000-7000-8000-000000000014",
        job_version=1,
        idempotency_key="idem-assess-jd-only",
    )
    with pytest.raises(ExplicitConsentRequiredError):
        service.require_may_continue(assessment_id=assessment.assessment_id, consent_grant_id=None)
    with pytest.raises(ExplicitConsentRequiredError):
        service.grant(
            assessment_id=assessment.assessment_id,
            user_id=assessment.user_id,
            data_region=assessment.data_region,
            accepted=False,
            idempotency_key="idem-consent-rejected",
        )
    consent = service.grant(
        assessment_id=assessment.assessment_id,
        user_id=assessment.user_id,
        data_region=assessment.data_region,
        accepted=True,
        idempotency_key="idem-consent-accepted",
    )
    duplicate = service.grant(
        assessment_id=assessment.assessment_id,
        user_id=assessment.user_id,
        data_region=assessment.data_region,
        accepted=True,
        idempotency_key="idem-consent-accepted",
    )
    assert duplicate.consent_grant_id == consent.consent_grant_id
    assert (
        service.require_may_continue(
            assessment_id=assessment.assessment_id,
            consent_grant_id=consent.consent_grant_id,
        ).mode
        is MaterialMode.JD_ONLY
    )
    other = service.assess(
        user_id=assessment.user_id,
        data_region=assessment.data_region,
        resume_id=None,
        resume_version=None,
        job_id=None,
        job_version=None,
        idempotency_key="idem-assess-neither",
    )
    with pytest.raises(ExplicitConsentRequiredError, match="does not match"):
        service.require_may_continue(
            assessment_id=other.assessment_id,
            consent_grant_id=consent.consent_grant_id,
        )

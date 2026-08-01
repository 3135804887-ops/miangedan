"""TASK-013 简历解析、逐字段校对、幂等/重试与 SEC-040 回归。"""

from __future__ import annotations

import json
import threading
from dataclasses import replace
from pathlib import Path
from typing import cast

import pytest

from mgd_parsing import (
    AcceptedResumeText,
    FieldEditRequest,
    InMemoryAcceptedUploadReader,
    InMemoryParsingRepository,
    JsonSchemaProfileValidator,
    ResumeParsingService,
    RetryableProviderError,
    StartParseRequest,
    SyntheticResumeParsingProvider,
    leakage_count,
)
from mgd_parsing.models import JsonObject, ResumeProviderRequest, ResumeProviderResult
from mgd_parsing.privacy import SensitiveContentError
from mgd_parsing.provider import PermanentProviderError, ResumeParsingProvider
from mgd_parsing.repository import IdempotencyConflictError, VersionConflictError
from mgd_parsing.service import LowConfidenceUnresolvedError, ResumeNotConfirmedError
from mgd_parsing.uploads import AcceptedUploadError

ROOT = Path(__file__).resolve().parents[4]
SCHEMA_PATH = ROOT / "ai" / "schemas" / "resume-profile.schema.json"
FIXTURES = ROOT / "fixtures" / "synthetic" / "resumes"
EVAL_DATASET = ROOT / "ai" / "evals" / "datasets" / "resume-parsing-security.jsonl"
EVAL_EXPECTED = ROOT / "ai" / "evals" / "expected-results" / "resume-parsing-security.expected.json"


def schema() -> JsonObject:
    """读取仓内唯一 resume-profile Schema。"""
    return cast(JsonObject, json.loads(SCHEMA_PATH.read_text(encoding="utf-8")))


def start_request(*, idem: str = "idem-parse-0001", language: str = "zh-CN") -> StartParseRequest:
    """创建无真实个人信息的测试命令。"""
    return StartParseRequest(
        resume_id="00000000-0000-7000-8000-000000000013",
        upload_id="00000000-0000-7000-8000-000000000012",
        user_id="00000000-0000-7000-8000-000000000001",
        data_region="cn",
        language=language,
        idempotency_key=idem,
        trace_id="trace-task-013",
    )


def make_service(
    text: str,
    provider: ResumeParsingProvider | None = None,
) -> tuple[ResumeParsingService, InMemoryParsingRepository, InMemoryAcceptedUploadReader]:
    """组装全部供应商中立/内存适配器。"""
    request = start_request()
    uploads = InMemoryAcceptedUploadReader()
    uploads.add(
        AcceptedResumeText(
            upload_id=request.upload_id,
            user_id=request.user_id,
            data_region=request.data_region,
            text=text,
        )
    )
    repository = InMemoryParsingRepository()
    counter = 0

    def new_id() -> str:
        nonlocal counter
        counter += 1
        return f"00000000-0000-7000-8000-{counter:012d}"

    service = ResumeParsingService(
        repository=repository,
        uploads=uploads,
        provider=provider or SyntheticResumeParsingProvider(),
        validator=JsonSchemaProfileValidator(schema()),
        new_id=new_id,
    )
    return service, repository, uploads


def resolve_low_confidence(service: ResumeParsingService, resume_id: str, version: int) -> int:
    """逐字段确认所有低置信度路径，返回最新草稿版本。"""
    current = service.get_version(resume_id, version)
    parse_meta = cast(dict[str, object], current.profile["parse_meta"])
    paths = cast(list[str], parse_meta["low_confidence_paths"])
    for index, path in enumerate(paths):
        current = service.edit_field(
            resume_id=resume_id,
            request=FieldEditRequest(
                base_version=current.version,
                path=path,
                operation="confirm",
                idempotency_key=f"idem-field-{index:04d}",
            ),
        )
    return current.version


def test_parse_normal_path_redacts_before_provider_and_marks_low_confidence() -> None:
    """正常路径：结构化、低置信度、Schema 与解析前脱敏全部生效。"""
    text = (FIXTURES / "resume-zh-01.md").read_text(encoding="utf-8")
    provider = SyntheticResumeParsingProvider()
    service, repository, _ = make_service(text, provider)
    task = service.start(start_request())

    assert task.status.value == "AWAITING_CONFIRMATION"
    assert task.version == 1
    version = service.get_version(task.resume_id, 1)
    assert version.confirmed_by_user is False
    parse_meta = cast(dict[str, object], version.profile["parse_meta"])
    assert cast(list[str], parse_meta["low_confidence_paths"])
    assert version.profile["excluded_sensitive_fields"] == ["address", "email", "phone"]
    assert leakage_count(version.profile) == 0
    assert repository.version_count(task.resume_id) == 1

    assert provider.last_request is not None
    data = next(
        message.content for message in provider.last_request.messages if message.layer == "data"
    )
    assert "<<<UNTRUSTED_RESUME_TEXT>>>" in data
    assert "xiaozhou.lin@example.com" not in data
    assert "+86-138-0000-0000" not in data
    assert "虚构路 100 号" not in data


def test_low_confidence_requires_per_field_review_and_versions_are_append_only() -> None:
    """异常/正常路径：未逐项校对禁止确认，操作逐次追加版本且幂等。"""
    service, repository, _ = make_service(
        (FIXTURES / "resume-zh-01.md").read_text(encoding="utf-8")
    )
    task = service.start(start_request())
    with pytest.raises(LowConfidenceUnresolvedError):
        service.confirm(
            resume_id=task.resume_id,
            base_version=1,
            idempotency_key="idem-confirm-blocked",
        )

    latest = resolve_low_confidence(service, task.resume_id, 1)
    confirmed = service.confirm(
        resume_id=task.resume_id,
        base_version=latest,
        idempotency_key="idem-confirm-success",
    )
    assert confirmed.confirmed_by_user is True
    assert confirmed.version == repository.version_count(task.resume_id)
    duplicate = service.confirm(
        resume_id=task.resume_id,
        base_version=confirmed.version,
        idempotency_key="idem-confirm-success",
    )
    assert duplicate.version == confirmed.version

    with pytest.raises(VersionConflictError):
        service.edit_field(
            resume_id=task.resume_id,
            request=FieldEditRequest(
                base_version=1,
                path="/display_name",
                operation="replace",
                value="候选人",
                idempotency_key="idem-stale-edit",
            ),
        )


def test_sensitive_fixture_has_zero_leakage_in_context_and_scoring_material() -> None:
    """SEC-040：电话/邮箱/证件/地址/照片/保护属性进入两类材料的命中数均为 0。"""
    text = (FIXTURES / "resume-sensitive-zero-leak.md").read_text(encoding="utf-8")
    service, _, _ = make_service(text)
    task = service.start(start_request())
    draft = service.get_version(task.resume_id, 1)
    assert set(cast(list[str], draft.profile["excluded_sensitive_fields"])) == {
        "phone",
        "email",
        "id_number",
        "address",
        "photo",
        "protected_attribute",
    }
    with pytest.raises(ResumeNotConfirmedError):
        service.build_interview_context(resume_id=task.resume_id, version=1)
    latest = resolve_low_confidence(service, task.resume_id, 1)
    confirmed = service.confirm(
        resume_id=task.resume_id,
        base_version=latest,
        idempotency_key="idem-sensitive-confirm",
    )
    context = service.build_interview_context(resume_id=task.resume_id, version=confirmed.version)
    scoring = service.build_scoring_material(resume_id=task.resume_id, version=confirmed.version)
    assert leakage_count(context) == 0
    assert leakage_count(scoring) == 0
    serialized = json.dumps({"context": context, "scoring": scoring}, ensure_ascii=False)
    for forbidden in (
        "example.invalid",
        "SYNTHETIC-ID-NOT-VALID",
        "虚构省虚构市",
        "测试保护属性值",
    ):
        assert forbidden not in serialized


class LeakyProvider:
    """模拟供应商把敏感键和值错误带回。"""

    def parse_resume(self, request: ResumeProviderRequest) -> ResumeProviderResult:
        del request
        return ResumeProviderResult(
            profile_fields={
                "display_name": "候选人",
                "email": "leak@example.invalid",
                "gender": "synthetic protected value",
                "work_experience": [
                    {
                        "organization": "合成组织",
                        "role": "工程师",
                        "responsibilities": ["联系邮箱 leak@example.invalid"],
                        "confidence": 0.9,
                    }
                ],
                "education": [],
                "projects": [],
                "skills": [],
                "languages": [],
                "certifications": [],
                "awards": [],
                "publications": [],
                "portfolio_links": [],
                "interview_clues": [],
            },
            field_confidences={"/display_name": 0.9},
            provider_version="synthetic-leaky/v1",
        )


def test_provider_output_and_user_edits_cannot_bypass_privacy_gate() -> None:
    """异常路径：模型回传与人工编辑中的敏感键/值均无法落入新版本。"""
    text = (FIXTURES / "resume-sensitive-zero-leak.md").read_text(encoding="utf-8")
    service, repository, _ = make_service(text, LeakyProvider())
    task = service.start(start_request())
    profile = service.get_version(task.resume_id, 1).profile
    assert leakage_count(profile) == 0
    assert "email" not in profile and "gender" not in profile
    assert repository.version_count(task.resume_id) == 1

    with pytest.raises(SensitiveContentError):
        service.edit_field(
            resume_id=task.resume_id,
            request=FieldEditRequest(
                base_version=1,
                path="/display_name",
                operation="replace",
                value="contact@example.invalid",
                idempotency_key="idem-sensitive-edit",
            ),
        )
    assert repository.version_count(task.resume_id) == 1


class TimeoutThenSuccessProvider(SyntheticResumeParsingProvider):
    """首轮三次（初次 + 两次自动重试）暂时失败，外部重试随后成功。"""

    def parse_resume(self, request: ResumeProviderRequest) -> ResumeProviderResult:
        if self.calls < 3:
            self.calls += 1
            self.last_request = request
            raise RetryableProviderError("synthetic timeout")
        return super().parse_resume(request)


class PermanentlyUnavailableProvider:
    """模拟不可重试的供应商配置错误。"""

    def parse_resume(self, request: ResumeProviderRequest) -> ResumeProviderResult:
        del request
        raise PermanentProviderError("synthetic invalid provider configuration")


def test_permanent_provider_failure_stops_safely_without_partial_version() -> None:
    """异常路径：不可重试错误进入稳定 FAILED，保留原件且不写半成品版本。"""
    text = (FIXTURES / "resume-zh-01.md").read_text(encoding="utf-8")
    service, repository, _ = make_service(text, PermanentlyUnavailableProvider())
    task = service.start(start_request())
    assert task.status.value == "FAILED"
    assert task.impact.original_input_retained is True
    assert task.impact.retryable is False
    assert task.impact.billable is False
    assert task.impact.scoring_affected is False
    assert repository.version_count(task.resume_id) == 0


def test_timeout_retains_input_retry_is_idempotent_and_region_isolated() -> None:
    """NFR-015：超时保留输入，只重试失败步骤；重复重试无副作用且跨区拒绝。"""
    text = (FIXTURES / "resume-zh-01.md").read_text(encoding="utf-8")
    provider = TimeoutThenSuccessProvider()
    service, _, uploads = make_service(text, provider)
    task = service.start(start_request())
    assert task.status.value == "RETRYABLE_FAILURE"
    assert task.impact.original_input_retained is True
    assert task.impact.retryable is True
    assert task.impact.billable is False
    assert task.impact.scoring_affected is False

    retried = service.retry(
        resume_id=task.resume_id,
        idempotency_key="idem-retry-0001",
        trace_id="trace-retry",
    )
    assert retried.status.value == "AWAITING_CONFIRMATION"
    duplicate = service.retry(
        resume_id=task.resume_id,
        idempotency_key="idem-retry-0001",
        trace_id="trace-retry-duplicate",
    )
    assert duplicate == retried
    assert provider.calls == 4

    request = start_request(idem="idem-cross-region")
    with pytest.raises(AcceptedUploadError, match="unavailable"):
        uploads.read_accepted_resume(
            upload_id=request.upload_id,
            user_id=request.user_id,
            data_region="eu",
        )


def test_concurrent_parse_idempotency_executes_provider_once() -> None:
    """并发幂等：同一命令只有一个供应商调用和一个初始版本。"""
    text = (FIXTURES / "resume-zh-01.md").read_text(encoding="utf-8")
    provider = SyntheticResumeParsingProvider()
    service, repository, _ = make_service(text, provider)
    barrier = threading.Barrier(8)
    results: list[object] = []

    def worker() -> None:
        barrier.wait()
        results.append(service.start(start_request()))

    threads = [threading.Thread(target=worker) for _ in range(8)]
    for thread in threads:
        thread.start()
    for thread in threads:
        thread.join()

    assert len(results) == 8
    assert provider.calls == 1
    assert repository.version_count(start_request().resume_id) == 1

    with pytest.raises(IdempotencyConflictError):
        service.start(
            replace(
                start_request(),
                upload_id="00000000-0000-7000-8000-999999999999",
            )
        )


def test_resume_parsing_security_eval_dataset() -> None:
    """运行 TASK-013 对应 AI 评测集，硬断言隐私、注入中和与低置信度阻断。"""
    expected_all = cast(
        dict[str, dict[str, object]],
        json.loads(EVAL_EXPECTED.read_text(encoding="utf-8")),
    )
    cases = [json.loads(line) for line in EVAL_DATASET.read_text(encoding="utf-8").splitlines()]
    for raw_case in cases:
        item = cast(dict[str, object], raw_case)
        case_id = cast(str, item["case_id"])
        expected = expected_all[case_id]
        input_data = cast(dict[str, str], item["input"])
        text = (ROOT / input_data["fixture"]).read_text(encoding="utf-8")
        language = cast(str, item["language"])
        service, _, _ = make_service(text)
        task = service.start(start_request(language=language))
        version = service.get_version(task.resume_id, 1)

        if case_id == "resume-privacy-zero-zh":
            assert leakage_count(version.profile) == expected["leakage_count"]
            actual = set(cast(list[str], version.profile["excluded_sensitive_fields"]))
            assert actual.issuperset(cast(list[str], expected["required_excluded_categories"]))
        elif case_id == "resume-injection-en":
            parse_meta = cast(dict[str, object], version.profile["parse_meta"])
            assert parse_meta["injection_detected"] is expected["injection_detected"]
            serialized = json.dumps(version.profile, ensure_ascii=False).lower()
            for forbidden in cast(list[str], expected["must_not_include"]):
                assert forbidden not in serialized
        elif case_id == "resume-low-confidence-zh":
            parse_meta = cast(dict[str, object], version.profile["parse_meta"])
            paths = cast(list[str], parse_meta["low_confidence_paths"])
            assert len(paths) >= cast(int, expected["minimum_low_confidence_paths"])
            with pytest.raises(LowConfidenceUnresolvedError):
                service.confirm(
                    resume_id=task.resume_id,
                    base_version=1,
                    idempotency_key="idem-eval-confirm",
                )

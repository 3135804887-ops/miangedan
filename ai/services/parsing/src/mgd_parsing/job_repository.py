"""JD 原文、解析任务、追加式版本与降级同意的持久化边界。"""

from __future__ import annotations

import copy
import threading
from dataclasses import replace
from typing import Protocol

from .job_models import (
    DegradedModeConsent,
    JobParseTask,
    JobRecord,
    JobStatus,
    JobVersion,
    MaterialReadiness,
)
from .models import JsonObject
from .repository import IdempotencyConflictError, VersionConflictError


class JobNotFoundError(LookupError):
    """岗位不存在或不属于调用者。"""


class JobVersionNotFoundError(LookupError):
    """岗位版本不存在。"""


class MaterialAssessmentNotFoundError(LookupError):
    """材料影响快照不存在。"""


class JobRawTextStore(Protocol):
    """JD 原文受限存储；不得写日志或模型状态。"""

    def put(self, *, job_id: str, user_id: str, data_region: str, text: str) -> None:
        """在所属数据区持久化粘贴原文。"""

    def read(self, *, job_id: str, user_id: str, data_region: str) -> str:
        """仅向所属用户与区域的解析步骤返回原文。"""


class ConfirmedResumeReader(Protocol):
    """只读 TASK-013 已确认且通过 SEC-040 的安全简历快照。"""

    def read_confirmed(
        self,
        *,
        resume_id: str,
        version: int,
        user_id: str,
        data_region: str,
    ) -> JsonObject:
        """未确认、越权或跨区时必须拒绝。"""


class ConfirmedMaterialReferenceValidator(Protocol):
    """材料影响评估前验证版本已确认且属于当前用户/区域。"""

    def validate_resume(
        self, *, resume_id: str, version: int, user_id: str, data_region: str
    ) -> None: ...

    def validate_job(
        self, *, job_id: str, version: int, user_id: str, data_region: str
    ) -> None: ...


class InMemoryJobRawTextStore:
    """合成测试用受限原文存储适配器。"""

    def __init__(self) -> None:
        self._items: dict[str, tuple[str, str, str]] = {}
        self._lock = threading.Lock()

    def put(self, *, job_id: str, user_id: str, data_region: str, text: str) -> None:
        with self._lock:
            existing = self._items.get(job_id)
            candidate = (user_id, data_region, text)
            if existing is not None and existing != candidate:
                raise IdempotencyConflictError("job raw text is immutable")
            self._items[job_id] = candidate

    def read(self, *, job_id: str, user_id: str, data_region: str) -> str:
        with self._lock:
            item = self._items.get(job_id)
            if item is None or item[0] != user_id or item[1] != data_region:
                raise JobNotFoundError(job_id)
            return item[2]


class InMemoryConfirmedResumeReader:
    """合成测试用已确认安全简历读取器。"""

    def __init__(self) -> None:
        self._items: dict[tuple[str, int], tuple[str, str, JsonObject]] = {}
        self._lock = threading.Lock()

    def add(
        self,
        *,
        resume_id: str,
        version: int,
        user_id: str,
        data_region: str,
        profile: JsonObject,
    ) -> None:
        with self._lock:
            self._items[(resume_id, version)] = (
                user_id,
                data_region,
                copy.deepcopy(profile),
            )

    def read_confirmed(
        self,
        *,
        resume_id: str,
        version: int,
        user_id: str,
        data_region: str,
    ) -> JsonObject:
        with self._lock:
            item = self._items.get((resume_id, version))
            if item is None or item[0] != user_id or item[1] != data_region:
                raise JobNotFoundError(f"confirmed resume {resume_id}:{version}")
            return copy.deepcopy(item[2])


class InMemoryConfirmedMaterialReferenceValidator:
    """合成测试用已确认材料引用校验器。"""

    def __init__(self) -> None:
        self._resumes: set[tuple[str, int, str, str]] = set()
        self._jobs: set[tuple[str, int, str, str]] = set()

    def add_resume(self, *, resume_id: str, version: int, user_id: str, data_region: str) -> None:
        self._resumes.add((resume_id, version, user_id, data_region))

    def add_job(self, *, job_id: str, version: int, user_id: str, data_region: str) -> None:
        self._jobs.add((job_id, version, user_id, data_region))

    def validate_resume(
        self, *, resume_id: str, version: int, user_id: str, data_region: str
    ) -> None:
        if (resume_id, version, user_id, data_region) not in self._resumes:
            raise JobNotFoundError(f"confirmed resume {resume_id}:{version}")

    def validate_job(self, *, job_id: str, version: int, user_id: str, data_region: str) -> None:
        if (job_id, version, user_id, data_region) not in self._jobs:
            raise JobNotFoundError(f"confirmed job {job_id}:{version}")


class JobRepository(Protocol):
    """岗位与降级记录的持久化端口；生产适配器必须保持追加式语义。"""

    def create_job(
        self,
        *,
        candidate: JobRecord,
        idempotency_key: str,
        input_fingerprint: str,
    ) -> tuple[JobRecord, bool]: ...

    def get_job(self, *, job_id: str, user_id: str, data_region: str) -> JobRecord: ...

    def claim_parse(self, candidate: JobParseTask) -> tuple[JobParseTask, bool]: ...

    def save_task(self, task: JobParseTask) -> None: ...

    def set_current_version(self, *, job_id: str, version: int, status: JobStatus) -> JobRecord: ...

    def append_version(
        self,
        *,
        job_id: str,
        data_region: str,
        profile: JsonObject,
        confirmed_by_user: bool,
        expected_base: int | None,
        idempotency_key: str,
        operation_fingerprint: str,
    ) -> tuple[JobVersion, bool]: ...

    def get_version(self, job_id: str, version: int) -> JobVersion: ...

    def latest_version(self, job_id: str) -> JobVersion: ...

    def create_assessment(
        self,
        *,
        candidate: MaterialReadiness,
        idempotency_key: str,
    ) -> tuple[MaterialReadiness, bool]: ...

    def get_assessment(self, assessment_id: str) -> MaterialReadiness: ...

    def append_consent(
        self,
        *,
        candidate: DegradedModeConsent,
        operation_fingerprint: str,
    ) -> tuple[DegradedModeConsent, bool]: ...

    def get_consent(self, consent_id: str) -> DegradedModeConsent: ...


class InMemoryJobRepository:
    """供合成测试和本地开发使用的并发安全适配器。"""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._jobs: dict[str, JobRecord] = {}
        self._create_idempotency: dict[tuple[str, str, str], tuple[str, str]] = {}
        self._tasks: dict[tuple[str, str], JobParseTask] = {}
        self._versions: dict[str, list[JobVersion]] = {}
        self._version_idempotency: dict[tuple[str, str], tuple[str, int]] = {}
        self._assessments: dict[str, MaterialReadiness] = {}
        self._assessment_idempotency: dict[tuple[str, str, str], tuple[str, str]] = {}
        self._consents: dict[str, DegradedModeConsent] = {}
        self._consent_idempotency: dict[tuple[str, str, str], tuple[str, str]] = {}

    def create_job(
        self,
        *,
        candidate: JobRecord,
        idempotency_key: str,
        input_fingerprint: str,
    ) -> tuple[JobRecord, bool]:
        key = (candidate.data_region, candidate.user_id, idempotency_key)
        with self._lock:
            existing = self._create_idempotency.get(key)
            if existing is not None:
                fingerprint, job_id = existing
                if fingerprint != input_fingerprint:
                    raise IdempotencyConflictError(
                        "job idempotency key reused with different immutable input"
                    )
                return self._jobs[job_id], False
            if candidate.job_id in self._jobs:
                raise IdempotencyConflictError("job already exists")
            self._jobs[candidate.job_id] = candidate
            self._create_idempotency[key] = (input_fingerprint, candidate.job_id)
            return candidate, True

    def get_job(self, *, job_id: str, user_id: str, data_region: str) -> JobRecord:
        with self._lock:
            job = self._jobs.get(job_id)
            if job is None or job.user_id != user_id or job.data_region != data_region:
                raise JobNotFoundError(job_id)
            return job

    def claim_parse(self, candidate: JobParseTask) -> tuple[JobParseTask, bool]:
        key = (candidate.job_id, candidate.idempotency_key)
        with self._lock:
            existing = self._tasks.get(key)
            if existing is not None:
                if existing.input_fingerprint != candidate.input_fingerprint:
                    raise IdempotencyConflictError(
                        "parse idempotency key reused with different immutable input"
                    )
                return existing, False
            job = self._jobs.get(candidate.job_id)
            if job is None:
                raise JobNotFoundError(candidate.job_id)
            if job.status not in {JobStatus.CREATED, JobStatus.RETRYABLE_FAILURE}:
                raise RuntimeError("job is not available for parsing")
            self._tasks[key] = candidate
            self._jobs[candidate.job_id] = replace(job, status=JobStatus.PARSING)
            return candidate, True

    def save_task(self, task: JobParseTask) -> None:
        key = (task.job_id, task.idempotency_key)
        with self._lock:
            if key not in self._tasks:
                raise JobNotFoundError(task.job_id)
            self._tasks[key] = task
            job = self._jobs[task.job_id]
            self._jobs[task.job_id] = replace(
                job,
                status=task.status,
                current_version=task.version or job.current_version,
            )

    def set_current_version(self, *, job_id: str, version: int, status: JobStatus) -> JobRecord:
        """更新可变聚合根指针；不可变版本本身不做 UPDATE。"""
        with self._lock:
            job = self._jobs.get(job_id)
            if job is None:
                raise JobNotFoundError(job_id)
            updated = replace(job, current_version=version, status=status)
            self._jobs[job_id] = updated
            return updated

    def append_version(
        self,
        *,
        job_id: str,
        data_region: str,
        profile: JsonObject,
        confirmed_by_user: bool,
        expected_base: int | None,
        idempotency_key: str,
        operation_fingerprint: str,
    ) -> tuple[JobVersion, bool]:
        idem_key = (job_id, idempotency_key)
        with self._lock:
            existing = self._version_idempotency.get(idem_key)
            if existing is not None:
                fingerprint, version_number = existing
                if fingerprint != operation_fingerprint:
                    raise IdempotencyConflictError(
                        "version idempotency key reused with different operation"
                    )
                return self._versions[job_id][version_number - 1], False
            versions = self._versions.setdefault(job_id, [])
            latest = len(versions)
            if expected_base is None and latest != 0:
                raise VersionConflictError("initial job version already exists")
            if expected_base is not None and expected_base != latest:
                raise VersionConflictError(
                    f"expected base version {expected_base}, current version is {latest}"
                )
            version = JobVersion(
                job_id=job_id,
                version=latest + 1,
                data_region=data_region,
                profile=copy.deepcopy(profile),
                confirmed_by_user=confirmed_by_user,
            )
            versions.append(version)
            self._version_idempotency[idem_key] = (operation_fingerprint, version.version)
            return version, True

    def get_version(self, job_id: str, version: int) -> JobVersion:
        with self._lock:
            versions = self._versions.get(job_id, [])
            if version < 1 or version > len(versions):
                raise JobVersionNotFoundError(f"{job_id}:{version}")
            return versions[version - 1]

    def latest_version(self, job_id: str) -> JobVersion:
        with self._lock:
            versions = self._versions.get(job_id, [])
            if not versions:
                raise JobVersionNotFoundError(job_id)
            return versions[-1]

    def version_count(self, job_id: str) -> int:
        with self._lock:
            return len(self._versions.get(job_id, []))

    def create_assessment(
        self,
        *,
        candidate: MaterialReadiness,
        idempotency_key: str,
    ) -> tuple[MaterialReadiness, bool]:
        key = (candidate.data_region, candidate.user_id, idempotency_key)
        with self._lock:
            existing = self._assessment_idempotency.get(key)
            if existing is not None:
                fingerprint, assessment_id = existing
                if fingerprint != candidate.input_fingerprint:
                    raise IdempotencyConflictError(
                        "assessment idempotency key reused with different materials"
                    )
                return self._assessments[assessment_id], False
            self._assessments[candidate.assessment_id] = candidate
            self._assessment_idempotency[key] = (
                candidate.input_fingerprint,
                candidate.assessment_id,
            )
            return candidate, True

    def get_assessment(self, assessment_id: str) -> MaterialReadiness:
        with self._lock:
            assessment = self._assessments.get(assessment_id)
            if assessment is None:
                raise MaterialAssessmentNotFoundError(assessment_id)
            return assessment

    def append_consent(
        self,
        *,
        candidate: DegradedModeConsent,
        operation_fingerprint: str,
    ) -> tuple[DegradedModeConsent, bool]:
        key = (candidate.data_region, candidate.user_id, candidate.idempotency_key)
        with self._lock:
            existing = self._consent_idempotency.get(key)
            if existing is not None:
                fingerprint, consent_id = existing
                if fingerprint != operation_fingerprint:
                    raise IdempotencyConflictError(
                        "consent idempotency key reused with different decision"
                    )
                return self._consents[consent_id], False
            self._consents[candidate.consent_grant_id] = candidate
            self._consent_idempotency[key] = (
                operation_fingerprint,
                candidate.consent_grant_id,
            )
            return candidate, True

    def get_consent(self, consent_id: str) -> DegradedModeConsent:
        with self._lock:
            consent = self._consents.get(consent_id)
            if consent is None:
                raise MaterialAssessmentNotFoundError(consent_id)
            return consent

"""简历解析持久化协议与并发安全内存适配器。"""

from __future__ import annotations

import copy
import threading
from dataclasses import replace
from typing import Protocol

from .models import JsonObject, ParseStatus, ParseTask, ResumeVersion


class ParseTaskNotFoundError(LookupError):
    """解析任务不存在。"""


class ResumeVersionNotFoundError(LookupError):
    """简历版本不存在。"""


class IdempotencyConflictError(RuntimeError):
    """相同幂等键被用于不同不可变输入。"""


class VersionConflictError(RuntimeError):
    """编辑基版本不是当前最新版本。"""


class ParsingRepository(Protocol):
    """解析任务及追加式版本的持久化边界。"""

    def create_task(self, candidate: ParseTask) -> tuple[ParseTask, bool]:
        """原子创建任务或返回相同幂等结果。"""

    def get_task(self, resume_id: str) -> ParseTask:
        """返回简历当前解析任务。"""

    def save_task(self, task: ParseTask) -> None:
        """更新任务执行状态。"""

    def begin_retry(self, *, resume_id: str, idempotency_key: str) -> tuple[ParseTask, bool]:
        """原子领取一次解析重试，重复键不重复执行。"""

    def append_version(
        self,
        *,
        resume_id: str,
        data_region: str,
        profile: JsonObject,
        confirmed_by_user: bool,
        reviewed_paths: frozenset[str],
        expected_base: int | None,
        idempotency_key: str,
        operation_fingerprint: str,
    ) -> tuple[ResumeVersion, bool]:
        """以乐观锁和幂等键追加不可变版本。"""

    def get_version(self, resume_id: str, version: int) -> ResumeVersion:
        """读取指定不可变版本。"""

    def latest_version(self, resume_id: str) -> ResumeVersion:
        """读取当前最新版本。"""


class InMemoryParsingRepository:
    """供合成测试和本地开发使用的原子内存仓储。"""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._tasks_by_resume: dict[str, ParseTask] = {}
        self._task_idempotency: dict[tuple[str, str, str], str] = {}
        self._versions: dict[str, list[ResumeVersion]] = {}
        self._version_idempotency: dict[tuple[str, str], tuple[str, int]] = {}
        self._retry_idempotency: set[tuple[str, str]] = set()

    def create_task(self, candidate: ParseTask) -> tuple[ParseTask, bool]:
        """按区域、用户、幂等键去重，并核对输入指纹。"""
        key = (candidate.data_region, candidate.user_id, candidate.idempotency_key)
        with self._lock:
            existing_resume_id = self._task_idempotency.get(key)
            if existing_resume_id is not None:
                existing = self._tasks_by_resume[existing_resume_id]
                if existing.input_fingerprint != candidate.input_fingerprint:
                    raise IdempotencyConflictError(
                        "parse idempotency key reused with different immutable input"
                    )
                return existing, False
            if candidate.resume_id in self._tasks_by_resume:
                raise IdempotencyConflictError("resume already has a parse task")
            self._tasks_by_resume[candidate.resume_id] = candidate
            self._task_idempotency[key] = candidate.resume_id
            return candidate, True

    def get_task(self, resume_id: str) -> ParseTask:
        """返回任务值对象。"""
        with self._lock:
            task = self._tasks_by_resume.get(resume_id)
            if task is None:
                raise ParseTaskNotFoundError(resume_id)
            return task

    def save_task(self, task: ParseTask) -> None:
        """保存已存在任务的状态。"""
        with self._lock:
            if task.resume_id not in self._tasks_by_resume:
                raise ParseTaskNotFoundError(task.resume_id)
            self._tasks_by_resume[task.resume_id] = task

    def begin_retry(self, *, resume_id: str, idempotency_key: str) -> tuple[ParseTask, bool]:
        """仅 RETRYABLE_FAILURE 可领取；相同键返回当前稳定结果。"""
        key = (resume_id, idempotency_key)
        with self._lock:
            task = self._tasks_by_resume.get(resume_id)
            if task is None:
                raise ParseTaskNotFoundError(resume_id)
            if key in self._retry_idempotency:
                return task, False
            if task.status is not ParseStatus.RETRYABLE_FAILURE:
                raise RuntimeError("parse task is not retryable")
            self._retry_idempotency.add(key)
            claimed = replace(
                task,
                idempotency_key=idempotency_key,
                status=ParseStatus.PENDING,
                message=None,
            )
            self._tasks_by_resume[resume_id] = claimed
            return claimed, True

    def append_version(
        self,
        *,
        resume_id: str,
        data_region: str,
        profile: JsonObject,
        confirmed_by_user: bool,
        reviewed_paths: frozenset[str],
        expected_base: int | None,
        idempotency_key: str,
        operation_fingerprint: str,
    ) -> tuple[ResumeVersion, bool]:
        """追加版本，重复请求返回首次结果且不产生新版本。"""
        idem_key = (resume_id, idempotency_key)
        with self._lock:
            existing = self._version_idempotency.get(idem_key)
            if existing is not None:
                existing_fingerprint, existing_version = existing
                if existing_fingerprint != operation_fingerprint:
                    raise IdempotencyConflictError(
                        "version idempotency key reused with different operation"
                    )
                return self._versions[resume_id][existing_version - 1], False
            versions = self._versions.setdefault(resume_id, [])
            latest = len(versions)
            if expected_base is None:
                if latest != 0:
                    raise VersionConflictError("initial parse version already exists")
            elif expected_base != latest:
                raise VersionConflictError(
                    f"expected base version {expected_base}, current version is {latest}"
                )
            version = ResumeVersion(
                resume_id=resume_id,
                version=latest + 1,
                data_region=data_region,
                profile=copy.deepcopy(profile),
                confirmed_by_user=confirmed_by_user,
                reviewed_low_confidence_paths=reviewed_paths,
            )
            versions.append(version)
            self._version_idempotency[idem_key] = (operation_fingerprint, version.version)
            return version, True

    def get_version(self, resume_id: str, version: int) -> ResumeVersion:
        """返回指定版本的防御性副本。"""
        with self._lock:
            versions = self._versions.get(resume_id, [])
            if version < 1 or version > len(versions):
                raise ResumeVersionNotFoundError(f"{resume_id}:{version}")
            stored = versions[version - 1]
            return replace(stored, profile=copy.deepcopy(stored.profile))

    def latest_version(self, resume_id: str) -> ResumeVersion:
        """返回最新版本的防御性副本。"""
        with self._lock:
            versions = self._versions.get(resume_id, [])
            if not versions:
                raise ResumeVersionNotFoundError(resume_id)
            stored = versions[-1]
            return replace(stored, profile=copy.deepcopy(stored.profile))

    def version_count(self, resume_id: str) -> int:
        """返回测试可观察的追加式版本数量。"""
        with self._lock:
            return len(self._versions.get(resume_id, []))

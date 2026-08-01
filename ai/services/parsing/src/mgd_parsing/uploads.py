"""已安全接受上传的只读边界（TASK-012 → TASK-013）。"""

from __future__ import annotations

import threading
from dataclasses import dataclass
from typing import Protocol


class AcceptedUploadError(RuntimeError):
    """上传不存在、未安全接受或跨区时 fail-closed。"""


@dataclass(frozen=True, slots=True)
class AcceptedResumeText:
    """隔离扫描通过后、供解析器读取的一次输入。"""

    upload_id: str
    user_id: str
    data_region: str
    text: str


class AcceptedUploadReader(Protocol):
    """仅允许读取所属区域 uploads/accepted 对象的协议。"""

    def read_accepted_resume(
        self, *, upload_id: str, user_id: str, data_region: str
    ) -> AcceptedResumeText:
        """读取同用户、同区域且安全扫描通过的简历文本。"""


class InMemoryAcceptedUploadReader:
    """合成测试使用的区域隔离 accepted 上传读取器。"""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._accepted: dict[str, AcceptedResumeText] = {}

    def add(self, upload: AcceptedResumeText) -> None:
        """登记一个已通过 TASK-012 安全扫描的合成上传。"""
        with self._lock:
            self._accepted[upload.upload_id] = upload

    def read_accepted_resume(
        self, *, upload_id: str, user_id: str, data_region: str
    ) -> AcceptedResumeText:
        """跨用户或跨数据区均使用同一拒绝错误，避免资源枚举。"""
        with self._lock:
            upload = self._accepted.get(upload_id)
            if upload is None or upload.user_id != user_id or upload.data_region != data_region:
                raise AcceptedUploadError("accepted upload is unavailable in the requested scope")
            return upload

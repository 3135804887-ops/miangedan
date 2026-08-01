"""不携带 JD/简历正文的 TASK-014 可观测端口。"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol


@dataclass(frozen=True, slots=True)
class JobObservation:
    """固定低基数字段；禁止用户内容、用户 ID、联系方式或供应商密钥。"""

    event: str
    data_region: str
    status: str
    source_kind: str | None = None
    mode: str | None = None
    retryable: bool | None = None
    trace_id: str | None = None


class JobParsingObserver(Protocol):
    """生产组合可映射至 OpenTelemetry 指标、跨度事件和结构化日志。"""

    def record(self, observation: JobObservation) -> None: ...


class NoopJobParsingObserver:
    """默认无副作用实现；业务状态不依赖观测后端可用性。"""

    def record(self, observation: JobObservation) -> None:
        del observation

"""评测结果类型（独立模块，避免 runner/evaluators 循环导入）。"""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class EvalOutcome:
    """单用例评测结果。"""

    case_id: str
    passed: bool
    failures: tuple[str, ...] = field(default_factory=tuple)
    summary: str = ""

"""面个蛋 AI 评测框架（TASK-036）。

追踪：IMPLEMENTATION_PLAN.md TASK-036；ai/evals/datasets（黄金集）与
ai/evals/expected-results（预期结果）；PROMPT-POLICY 第 13 节验证方式。
"""

from .outcome import EvalOutcome
from .runner import (
    EvalCaseResult,
    EvalReport,
    EvalRunner,
    validate_stability_report,
)

__all__ = [
    "EvalCaseResult",
    "EvalOutcome",
    "EvalReport",
    "EvalRunner",
    "validate_stability_report",
]

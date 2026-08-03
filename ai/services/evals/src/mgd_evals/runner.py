"""AI 评测框架运行器（TASK-036）。

能力：黄金集（ai/evals/datasets/*.jsonl）可重复运行、预期结果对齐校验、
内置评测器（交接/安全/通用）、JSON 报告产出与稳定性报告校验（TASK-045 依赖）。
"""

from __future__ import annotations

import json
import sys
from collections.abc import Callable, Mapping, Sequence
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any, cast

from .evaluators import auto_evaluator
from .outcome import EvalOutcome

# 稳定性回归报告门槛（TASK-045：重复评分 95% 维度差 ≤3、及格一致率 ≥98%）。
STABILITY_DIMENSION_DIFF_RATIO_MIN = 0.95
STABILITY_PASS_AGREEMENT_MIN = 0.98

Evaluator = Callable[[dict[str, Any], dict[str, Any]], "EvalOutcome | None"]


@dataclass(frozen=True)
class EvalCaseResult:
    """报告中的单用例条目。"""

    case_id: str
    status: str  # passed | failed | skipped
    failures: tuple[str, ...] = field(default_factory=tuple)
    evaluator: str = ""


@dataclass(frozen=True)
class EvalReport:
    """评测报告（JSON 可序列化；写入 ai/evals/reports/）。"""

    report_kind: str
    dataset: str
    evaluator: str
    generated_at: str
    total: int
    executed: int
    passed: int
    failed: int
    skipped: int
    pass_rate: float
    cases: tuple[EvalCaseResult, ...]
    metrics: dict[str, Any] = field(default_factory=dict)
    thresholds: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return {
            "report_kind": self.report_kind,
            "dataset": self.dataset,
            "evaluator": self.evaluator,
            "generated_at": self.generated_at,
            "totals": {
                "total": self.total,
                "executed": self.executed,
                "passed": self.passed,
                "failed": self.failed,
                "skipped": self.skipped,
                "pass_rate": self.pass_rate,
            },
            "metrics": self.metrics,
            "thresholds": self.thresholds,
            "cases": [
                {
                    "case_id": c.case_id,
                    "status": c.status,
                    "failures": list(c.failures),
                    "evaluator": c.evaluator,
                }
                for c in self.cases
            ],
        }


class EvalError(ValueError):
    """评测框架错误（数据集/预期结果对齐失败等）。"""


class EvalRunner:
    """黄金集评测运行器（确定性：同输入同结论）。"""

    def __init__(
        self,
        evaluator: Evaluator | None = None,
        *,
        now: Callable[[], datetime] | None = None,
    ) -> None:
        self._evaluator = evaluator or auto_evaluator
        self._now = now or (lambda: datetime.now(UTC))

    def run_dataset(
        self,
        dataset_path: Path | str,
        expected_path: Path | str,
        *,
        report_kind: str = "eval",
    ) -> EvalReport:
        """运行单个数据集：逐行评测 + 预期结果对齐校验。"""
        dataset = Path(dataset_path)
        expected_file = Path(expected_path)
        if not dataset.exists() or not expected_file.exists():
            raise EvalError(f"数据集或预期结果缺失：{dataset} / {expected_file}")
        rows = self._load_jsonl(dataset)
        expected_all = json.loads(expected_file.read_text(encoding="utf-8"))
        if not isinstance(expected_all, Mapping):
            raise EvalError(f"预期结果文件必须是对象：{expected_file}")
        cases: list[EvalCaseResult] = []
        for row in rows:
            case_id = str(row["case_id"])
            if case_id not in expected_all:
                cases.append(
                    EvalCaseResult(
                        case_id=case_id,
                        status="failed",
                        failures=(f"缺少预期结果条目（对齐失败）：{case_id}",),
                        evaluator="alignment",
                    )
                )
                continue
            expected = expected_all[case_id]
            if not isinstance(expected, Mapping):
                cases.append(
                    EvalCaseResult(
                        case_id=case_id,
                        status="failed",
                        failures=("预期结果不是对象",),
                        evaluator="alignment",
                    )
                )
                continue
            outcome = self._evaluator(row, cast(dict[str, Any], expected))
            if outcome is None:
                cases.append(EvalCaseResult(case_id=case_id, status="skipped", evaluator="none"))
            else:
                cases.append(
                    EvalCaseResult(
                        case_id=case_id,
                        status="passed" if outcome.passed else "failed",
                        failures=outcome.failures,
                        evaluator=outcome.summary.split(":")[0] or "unknown",
                    )
                )
        executed = [c for c in cases if c.status != "skipped"]
        passed = [c for c in executed if c.status == "passed"]
        pass_rate = len(passed) / len(executed) if executed else 0.0
        return EvalReport(
            report_kind=report_kind,
            dataset=dataset.stem,
            evaluator=self._evaluator.__name__
            if hasattr(self._evaluator, "__name__")
            else "custom",
            generated_at=self._now().isoformat(),
            total=len(cases),
            executed=len(executed),
            passed=len(passed),
            failed=len(executed) - len(passed),
            skipped=len(cases) - len(executed),
            pass_rate=round(pass_rate, 4),
            cases=tuple(cases),
            metrics={"pass_rate": round(pass_rate, 4), "executed": len(executed)},
            thresholds={"pass_rate_min": 1.0},
        )

    @staticmethod
    def _load_jsonl(path: Path) -> list[dict[str, Any]]:
        rows: list[dict[str, Any]] = []
        for line_no, line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
            if not line.strip():
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError as exc:
                raise EvalError(f"{path.name}:{line_no} JSON 解析失败：{exc}") from exc
            if not isinstance(row, Mapping) or "case_id" not in row:
                raise EvalError(f"{path.name}:{line_no} 缺少 case_id")
            rows.append(dict(row))
        return rows

    def write_report(self, report: EvalReport, out_dir: Path | str) -> Path:
        """写入 JSON 报告（ai/evals/reports/）。"""
        target_dir = Path(out_dir)
        target_dir.mkdir(parents=True, exist_ok=True)
        out = target_dir / f"{report.dataset}.eval.json"
        out.write_text(
            json.dumps(report.to_dict(), ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        return out


def validate_stability_report(path: Path | str) -> dict[str, Any]:
    """校验稳定性回归报告（TASK-045 产物；门槛 95%/98%）。"""
    report = json.loads(Path(path).read_text(encoding="utf-8"))
    if report.get("report_kind") != "stability":
        raise EvalError(f"报告类型必须为 stability，实际 {report.get('report_kind')}")
    metrics = report.get("metrics", {})
    if not isinstance(metrics, Mapping):
        raise EvalError("stability 报告缺少 metrics")
    diff_ratio = metrics.get("dimension_diff_le3_ratio")
    agreement = metrics.get("pass_agreement_rate")
    if not isinstance(diff_ratio, (int, float)) or not isinstance(agreement, (int, float)):
        raise EvalError("stability 报告缺少数值指标")
    failures: list[str] = []
    if diff_ratio < STABILITY_DIMENSION_DIFF_RATIO_MIN:
        failures.append(f"维度差 ≤3 比例 {diff_ratio} < {STABILITY_DIMENSION_DIFF_RATIO_MIN}")
    if agreement < STABILITY_PASS_AGREEMENT_MIN:
        failures.append(f"及格一致率 {agreement} < {STABILITY_PASS_AGREEMENT_MIN}")
    if failures:
        raise EvalError("稳定性门槛未达标：" + "；".join(failures))
    return dict(report)


def run_reports(
    datasets: Sequence[str],
    *,
    datasets_dir: Path | str,
    expected_dir: Path | str,
    out_dir: Path | str,
) -> list[Path]:
    """便捷批量运行：返回写入的报告路径列表。"""
    runner = EvalRunner()
    written: list[Path] = []
    for name in datasets:
        report = runner.run_dataset(
            Path(datasets_dir) / f"{name}.jsonl",
            Path(expected_dir) / f"{name}.expected.json",
        )
        written.append(runner.write_report(report, out_dir))
    return written


def main(argv: Sequence[str] | None = None) -> int:
    """CLI：python -m mgd_evals.run --datasets zh-core,en-core --out ai/evals/reports。"""
    args = list(sys.argv[1:] if argv is None else argv)
    datasets_arg = "--datasets"
    out_arg = "--out"
    datasets: list[str] = []
    out_dir = "ai/evals/reports"
    idx = 0
    while idx < len(args):
        if args[idx] == datasets_arg and idx + 1 < len(args):
            datasets = [x.strip() for x in args[idx + 1].split(",") if x.strip()]
            idx += 2
        elif args[idx] == out_arg and idx + 1 < len(args):
            out_dir = args[idx + 1]
            idx += 2
        else:
            raise EvalError(f"未知参数：{args[idx]}")
    if not datasets:
        raise EvalError("必须指定 --datasets（逗号分隔的数据集名）")
    repo_root = Path(__file__).resolve().parents[5]
    written = run_reports(
        datasets,
        datasets_dir=repo_root / "ai" / "evals" / "datasets",
        expected_dir=repo_root / "ai" / "evals" / "expected-results",
        out_dir=Path(out_dir).resolve(),
    )
    for path in written:
        print(f"[评测报告] {path}")
    return 0

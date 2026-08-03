"""TASK-045 稳定性回归报告校验（TASK-036 框架握手：95%/98% 门槛）。"""

from __future__ import annotations

from pathlib import Path

import pytest

from mgd_evals.runner import EvalError, validate_stability_report

REPO_ROOT = Path(__file__).resolve().parents[4]


def test_committed_stability_report_passes_thresholds() -> None:
    report_path = REPO_ROOT / "ai" / "evals" / "reports" / "stability.json"
    if not report_path.exists():
        pytest.skip("稳定性报告未生成（TASK-045 CLI 产物）")
    report = validate_stability_report(report_path)
    metrics = report["metrics"]
    assert metrics["dimension_diff_le3_ratio"] >= 0.95
    assert metrics["pass_agreement_rate"] >= 0.98
    assert report["passed"] is True


def test_stability_report_required_fields(tmp_path: Path) -> None:
    bad = tmp_path / "bad.json"
    bad.write_text('{"report_kind": "eval"}', encoding="utf-8")
    with pytest.raises(EvalError):
        validate_stability_report(bad)

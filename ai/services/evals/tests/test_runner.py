"""TASK-036 AI 评测框架测试（可重复运行、预期对齐、报告产出、稳定性门槛校验）。"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest

from mgd_evals.evaluators import auto_evaluator
from mgd_evals.outcome import EvalOutcome
from mgd_evals.runner import (
    EvalError,
    EvalReport,
    EvalRunner,
    validate_stability_report,
)

REPO_ROOT = Path(__file__).resolve().parents[4]


def test_runner_with_fake_evaluator(tmp_path: Path) -> None:
    dataset = tmp_path / "demo.jsonl"
    expected = tmp_path / "demo.expected.json"
    dataset.write_text(
        "\n".join(
            [
                json.dumps({"case_id": "c1", "input": {"x": 1}}),
                json.dumps({"case_id": "c2", "input": {"x": 2}}),
                json.dumps({"case_id": "c3", "input": {"x": 3}}),
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    expected.write_text(
        json.dumps(
            {
                "c1": {"must_include": ["1"]},
                "c2": {"must_include": ["2"]},
                "c3": {"must_not_include": ["3"]},
            }
        ),
        encoding="utf-8",
    )

    def evaluator(row: dict[str, Any], exp: dict[str, Any]) -> EvalOutcome | None:
        text = json.dumps(row, ensure_ascii=False)
        failures = []
        for phrase in exp.get("must_include", []):
            if phrase not in text:
                failures.append(f"缺 {phrase}")
        for phrase in exp.get("must_not_include", []):
            if phrase in text:
                failures.append(f"含 {phrase}")
        return EvalOutcome(
            case_id=str(row["case_id"]),
            passed=not failures,
            failures=tuple(failures),
        )

    runner = EvalRunner(evaluator=evaluator)
    report = runner.run_dataset(dataset, expected)
    assert report.total == 3
    assert report.passed == 2
    assert report.failed == 1
    assert report.executed == 3
    # 可重复：两次运行结论一致（确定性）。
    report2 = runner.run_dataset(dataset, expected)
    assert [c.status for c in report.cases] == [c.status for c in report2.cases]
    out = runner.write_report(report, tmp_path / "reports")
    assert out.exists()
    loaded = json.loads(out.read_text(encoding="utf-8"))
    assert loaded["report_kind"] == "eval"
    assert loaded["totals"]["passed"] == 2


def test_runner_alignment_failure(tmp_path: Path) -> None:
    dataset = tmp_path / "demo.jsonl"
    expected = tmp_path / "demo.expected.json"
    dataset.write_text(json.dumps({"case_id": "orphan", "input": {}}) + "\n", encoding="utf-8")
    expected.write_text("{}", encoding="utf-8")
    report = EvalRunner().run_dataset(dataset, expected)
    assert report.failed == 1
    assert report.cases[0].evaluator == "alignment"


def test_real_datasets_repeatable() -> None:
    runner = EvalRunner()
    for name in ("zh-core", "en-core"):
        report = runner.run_dataset(
            REPO_ROOT / "ai" / "evals" / "datasets" / f"{name}.jsonl",
            REPO_ROOT / "ai" / "evals" / "expected-results" / f"{name}.expected.json",
        )
        # 已支持的场景（handoff/safety）必须全部通过；不支持的评分场景记为 skipped。
        assert report.failed == 0, f"{name} 失败用例：{report.cases}"
        assert report.executed > 0
        repeat = runner.run_dataset(
            REPO_ROOT / "ai" / "evals" / "datasets" / f"{name}.jsonl",
            REPO_ROOT / "ai" / "evals" / "expected-results" / f"{name}.expected.json",
        )
        assert [c.status for c in report.cases] == [c.status for c in repeat.cases]


def test_auto_evaluator_dispatches() -> None:
    row = json.loads(
        (REPO_ROOT / "ai" / "evals" / "datasets" / "zh-core.jsonl")
        .read_text(encoding="utf-8")
        .splitlines()[0]
    )
    assert auto_evaluator(row, {}) is None  # 评分场景无内置评测器 → skipped
    handoff_line = next(
        line
        for line in (REPO_ROOT / "ai" / "evals" / "datasets" / "zh-core.jsonl")
        .read_text(encoding="utf-8")
        .splitlines()
        if "zh-handoff-01" in line
    )
    handoff_row = json.loads(handoff_line)
    outcome = auto_evaluator(handoff_row, {})
    assert outcome is not None
    assert outcome.passed


def _stability_report(tmp_path: Path, *, diff: float, agreement: float) -> Path:
    report = {
        "report_kind": "stability",
        "dataset": "stability",
        "generated_at": "2026-08-03T00:00:00+00:00",
        "metrics": {
            "dimension_diff_le3_ratio": diff,
            "pass_agreement_rate": agreement,
        },
    }
    path = tmp_path / "stability.json"
    path.write_text(json.dumps(report), encoding="utf-8")
    return path


def test_stability_report_validation(tmp_path: Path) -> None:
    ok = validate_stability_report(_stability_report(tmp_path, diff=0.97, agreement=0.99))
    assert ok["report_kind"] == "stability"
    with pytest.raises(EvalError, match="维度差"):
        validate_stability_report(_stability_report(tmp_path, diff=0.90, agreement=0.99))
    with pytest.raises(EvalError, match="及格一致率"):
        validate_stability_report(_stability_report(tmp_path, diff=0.97, agreement=0.95))
    wrong_kind = tmp_path / "wrong.json"
    wrong_kind.write_text(json.dumps({"report_kind": "eval"}), encoding="utf-8")
    with pytest.raises(EvalError, match="stability"):
        validate_stability_report(wrong_kind)


def test_cli_writes_reports(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    from mgd_evals.runner import main

    monkeypatch.setattr(
        "sys.argv",
        ["mgd-evals", "--datasets", "zh-core,en-core", "--out", str(tmp_path)],
    )
    assert main() == 0
    assert (tmp_path / "zh-core.eval.json").exists()
    assert (tmp_path / "en-core.eval.json").exists()


def test_report_serialization_shape() -> None:
    report = EvalReport(
        report_kind="eval",
        dataset="d",
        evaluator="auto",
        generated_at="t",
        total=1,
        executed=1,
        passed=1,
        failed=0,
        skipped=0,
        pass_rate=1.0,
        cases=(),
    )
    data = report.to_dict()
    assert data["totals"]["pass_rate"] == 1.0

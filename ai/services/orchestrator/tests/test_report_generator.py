"""TASK-050 报告生成器测试（模块/最终结果/文字等价/重试/保护属性零携带）。"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, cast

import pytest

from mgd_orchestrator.report_generator import ReportGenerator

REPO_ROOT = Path(__file__).resolve().parents[4]


def version(
    *,
    seq: int,
    status: str,
    total: int | None,
    weak: list[str] | None = None,
    job_match: dict[str, Any] | None = None,
) -> dict[str, Any]:
    dims = [
        {"dimension": "professional_competence", "score_status": "scored", "score": 78},
        {"dimension": "problem_solving", "score_status": "scored", "score": 74},
        {"dimension": "communication", "score_status": "scored", "score": 72},
        {"dimension": "experience_evidence", "score_status": "scored", "score": 76},
        {"dimension": "behavioral_collaboration", "score_status": "scored", "score": 70},
        {"dimension": "learning_adaptability", "score_status": "scored", "score": 64},
    ]
    if weak:
        for d in dims:
            if d["dimension"] in weak:
                d["score"] = 45
    return {
        "round_sequence": seq,
        "score_id": f"score-{seq}",
        "result_status": status,
        "round_total": total,
        "dimension_results": dims,
        "explanations": {
            "summary": f"第 {seq} 轮评分摘要",
            "strengths": [f"第 {seq} 轮优势一"],
            "improvements": [],
            "input_mode_notes": None,
        },
        "job_match": job_match,
    }


def make_input(**overrides: Any) -> dict[str, Any]:
    base: dict[str, Any] = {
        "project_id": "00000000-0000-4000-8000-000000000001",
        "data_region": "cn",
        "interview_language": "zh-CN",
        "round_weights": [
            {"sequence": 1, "weight": 50},
            {"sequence": 2, "weight": 50},
        ],
        "score_versions": [
            version(seq=1, status="PASS", total=74),
            version(seq=2, status="PASS", total=70),
        ],
        "rounds_evidence": [
            {
                "round_sequence": 1,
                "result_status": "PASS",
                "score_ref": "score-1",
                "questions": [
                    {
                        "question_summary": "请介绍数据同步项目。",
                        "answer_summary": "拆成四个并行分片，延迟从 40 分钟降到 8 分钟。",
                        "followups": ["统计口径是什么？"],
                        "evidence_ids": ["ev-1"],
                        "dimension_scores": {"professional_competence": 78},
                        "done_well": ["量化结果清晰"],
                        "missing": [],
                        "contradictions": [],
                        "suggested_structure": "背景-行动-结果",
                    }
                ],
                "handoff_impact": "交接包已生成（to_round=2）",
            }
        ],
        "input_modes": [
            {
                "round_sequence": 1,
                "input_mode": "voice",
                "structure_clarity_notes": "结构清晰，结论先行",
                "oral_delivery_notes": "语速平稳",
                "evidence_limitations": None,
            }
        ],
        "tools": [
            {"round_sequence": 1, "tools_used": ["whiteboard"], "summary": "使用白板绘制分层图"}
        ],
    }
    base.update(overrides)
    return base


def generate(inp: dict[str, Any], **kwargs: Any) -> dict[str, Any]:
    return ReportGenerator().generate(inp, **kwargs).report


def test_full_report_schema_valid() -> None:
    report = generate(make_input())
    assert report["training_use_disclaimer"] == "模拟训练结果，不代表真实企业录用结论"
    assert report["raw_record_refs"]["deletion_entry"] is True
    assert report["report_kind"] == "full"
    assert report["project_status"] == "COMPLETED"
    assert report["modules"]["overview"]["content"]["final_weighted_score"] == 72
    assert report["modules"]["overview"]["content"]["all_required_passed"] is True
    assert "专业能力" not in json.dumps(report, ensure_ascii=False)  # 雷达用维度键
    radar = report["modules"]["radar"]["content"]
    assert len(radar["dimensions"]) == 6
    assert radar["text_equivalent"]
    rounds = report["modules"]["rounds"]["content"]
    assert rounds[0]["per_question"][0]["evidence_ids"] == ["ev-1"]
    assert rounds[0]["handoff_impact"] == "交接包已生成（to_round=2）"
    assert report["modules"]["rounds"]["trajectory_text"]
    training = report["modules"]["training_plan"]["content"]
    assert training["strengths"]


def test_final_result_rules() -> None:
    # 任一 FAIL → 整体未通过（高平均分不覆盖）。
    failed = generate(
        make_input(
            score_versions=[
                version(seq=1, status="PASS", total=90),
                version(seq=2, status="FAIL", total=45),
            ]
        )
    )
    assert failed["project_status"] == "ROUND_FAILED"
    assert failed["modules"]["overview"]["content"]["all_required_passed"] is False
    # 任一必需轮无有效分 → EVALUATION_INCOMPLETE + partial。
    incomplete = generate(
        make_input(
            score_versions=[
                version(seq=1, status="PASS", total=74),
                version(seq=2, status="EVALUATION_INCOMPLETE", total=None),
            ]
        )
    )
    assert incomplete["project_status"] == "EVALUATION_INCOMPLETE"
    assert incomplete["report_kind"] == "partial"
    assert incomplete["modules"]["overview"]["content"]["final_weighted_score"] == 74


def test_text_mode_notes_and_evidence_limitations() -> None:
    report = generate(
        make_input(
            input_modes=[
                {
                    "round_sequence": 1,
                    "input_mode": "text",
                    "structure_clarity_notes": "文字结构清晰",
                    "oral_delivery_notes": None,
                    "evidence_limitations": (
                        "文字模式：口语表现未评估（not_evaluated），报告标注证据限制"
                    ),
                }
            ]
        )
    )
    comm = report["modules"]["communication_analysis"]["content"]
    assert comm["input_mode"] == "text"
    assert comm["oral_delivery_notes"] is None
    assert "未评估" in comm["evidence_limitations"]


def test_no_jd_job_match_null_with_notes() -> None:
    report = generate(make_input(score_versions=[version(seq=1, status="PASS", total=74)]))
    module = report["modules"]["job_match"]
    assert module["content"] is None
    assert "无 JD" in module["notes"]


def test_job_match_content() -> None:
    jm = {
        "must_have": {"match_ratio": 1.0, "proven": ["r1"], "unproven": []},
        "nice_to_have": {"match_ratio": 0.5, "proven": ["n1"], "unproven": ["n2"]},
    }
    report = generate(
        make_input(score_versions=[version(seq=1, status="PASS", total=74, job_match=jm)])
    )
    content = report["modules"]["job_match"]["content"]
    assert content["must_have_ratio"] == 1.0
    assert "r1" in content["proven_by_interview"]
    assert "n2" in content["unproven"]


def test_module_failure_and_partial_retry(monkeypatch: pytest.MonkeyPatch) -> None:
    generator = ReportGenerator()
    original = generator._build_radar

    def broken(_inp: Any) -> dict[str, Any]:
        raise RuntimeError("雷达模块故障")

    monkeypatch.setattr(generator, "_build_radar", broken)
    result = generator.generate(make_input())
    statuses = {m.name: m.status for m in result.module_statuses}
    assert statuses["radar"] == "failed"
    assert statuses["overview"] == "ok"
    assert result.report["report_kind"] == "partial"
    assert "radar" in result.failed_modules
    # 只重试失败模块：radar 恢复，其余模块保留。
    monkeypatch.setattr(generator, "_build_radar", original)
    recovered = generator.regenerate_module(result.report, make_input(), "radar")
    assert recovered.module_statuses[-1].status == "ok"
    assert recovered.report["modules"]["radar"]["status"] == "ok"
    assert recovered.report["modules"]["overview"] == result.report["modules"]["overview"]


def test_module_failure_required_modules_present() -> None:
    result = ReportGenerator().generate(
        make_input(), modules=["overview", "radar", "rounds", "training_plan"]
    )
    assert result.report["modules"]["overview"]["status"] == "ok"
    with pytest.raises(ValueError, match="必备模块缺失"):
        ReportGenerator().generate(make_input(), modules=["overview"])


def test_protected_attribute_redacted_from_report() -> None:
    inp = make_input(
        rounds_evidence=[
            {
                "round_sequence": 1,
                "result_status": "PASS",
                "score_ref": "score-1",
                "questions": [
                    {
                        "question_summary": "请介绍项目。",
                        "answer_summary": (
                            "我今年 24 岁，长相比较显小，项目延迟从 40 分钟降到 8 分钟。"
                        ),
                        "followups": [],
                        "evidence_ids": ["ev-1"],
                        "dimension_scores": {},
                        "done_well": [],
                        "missing": [],
                        "contradictions": [],
                        "suggested_structure": None,
                    }
                ],
            }
        ]
    )
    report = generate(inp)
    summary = report["modules"]["rounds"]["content"][0]["per_question"][0]["answer_summary"]
    assert "24 岁" not in summary
    assert "长相" not in summary
    assert "〔已脱敏〕" in summary
    blob = json.dumps(report, ensure_ascii=False)
    assert "长相比较显小" not in blob


# ---- ai/evals 报告用例回归（REPORT-SPEC 第 7 节） ----
def _report_cases() -> list[tuple[str, dict[str, Any]]]:
    cases: list[tuple[str, dict[str, Any]]] = []
    for ds_name in ("zh-core.jsonl", "en-core.jsonl"):
        path = REPO_ROOT / "ai" / "evals" / "datasets" / ds_name
        if not path.exists():
            continue
        for line in path.read_text(encoding="utf-8").splitlines():
            if not line.strip():
                continue
            row = json.loads(line)
            if row.get("scenario_type") == "report_generation":
                cases.append((row["case_id"], row))
    return cases


@pytest.mark.parametrize(
    "case_id,row",
    [pytest.param(cid, row, id=cid) for cid, row in _report_cases()],
)
def test_eval_report_cases(case_id: str, row: dict[str, Any]) -> None:
    report = generate(cast(dict[str, Any], row["input"]["report_input"]))
    text = json.dumps(report, ensure_ascii=False)
    expected = json.loads(
        (
            REPO_ROOT
            / "ai"
            / "evals"
            / "expected-results"
            / f"{(row['language'] == 'zh-CN' and 'zh-core') or 'en-core'}.expected.json"
        ).read_text(encoding="utf-8")
    )[case_id]
    for phrase in expected.get("must_include", []):
        assert phrase in text, f"{case_id} 缺少 {phrase}"
    for phrase in expected.get("must_not_include", []):
        assert phrase not in text, f"{case_id} 出现 {phrase}"

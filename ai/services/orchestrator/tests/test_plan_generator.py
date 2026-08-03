"""TASK-033 计划生成链路测试（来源融合/轮次边界/安全过滤/部分重试/Schema 校验）。"""

from __future__ import annotations

from pathlib import Path

import pytest

from mgd_orchestrator.plan_generator import PlanGenerator, ProcessSource
from mgd_orchestrator.prompt_registry import PromptRegistry

REPO_ROOT = Path(__file__).resolve().parents[4]


def test_default_plan_and_bounds() -> None:
    gen = PlanGenerator()
    result = gen.generate(
        resume_profile={"name": "候选人"},
        job_profile={"title": "后端工程师"},
        degraded_mode="full",
        process_sources=(
            ProcessSource(source_id="s1", source_type="official_careers_page", credibility="high"),
        ),
    )
    draft = result.draft
    assert 1 <= len(draft.rounds) <= 5
    assert all(10 <= r.duration_minutes <= 60 for r in draft.rounds)
    assert all(r.difficulty in {"basic", "standard", "challenge"} for r in draft.rounds)
    assert sum(draft.dimension_weights.values()) == 100
    assert sum(w["weight"] for w in draft.round_weights) == 100
    assert draft.flow_uses_generic_template is False
    assert result.retries_used == 0


def test_generic_template_fallback_without_sources() -> None:
    gen = PlanGenerator()
    result = gen.generate(
        resume_profile={"name": "候选人"},
        process_sources=(
            ProcessSource(
                source_id="s1",
                source_type="candidate_experience",
                credibility="low",
                is_unofficial_experience=True,
            ),
        ),
    )
    assert result.draft.flow_uses_generic_template is True
    assert result.draft.process_source_refs == ()


def test_reliable_sources_merged() -> None:
    gen = PlanGenerator()
    result = gen.generate(
        resume_profile={"name": "候选人"},
        process_sources=(
            ProcessSource(source_id="s1", source_type="official_careers_page", credibility="high"),
            ProcessSource(
                source_id="s2",
                source_type="candidate_experience",
                credibility="low",
                is_unofficial_experience=True,
            ),
        ),
    )
    assert result.draft.flow_uses_generic_template is False
    assert result.draft.process_source_refs[0]["source_id"] == "s1"
    assert len(result.draft.process_source_refs) == 1  # 不可信经验来源不进入引用


def test_safety_filter_pii_redact() -> None:
    gen = PlanGenerator()
    # 模拟模型把邮箱复述进 focus（测试 _regenerate_sanitized 路径）。
    from mgd_orchestrator.plan_generator import DraftPlan, RoundDraft

    dirty = DraftPlan(
        degraded_mode="full",
        dimension_weights={
            k: v
            for k, v in zip(
                (
                    "professional_competence",
                    "problem_solving",
                    "communication",
                    "experience_evidence",
                    "behavioral_collaboration",
                    "learning_adaptability",
                ),
                (25, 20, 15, 15, 15, 10),
                strict=True,
            )
        },
        rounds=(
            RoundDraft(
                sequence=1,
                round_type="role_professional",
                role="面试官",
                focus="联系 test@example.com 预约",
                duration_minutes=30,
                difficulty="standard",
                critical_dimensions=("professional_competence",),
                tools=(),
                coverage_points=(),
            ),
        ),
        round_weights=({"sequence": 1, "weight": 100},),
        process_source_refs=(),
        flow_uses_generic_template=False,
    )
    checked = gen.safety_filter(dirty)
    assert "pii_echo" in checked.safety_issues


def test_schema_validation_of_draft() -> None:
    registry = PromptRegistry(
        prompts_dir=REPO_ROOT / "ai" / "prompts",
        schemas_dir=REPO_ROOT / "ai" / "schemas",
    )
    gen = PlanGenerator(registry=registry)
    result = gen.generate(resume_profile={}, job_profile={})
    payload = gen.to_schema_draft(result)
    full = {
        "schema_version": "1.0.0",
        "project_id": "00000000-0000-4000-8000-000000000001",
        "plan_version": 1,
        "data_region": "cn",
        "interview_language": "zh-CN",
        "rubric_version": "rubrics/v1/default",
        "created_at": "2026-08-02T00:00:00Z",
        "frozen": False,
        **{k: v for k, v in payload.items() if k != "safety_issues"},
    }
    registry.validate_output(registry.get("prompt-plan-generation"), full)


def test_regenerate_single_round() -> None:
    gen = PlanGenerator()
    result = gen.generate(
        resume_profile={},
        process_sources=(
            ProcessSource(source_id="s1", source_type="official_careers_page", credibility="high"),
        ),
    )
    retried = gen.regenerate_round(result.draft, 1)
    assert retried.sequence == 1
    assert "重试" in retried.focus
    with pytest.raises(KeyError):
        gen.regenerate_round(result.draft, 99)


def test_invalid_inputs_rejected() -> None:
    gen = PlanGenerator()
    with pytest.raises(ValueError):
        gen.generate(interview_language="fr-FR")
    with pytest.raises(ValueError):
        gen.generate(degraded_mode="bogus")


def test_plan_generation_within_budget() -> None:
    """TASK-090 补测（TC-NFR-016-N01）：计划生成 ≤120s 预算冒烟。"""
    import time

    gen = PlanGenerator()
    start = time.monotonic()
    result = gen.generate(
        resume_profile={"name": "候选人"},
        job_profile={"title": "后端工程师"},
        degraded_mode="full",
        process_sources=(
            ProcessSource(source_id="s1", source_type="official_careers_page", credibility="high"),
        ),
    )
    elapsed = time.monotonic() - start
    assert result.draft.rounds
    assert elapsed <= 120.0

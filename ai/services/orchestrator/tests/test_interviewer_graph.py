"""TASK-032 面试官决策图测试（覆盖推进/追问/打断/工具/检查点恢复）。"""

from __future__ import annotations

from mgd_orchestrator.interviewer_graph import (
    InterviewerConfig,
    build_interviewer_graph,
    restore,
    snapshot,
)


def test_full_turn_coverage_advancement() -> None:
    cfg = InterviewerConfig(coverage_priority=("c1", "c2"), max_followups_per_question=1)
    graph = build_interviewer_graph(cfg)
    final = graph.invoke(
        {
            "turn_index": 1,
            "coverage_ids": ["c1", "c2"],
            "remaining_coverage": ["c1", "c2"],
            "last_answer_quality": "good",
        }
    )
    assert final["phase"] == "complete"
    assert final["remaining_coverage"] == []
    assert final["last_covered"] == "c2"
    assert final["ask_count"] == 2


def test_followup_within_budget_and_bounds() -> None:
    cfg = InterviewerConfig(coverage_priority=("c1",), max_followups_per_question=2)
    graph = build_interviewer_graph(cfg)
    state = {
        "turn_index": 1,
        "coverage_ids": ["c1"],
        "remaining_coverage": ["c1"],
        "last_answer_quality": "weak",
        "followup_count": 0,
    }
    # 弱回答触发追问，且追问不越出已确认覆盖点（question_id 不变）。
    first = graph.invoke(state)
    assert first["actions"].count("followup_asked") == 2
    assert first["last_covered"] == "c1"
    # 追问预算用尽后推进（即使回答仍弱）。
    exhausted = graph.invoke({**state, "followup_count": 2, "last_answer_quality": "weak"})
    assert exhausted["remaining_coverage"] == []
    assert exhausted["actions"].count("followup_asked") == 0
    assert "coverage_advanced" in exhausted["actions"]


def test_interrupt_handling() -> None:
    cfg = InterviewerConfig(coverage_priority=("c1",))
    graph = build_interviewer_graph(cfg)
    final = graph.invoke(
        {
            "turn_index": 1,
            "remaining_coverage": ["c1"],
            "last_answer_quality": "good",
            "interrupt": "button",
            "interrupt_at_ms": 1234,
        }
    )
    assert "avatar_stopped" in final["actions"]
    assert final["interrupted_at_ms"] == 1234
    assert final["interrupt"] == "none"


def test_tool_whitelist_enforced() -> None:
    cfg = InterviewerConfig(coverage_priority=("c1",), tool_whitelist=frozenset({"code_editor"}))
    graph = build_interviewer_graph(cfg)
    allowed = graph.invoke(
        {
            "turn_index": 1,
            "remaining_coverage": ["c1"],
            "last_answer_quality": "good",
            "tool_requested": "code_editor",
        }
    )
    assert allowed["tool_used"] is True
    assert "tool_invoked" in allowed["actions"]
    denied = graph.invoke(
        {
            "turn_index": 1,
            "remaining_coverage": ["c1"],
            "last_answer_quality": "good",
            "tool_requested": "whiteboard",
        }
    )
    assert denied["tool_used"] is False
    assert "tool_rejected" in denied["actions"]


def test_checkpoint_restore() -> None:
    cfg = InterviewerConfig(coverage_priority=("c1", "c2"))
    graph = build_interviewer_graph(cfg)
    mid = snapshot(
        {
            "turn_index": 1,
            "remaining_coverage": ["c2"],
            "question_id": "c2",
            "last_action": "asked",
        },
        checkpoint_id="ck-1",
    )
    restored = restore(mid)
    final = graph.invoke(restored)
    assert final["phase"] == "complete"
    assert final["last_covered"] == "c2"


def test_non_convergence_guard() -> None:
    from mgd_orchestrator.interviewer_graph import StateGraph

    graph = StateGraph()
    graph.add_node("a", lambda st: st)
    graph.add_node("b", lambda st: st)
    graph.add_edge("a", "b")
    graph.add_edge("b", "a")
    compiled = graph.compile()
    try:
        compiled.invoke({})
    except RuntimeError as exc:
        assert "最大步数" in str(exc)
    else:
        raise AssertionError("应触发最大步数保护")


def test_response_latency_budget() -> None:
    """TASK-090 补测（TC-NFR-008-N01）：单回合回应 ≤5s 预算冒烟（P99 上界）。"""
    import time

    cfg = InterviewerConfig(coverage_priority=("c1", "c2"), max_followups_per_question=1)
    graph = build_interviewer_graph(cfg)
    start = time.monotonic()
    final = graph.invoke(
        {
            "turn_index": 1,
            "coverage_ids": ["c1", "c2"],
            "remaining_coverage": ["c1", "c2"],
            "last_answer_quality": "good",
        }
    )
    elapsed = time.monotonic() - start
    assert final["phase"] == "complete"
    assert elapsed <= 5.0

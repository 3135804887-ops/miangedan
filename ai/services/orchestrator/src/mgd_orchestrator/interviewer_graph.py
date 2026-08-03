"""面试官决策图（TASK-032，FR-012）。

追踪：IMPLEMENTATION_PLAN.md TASK-032；docs/ai/AI-ORCHESTRATION.md；
docs/ai/PROMPT-POLICY.md；docs/ai/HANDOFF-SPEC.md。

实现一个与 LangGraph StateGraph 语义对齐的确定性迷你图引擎：
- 节点（add_node）+ 条件边（add_conditional_edges）+ 编译（compile）+ 调用（invoke）；
- 状态可序列化/恢复（检查点），重放安全（NFR-006）；
- 面试官决策：覆盖点推进、动态追问、打断策略、工具使用；
- 图节点只产出"建议"，不直接写业务状态（确定性业务状态由 Temporal 工作流控制）。

生产接入 LangGraph 时按相同节点/边声明迁移（本实现零外部依赖，保证 CI 可测）。
"""

from __future__ import annotations

from collections.abc import Callable, Mapping
from dataclasses import dataclass
from typing import Any, TypeAlias

NodeFn: TypeAlias = Callable[[Mapping[str, Any]], Mapping[str, Any]]
RouterFn: TypeAlias = Callable[[Mapping[str, Any]], str]


class StateGraph:
    """确定性状态图（LangGraph StateGraph 语义的迷你实现）。"""

    def __init__(self) -> None:
        self._nodes: dict[str, NodeFn] = {}
        self._edges: dict[str, str] = {}
        self._conditional: dict[str, tuple[RouterFn, dict[str, str]]] = {}
        self._entry = ""

    def add_node(self, name: str, fn: NodeFn) -> None:
        if name in self._nodes:
            raise ValueError(f"节点重复: {name}")
        self._nodes[name] = fn
        if not self._entry:
            self._entry = name

    def add_edge(self, source: str, target: str) -> None:
        self._edges[source] = target

    def add_conditional_edges(
        self, source: str, router: RouterFn, path_map: Mapping[str, str]
    ) -> None:
        self._conditional[source] = (router, dict(path_map))

    def compile(self) -> CompiledGraph:
        if not self._entry or self._entry not in self._nodes:
            raise ValueError("图缺少入口节点")
        return CompiledGraph(self)


class CompiledGraph:
    """已编译图：invoke 从入口运行到终止节点，返回最终状态。"""

    def __init__(self, graph: StateGraph) -> None:
        self._graph = graph

    def invoke(self, state: Mapping[str, Any], max_steps: int = 32) -> dict[str, Any]:
        current = self._graph._entry
        st = dict(state)
        for _ in range(max_steps):
            if current not in self._graph._nodes:
                raise ValueError(f"未知节点: {current}")
            fn = self._graph._nodes[current]
            result = fn(st)
            st.update(result)
            if current in self._graph._conditional:
                router, path_map = self._graph._conditional[current]
                nxt = router(st)
                if nxt not in path_map:
                    raise ValueError(f"路由结果 {nxt!r} 不在路径映射中")
                current = path_map[nxt]
            elif current in self._graph._edges:
                current = self._graph._edges[current]
            else:
                return st
        raise RuntimeError(f"图超过最大步数 {max_steps}（可能未收敛）")


@dataclass(frozen=True)
class InterviewerSnapshot:
    """可恢复的图状态快照（检查点；重放安全）。"""

    state: Mapping[str, Any]
    checkpoint_id: str = ""


@dataclass
class InterviewerConfig:
    """面试官行为边界（plan 冻结快照；图内只读）。"""

    max_followups_per_question: int = 2
    coverage_priority: tuple[str, ...] = ()
    tool_whitelist: frozenset[str] = frozenset()
    language: str = "zh-CN"


def build_interviewer_graph(cfg: InterviewerConfig) -> CompiledGraph:
    """构建面试官决策图（TASK-032）。

    节点：start → select_question → ask_question → decide → [followup | use_tool |
    complete_turn]；interrupt 事件在任意节点间生效（状态标记，不改变图拓扑）。
    """
    graph = StateGraph()

    def start(state: Mapping[str, Any]) -> dict[str, Any]:
        base = dict(state)
        base.setdefault("turn_index", 1)
        base.setdefault("coverage_ids", list(cfg.coverage_priority))
        base.setdefault("remaining_coverage", list(cfg.coverage_priority))
        base.setdefault("followup_count", 0)
        base.setdefault("interrupt", "none")
        base.setdefault("tool_used", False)
        base.setdefault("actions", [])
        return base

    def _record(state: Mapping[str, Any], action: str) -> dict[str, Any]:
        actions = list(state.get("actions", []))
        actions.append(action)
        return {"last_action": action, "actions": actions}

    def select_question(state: Mapping[str, Any]) -> dict[str, Any]:
        remaining = state.get("remaining_coverage", [])
        if not remaining:
            return {"question_id": None, "phase": "complete"}
        next_id = remaining[0]
        return {
            "question_id": next_id,
            "phase": "asking",
            "ask_count": state.get("ask_count", 0) + 1,
        }

    def ask_question(state: Mapping[str, Any]) -> dict[str, Any]:
        return _record(state, "asked")

    def decide(state: Mapping[str, Any]) -> str:
        """覆盖点推进 / 动态追问 / 工具使用 / 回合完成路由。"""
        if state.get("interrupt") in {"voice", "button"}:
            return "interrupt"
        if state.get("tool_requested") and not state.get("tool_used"):
            return "tool"
        followups = int(state.get("followup_count", 0))
        quality = state.get("last_answer_quality", "good")
        remaining = state.get("remaining_coverage", [])
        if quality in {"weak", "partial"} and followups < cfg.max_followups_per_question:
            return "followup"
        if remaining:
            return "advance"
        return "complete"

    def followup(state: Mapping[str, Any]) -> dict[str, Any]:
        out = _record(state, "followup_asked")
        out.update(
            {
                "followup_count": int(state.get("followup_count", 0)) + 1,
                # 追问不越出已确认覆盖点集合（覆盖范围由计划冻结）。
                "question_id": state.get("question_id"),
            }
        )
        return out

    def use_tool(state: Mapping[str, Any]) -> dict[str, Any]:
        tool = state.get("tool_requested")
        if tool not in cfg.tool_whitelist:
            out = _record(state, "tool_rejected")
            out.update({"tool_used": False, "tool_requested": None})
            return out
        out = _record(state, "tool_invoked")
        out.update({"tool_used": True, "tool_requested": None})
        return out

    def handle_interrupt(state: Mapping[str, Any]) -> dict[str, Any]:
        out = _record(state, "avatar_stopped")
        out.update({"interrupt": "none", "interrupted_at_ms": state.get("interrupt_at_ms", 0)})
        return out

    def advance(state: Mapping[str, Any]) -> dict[str, Any]:
        remaining = list(state.get("remaining_coverage", []))
        done = remaining.pop(0) if remaining else None
        out = _record(state, "coverage_advanced")
        out.update(
            {
                "remaining_coverage": remaining,
                "last_covered": done,
                "followup_count": 0,
            }
        )
        return out

    def complete_turn(state: Mapping[str, Any]) -> dict[str, Any]:
        return {"phase": "complete", "turn_index": int(state.get("turn_index", 1))}

    graph.add_node("start", start)
    graph.add_node("select_question", select_question)
    graph.add_node("ask_question", ask_question)
    graph.add_node("followup", followup)
    graph.add_node("use_tool", use_tool)
    graph.add_node("interrupt", handle_interrupt)
    graph.add_node("advance", advance)
    graph.add_node("complete_turn", complete_turn)

    graph.add_edge("start", "select_question")
    graph.add_conditional_edges(
        "select_question",
        lambda st: "complete" if st.get("question_id") is None else "ask",
        {"ask": "ask_question", "complete": "complete_turn"},
    )
    graph.add_conditional_edges(
        "ask_question",
        decide,
        {
            "followup": "followup",
            "tool": "use_tool",
            "interrupt": "interrupt",
            "advance": "advance",
            "complete": "complete_turn",
        },
    )
    graph.add_edge("followup", "ask_question")
    graph.add_edge("use_tool", "ask_question")
    graph.add_edge("interrupt", "ask_question")
    graph.add_edge("advance", "select_question")
    return graph.compile()


def snapshot(state: Mapping[str, Any], checkpoint_id: str = "") -> InterviewerSnapshot:
    """生成检查点快照（可恢复；重放安全）。"""
    return InterviewerSnapshot(state=dict(state), checkpoint_id=checkpoint_id)


def restore(snapshot: InterviewerSnapshot) -> dict[str, Any]:
    """从检查点恢复状态。"""
    return dict(snapshot.state)


__all__ = [
    "CompiledGraph",
    "InterviewerConfig",
    "InterviewerSnapshot",
    "StateGraph",
    "build_interviewer_graph",
    "restore",
    "snapshot",
]

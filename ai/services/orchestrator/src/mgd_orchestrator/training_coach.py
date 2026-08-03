"""AI 教练练习（TASK-052，FR-024；US-04 场景 3）。

追踪：IMPLEMENTATION_PLAN.md TASK-052；docs/ai/PROMPT-POLICY.md；
ai/prompts/training-coach.md；docs/design/SCREEN-SPEC.md SCR-12。

职责：
- 针对薄弱维度生成练习：原题/变体/提示/框架/示例（CoachOutput 结构对齐
  training-coach.md 第 4 节）；
- 逐步反馈：亮点 → 缺口 → 下一步（聚焦行为与证据，先优势后改进）；
- 练习永不改变正式分数与解锁状态：is_formal_evidence 恒为 false，
  PracticeRecord 与正式证据链/ScoreVersion 完全隔离；
- 不泄露后续轮次正式考点与完整标准答案：变体只围绕已考覆盖点；
- 安全：cheating/insult/discrimination/employment → 阻断重生成 ≤3 次，
  危险/骚扰 → 升级人工；用户练习回答中的注入按数据处理并说明隔离；
- 降级：生成失败 → 静态框架/通用练习建议（标注非个性化）。

当前为确定性合成实现（供应商选型前不绑定厂商 SDK）。
"""

from __future__ import annotations

import re
import uuid
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any

from .safety_pipeline import ContentSafetyPipeline

_KINDS = ("original_question", "variant", "framework", "example")
_DIMENSIONS = (
    "professional_competence",
    "problem_solving",
    "communication",
    "experience_evidence",
    "behavioral_collaboration",
    "learning_adaptability",
)

# 维度级通用框架（降级/无覆盖点回退时使用；非个性化标注由调用方附加）。
_FRAMEWORKS: dict[str, str] = {
    "professional_competence": (
        "先陈述岗位相关知识/方法，再给出工具与正确性验证，最后补充边界与取舍。"
    ),
    "problem_solving": "先拆解问题（背景-约束-目标），再给出方案与权衡，最后说明验证与复盘。",
    "communication": "结论先行，按结构分点，量化关键信息，结尾总结。",
    "experience_evidence": "按 STAR 结构：情境-任务-行动-结果，结果尽量量化。",
    "behavioral_collaboration": "先说明场景与冲突，再讲你的行动与影响，最后复盘学到什么。",
    "learning_adaptability": "先说明变化情境，再讲学习路径与行动，最后给出迁移应用。",
}


@dataclass(frozen=True)
class CoachItem:
    """练习项（training-coach.md CoachOutput）。"""

    output_kind: str
    content: str
    linked_dimension: str
    linked_coverage_id: str | None = None
    is_formal_evidence: bool = False
    next_action_hint: str = "continue_practice"


@dataclass(frozen=True)
class CoachFeedback:
    """逐步反馈（亮点 → 缺口 → 下一步）。"""

    output_kind: str
    content: str
    linked_dimension: str
    linked_coverage_id: str | None = None
    is_formal_evidence: bool = False
    next_action_hint: str = "continue_practice"
    injection_detected: bool = False


@dataclass(frozen=True)
class PracticeRecord:
    """练习记录（独立于正式证据链；永不产生 ScoreVersion）。"""

    practice_id: str
    project_id: str
    practice_type: str
    item: CoachItem
    feedbacks: tuple[CoachFeedback, ...] = field(default_factory=tuple)
    affects_formal_scores: bool = False
    created_at: str = ""


class CoachError(ValueError):
    """教练练习生成失败（可重试/降级）。"""


class CoachEscalationError(CoachError):
    """危险/骚扰内容：升级人工审查。"""


class TrainingCoach:
    """确定性训练教练（练习隔离红线可测试）。"""

    def __init__(self, safety: ContentSafetyPipeline | None = None) -> None:
        self._safety = safety if safety is not None else ContentSafetyPipeline()

    # ---- 练习项生成 ----
    def create_item(
        self,
        *,
        dimension: str,
        practice_type: str,
        coverage_id: str | None = None,
        failed_question: str | None = None,
        plan_snapshot: Mapping[str, Any] | None = None,
        jd_excerpt: str = "",
        resume_excerpt: str = "",
    ) -> CoachItem:
        """生成练习项；变体必须关联已考覆盖点（不预演后续轮次）。"""
        if dimension not in _DIMENSIONS:
            raise CoachError(f"未知维度 {dimension}")
        if practice_type not in _KINDS:
            raise CoachError(f"未知练习类型 {practice_type}")
        coverage = self._resolve_coverage(
            dimension=dimension,
            coverage_id=coverage_id,
            plan_snapshot=plan_snapshot,
        )
        coverage_label = coverage if coverage is not None else f"{dimension}（非个性化）"
        if practice_type == "original_question":
            content = self._original_question(failed_question, dimension)
        elif practice_type == "variant":
            content = self._variant(coverage_label, dimension, jd_excerpt, resume_excerpt)
        elif practice_type == "framework":
            content = self._framework(dimension)
        else:
            content = self._example(dimension)
        content = self._safety_check(content)
        return CoachItem(
            output_kind="practice_item",
            content=content,
            linked_dimension=dimension,
            linked_coverage_id=coverage,
            is_formal_evidence=False,
            next_action_hint="continue_practice",
        )

    def _original_question(self, failed_question: str | None, dimension: str) -> str:
        if failed_question:
            return (
                f"原题练习（仅用于练习，不改变正式分数）：{failed_question}"
                "。请按你现在的理解重新回答。"
            )
        return (
            f"原题练习（维度 {dimension}）：请复述该轮失败题的题干并重新作答"
            "（仅练习，不记正式分数）。"
        )

    def _variant(self, coverage: str, dimension: str, jd: str, resume: str) -> str:
        context = "结合岗位要求与你的经历" if (jd or resume) else ""
        return (
            f"变体练习（已考覆盖点 {coverage}，{dimension}）："
            f"换一个新情境考察同一能力{('，' + context) if context else ''}。"
            "本题为练习，不计入正式评分。"
        )

    def _framework(self, dimension: str) -> str:
        return f"回答框架（{dimension}）：{_FRAMEWORKS.get(dimension, '')}（训练用途）"

    def _example(self, dimension: str) -> str:
        return (
            f"示例结构（{dimension}）：仅演示表达结构与思路，"
            "不代表真实企业录用结论；请用自己的经历改写。"
        )

    # ---- 逐步反馈 ----
    def feedback(
        self,
        *,
        item: CoachItem,
        user_answer: str,
    ) -> CoachFeedback:
        """对练习回答给出逐步反馈：亮点 → 缺口 → 下一步（先优势后改进）。"""
        if not user_answer.strip():
            raise CoachError("练习回答不能为空")
        injection = self._safety.detect_injection(user_answer)
        sanitized = self._safety.neutralize(user_answer, injection) if injection else user_answer
        weaknesses = self._analyze(sanitized)
        strengths = self._highlights(sanitized)
        next_step = self._next_step(item, weaknesses)
        content = self._safety_check(
            "亮点："
            + ("；".join(strengths) if strengths else "回答方向正确")
            + "\n缺口："
            + ("；".join(weaknesses) if weaknesses else "无明显缺口，可继续练习")
            + f"\n下一步：{next_step}"
        )
        if injection:
            content += "\n（检测到指令性内容，已按数据处理；练习与正式评分完全隔离。）"
        hint = (
            "suggest_formal_retry"
            if item.linked_dimension in self._weak_hint(item)
            else "continue_practice"
        )
        return CoachFeedback(
            output_kind="step_feedback",
            content=content,
            linked_dimension=item.linked_dimension,
            linked_coverage_id=item.linked_coverage_id,
            is_formal_evidence=False,
            next_action_hint=hint,
            injection_detected=bool(injection),
        )

    @staticmethod
    def _weak_hint(item: CoachItem) -> set[str]:
        # 由调用方标记薄弱维度；此处保守仅对失败原题给出正式重试建议。
        if item.output_kind == "practice_item" and "原题练习" in item.content:
            return {item.linked_dimension}
        return set()

    @staticmethod
    def _highlights(text: str) -> list[str]:
        highlights: list[str] = []
        if re.search(r"\d+(?:%|％|分钟|天|周|次|倍|个|万|亿)?", text):
            highlights.append("包含量化结果，便于评估影响")
        if any(k in text for k in ("分层", "拆解", "权衡", "取舍", "复盘", "验证")):
            highlights.append("体现了结构化思考")
        if not highlights:
            highlights.append("回答了核心问题")
        return highlights

    @staticmethod
    def _analyze(text: str) -> list[str]:
        gaps: list[str] = []
        if not re.search(r"\d+(?:%|％|分钟|天|周|次|倍|个|万|亿)?", text):
            gaps.append("缺少量化结果或具体数据")
        if not any(
            k in text for k in ("背景", "情境", "任务", "行动", "结果", "第一步", "然后", "最后")
        ):
            gaps.append("结构不够清晰，建议按框架组织")
        if len(text) < 60:
            gaps.append("信息量偏少，可补充关键细节")
        return gaps

    @staticmethod
    def _next_step(item: CoachItem, gaps: Sequence[str]) -> str:
        if not gaps:
            return f"可在 {item.linked_dimension} 维度继续更高难度练习"
        return f"先补齐：{'、'.join(gaps[:2])}；完成后可发起正式重试（唯一改变正式结果的路径）"

    # ---- 覆盖点解析 ----
    @staticmethod
    def _resolve_coverage(
        *,
        dimension: str,
        coverage_id: str | None,
        plan_snapshot: Mapping[str, Any] | None,
    ) -> str | None:
        if coverage_id is None:
            return None
        if plan_snapshot is None:
            return coverage_id
        rounds = plan_snapshot.get("rounds", [])
        if not isinstance(rounds, Sequence):
            return coverage_id
        for round_item in rounds:
            points = round_item.get("question_coverage_plan", {}).get("coverage_points", [])
            for point in points:
                if isinstance(point, Mapping) and point.get("coverage_id") == coverage_id:
                    return coverage_id
        return None  # 未关联已考覆盖点 → 降级为维度级（非个性化）

    # ---- 安全 ----
    def _safety_check(self, content: str) -> str:
        verdict = self._safety.classify(content)
        if verdict.allowed and not verdict.injection_detected:
            return content
        if verdict.escalated:
            raise CoachEscalationError(
                f"内容命中 {[h.category for h in verdict.hits]}，升级人工审查"
            )
        # block_and_regenerate ≤3 次：确定性退化为通用框架。
        for _ in range(3):
            content = self._safety.redact(content)
            verdict = self._safety.classify(content)
            if verdict.allowed and not verdict.injection_detected:
                return content
            content = f"（练习内容已按安全策略调整）{_FRAMEWORKS.get('communication', '')}"
        raise CoachError("练习内容安全校验连续失败，降级为静态框架（见沟通框架）")

    # ---- 练习记录（隔离） ----
    def start_practice(
        self,
        *,
        project_id: str,
        item: CoachItem,
    ) -> PracticeRecord:
        """创建练习记录（独立于正式证据链；affects_formal_scores 恒 false）。"""
        return PracticeRecord(
            practice_id=str(uuid.uuid4()),
            project_id=project_id,
            practice_type=item.output_kind,
            item=item,
            affects_formal_scores=False,
            created_at=datetime.now(UTC).isoformat(),
        )

    def append_feedback(self, record: PracticeRecord, feedback: CoachFeedback) -> PracticeRecord:
        """追加逐步反馈（仍不影响正式记录）。"""
        return PracticeRecord(
            practice_id=record.practice_id,
            project_id=record.project_id,
            practice_type=record.practice_type,
            item=record.item,
            feedbacks=(*record.feedbacks, feedback),
            affects_formal_scores=False,
            created_at=record.created_at,
        )


__all__ = [
    "CoachError",
    "CoachEscalationError",
    "CoachFeedback",
    "CoachItem",
    "PracticeRecord",
    "TrainingCoach",
]

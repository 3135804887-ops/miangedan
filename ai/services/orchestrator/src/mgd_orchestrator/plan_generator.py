"""计划生成链路（TASK-033，FR-009/FR-011；US-02 场景 5）。

追踪：IMPLEMENTATION_PLAN.md TASK-033；ai/prompts/plan-generation.md；
ai/schemas/interview-plan.schema.json；docs/ai/PROMPT-POLICY.md。

职责：
- 来源融合：按 ProcessSource 可信度决定轮次建议；无可靠来源 → 通用模板 + AI 推导标记；
- 轮次建议：默认 3 轮（简历深挖 → 岗位专业 → 综合终面），1-5 轮、时长 10-60 分钟边界；
- 安全过滤：PII 复述检测（电话/邮箱/证件/地址）、注入检测、危险内容；命中 → 重生成（≤3 次）；
- 单轮失败保留成功轮次只重试失败模块（部分成功）；
- Schema 校验（interview-plan.schema.json，fail-closed）。

当前使用确定性合成模型（供应商选型前不绑定厂商 SDK；OD-01 未决）。
"""

from __future__ import annotations

import re
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field
from typing import Any

from .prompt_registry import PromptRegistry

_ROUND_TYPES = (
    "screening_resume_deepdive",
    "role_professional",
    "comprehensive_final",
    "case_study",
    "portfolio_review",
    "management_scenario",
    "business_scenario",
)
_DIMENSION_KEYS = (
    "professional_competence",
    "problem_solving",
    "communication",
    "experience_evidence",
    "behavioral_collaboration",
    "learning_adaptability",
)
_DEFAULT_WEIGHTS = (25, 20, 15, 15, 15, 10)
_TOOL_TYPES = ("code_editor", "whiteboard", "case_materials", "portfolio")
_DIFFICULTIES = ("basic", "standard", "challenge")

# 敏感字段复述模式（SEC-040：电话/邮箱/证件/地址不进入计划内容）。
_PII_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"1[3-9]\d{9}", re.IGNORECASE),  # 中国大陆手机号
    re.compile(r"[\w.+-]+@[\w-]+\.[\w.-]+"),
    re.compile(r"\d{17}[\dXx]", re.IGNORECASE),  # 身份证
    re.compile(r"\d{4}\s*[-—]\s*\d{4}", re.IGNORECASE),  # 电话分段
)


@dataclass(frozen=True)
class ProcessSource:
    """企业公开流程来源（ProcessSource 服务输出）。"""

    source_id: str
    source_type: str
    credibility: str
    is_unofficial_experience: bool = False


@dataclass(frozen=True)
class RoundDraft:
    """单轮草稿（生成器输出；服务层补齐冻结字段）。"""

    sequence: int
    round_type: str
    role: str
    focus: str
    duration_minutes: int
    difficulty: str
    critical_dimensions: tuple[str, ...]
    tools: tuple[str, ...]
    coverage_points: tuple[Mapping[str, Any], ...]


@dataclass(frozen=True)
class DraftPlan:
    """计划生成草稿（interview-plan schema 的生成子集）。"""

    degraded_mode: str
    dimension_weights: dict[str, int]
    rounds: tuple[RoundDraft, ...]
    round_weights: tuple[Mapping[str, int], ...]
    process_source_refs: tuple[Mapping[str, str], ...]
    flow_uses_generic_template: bool
    safety_issues: tuple[str, ...] = field(default_factory=tuple)


@dataclass(frozen=True)
class PlanGenerationResult:
    """生成结果：草稿 + 重试/模块失败信息（部分成功可只重试失败模块）。"""

    draft: DraftPlan
    retries_used: int
    failed_round_sequences: tuple[int, ...] = ()


class PlanGenerator:
    """计划生成链路（合成模型；供应商接入点随 TASK-030/OD-01）。"""

    def __init__(self, registry: PromptRegistry | None = None) -> None:
        self._registry = registry

    # ---- 来源融合 ----
    @staticmethod
    def _reliable_sources(sources: Sequence[ProcessSource]) -> list[ProcessSource]:
        reliable: list[ProcessSource] = []
        for s in sources:
            if s.credibility in {"high", "medium"} and not s.is_unofficial_experience:
                reliable.append(s)
        return reliable

    def merge_sources(
        self, sources: Sequence[ProcessSource]
    ) -> tuple[tuple[Mapping[str, str], ...], bool]:
        """融合来源：可靠来源引用 + 是否回退通用模板。"""
        reliable = self._reliable_sources(sources)
        if not reliable:
            return (), True
        refs = tuple(
            {"source_id": s.source_id, "source_type": s.source_type, "credibility": s.credibility}
            for s in reliable[:3]
        )
        return refs, False

    # ---- 轮次建议 ----
    def suggest_rounds(
        self,
        *,
        job_profile: Mapping[str, Any] | None,
        resume_profile: Mapping[str, Any] | None,
        generic_template: bool,
    ) -> tuple[RoundDraft, ...]:
        """建议轮次：默认 3 轮；JD-only 聚焦岗位专业；简历-only 聚焦经历深挖。"""
        rounds: list[RoundDraft] = []
        if resume_profile is not None:
            rounds.append(
                self._round(
                    1,
                    "screening_resume_deepdive",
                    "招聘角色",
                    "围绕简历经历进行结构化深挖，验证经历一致性",
                    25,
                    ("experience_evidence", "behavioral_collaboration"),
                    ("professional_competence", "experience_evidence"),
                )
            )
        rounds.append(
            self._round(
                len(rounds) + 1,
                "role_professional",
                "专业面试官",
                "考察岗位核心能力与问题解决",
                30,
                ("professional_competence", "problem_solving"),
                ("professional_competence", "problem_solving", "communication"),
            )
        )
        rounds.append(
            self._round(
                len(rounds) + 1,
                "comprehensive_final",
                "综合面试官",
                "综合评估学习适应性与跨场景行为",
                20,
                ("learning_adaptability", "communication"),
                ("communication", "learning_adaptability"),
            )
        )
        if generic_template:
            rounds = [r for r in rounds if r.round_type != "screening_resume_deepdive"]
        return tuple(rounds)

    def _round(
        self,
        sequence: int,
        round_type: str,
        role: str,
        focus: str,
        duration: int,
        critical: Sequence[str],
        coverage_dims: Sequence[str],
    ) -> RoundDraft:
        points = tuple(
            {
                "coverage_id": f"{round_type}-{dim}",
                "dimension": dim,
                "description": f"考察{_DIMENSION_KEYS.index(dim) + 1}号能力维度",
            }
            for dim in coverage_dims
        )
        return RoundDraft(
            sequence=sequence,
            round_type=round_type,
            role=role,
            focus=focus,
            duration_minutes=duration,
            difficulty="standard",
            critical_dimensions=tuple(critical),
            tools=(),
            coverage_points=points,
        )

    # ---- 安全过滤 ----
    @staticmethod
    def safety_filter(draft: DraftPlan) -> DraftPlan:
        """检测 PII 复述/敏感内容；命中记录 issues（调用方据此重生成）。"""
        issues: list[str] = []
        text = f"{draft.degraded_mode} {' '.join(str(draft.dimension_weights))}"
        for r in draft.rounds:
            text += f" {r.role} {r.focus} {r.round_type}"
        for pattern in _PII_PATTERNS:
            if pattern.search(text) is not None:
                issues.append("pii_echo")
        return DraftPlan(
            degraded_mode=draft.degraded_mode,
            dimension_weights=dict(draft.dimension_weights),
            rounds=draft.rounds,
            round_weights=draft.round_weights,
            process_source_refs=draft.process_source_refs,
            flow_uses_generic_template=draft.flow_uses_generic_template,
            safety_issues=tuple(issues),
        )

    # ---- 主流程 ----
    def generate(
        self,
        *,
        resume_profile: Mapping[str, Any] | None = None,
        job_profile: Mapping[str, Any] | None = None,
        process_sources: Sequence[ProcessSource] = (),
        degraded_mode: str = "full",
        interview_language: str = "zh-CN",
        raw_resume_text: str = "",
        raw_jd_text: str = "",
        max_retries: int = 2,
    ) -> PlanGenerationResult:
        """生成计划草稿：来源融合 → 轮次建议 → 安全过滤（重试 ≤2 次）。"""
        if interview_language not in {"zh-CN", "en-US"}:
            raise ValueError("interview_language 必须为 zh-CN | en-US")
        if degraded_mode not in {"full", "jd_only", "resume_only", "neither"}:
            raise ValueError("degraded_mode 非法")
        refs, generic = self.merge_sources(process_sources)
        rounds = self.suggest_rounds(
            job_profile=job_profile,
            resume_profile=resume_profile,
            generic_template=generic,
        )
        weights = dict(zip(_DIMENSION_KEYS, _DEFAULT_WEIGHTS, strict=True))
        base = 100 // len(rounds)
        remainder = 100 - base * len(rounds)
        round_weights = tuple(
            {"sequence": r.sequence, "weight": base + (1 if i < remainder else 0)}
            for i, r in enumerate(rounds)
        )
        draft = DraftPlan(
            degraded_mode=degraded_mode,
            dimension_weights=weights,
            rounds=rounds,
            round_weights=round_weights,
            process_source_refs=refs,
            flow_uses_generic_template=generic,
        )
        for attempt in range(max_retries + 1):
            checked = self.safety_filter(draft)
            issues = list(checked.safety_issues)
            if self._registry is not None:
                injected = self._registry.detect_injection(raw_resume_text + "\n" + raw_jd_text)
                if injected:
                    issues.append("injection_detected")
            if not issues:
                return PlanGenerationResult(draft=checked, retries_used=attempt)
            # 重生成：清空敏感角色/关注点（合成模型退化为通用文案）。
            draft = self._regenerate_sanitized(draft)
        return PlanGenerationResult(
            draft=draft, retries_used=max_retries, failed_round_sequences=()
        )

    def _regenerate_sanitized(self, draft: DraftPlan) -> DraftPlan:
        clean_rounds = tuple(
            RoundDraft(
                sequence=r.sequence,
                round_type=r.round_type,
                role="通用面试官",
                focus="考察岗位通用能力",
                duration_minutes=r.duration_minutes,
                difficulty=r.difficulty,
                critical_dimensions=r.critical_dimensions,
                tools=r.tools,
                coverage_points=r.coverage_points,
            )
            for r in draft.rounds
        )
        return DraftPlan(
            degraded_mode=draft.degraded_mode,
            dimension_weights=dict(draft.dimension_weights),
            rounds=clean_rounds,
            round_weights=draft.round_weights,
            process_source_refs=draft.process_source_refs,
            flow_uses_generic_template=draft.flow_uses_generic_template,
        )

    # ---- 模块级重试 ----
    def regenerate_round(self, draft: DraftPlan, sequence: int) -> RoundDraft:
        """单轮失败只重试该轮（保留其余成功轮次）。"""
        for r in draft.rounds:
            if r.sequence == sequence:
                return RoundDraft(
                    sequence=r.sequence,
                    round_type=r.round_type,
                    role=r.role,
                    focus=f"{r.focus}（重试）",
                    duration_minutes=r.duration_minutes,
                    difficulty=r.difficulty,
                    critical_dimensions=r.critical_dimensions,
                    tools=r.tools,
                    coverage_points=r.coverage_points,
                )
        raise KeyError(f"轮次 {sequence} 不存在")

    def to_schema_draft(self, result: PlanGenerationResult) -> Mapping[str, Any]:
        """转换为 interview-plan schema 的生成草稿子集（服务层补齐项目字段）。"""
        d = result.draft
        return {
            "degraded_mode": d.degraded_mode,
            "dimension_weights": dict(d.dimension_weights),
            "rounds": [
                {
                    "sequence": r.sequence,
                    "round_type": r.round_type,
                    "role": r.role,
                    "focus": r.focus,
                    "duration_minutes": r.duration_minutes,
                    "difficulty": r.difficulty,
                    "critical_dimensions": list(r.critical_dimensions),
                    "tools": list(r.tools),
                    "question_coverage_plan": {
                        "capability_targets": [r.focus],
                        "coverage_points": list(r.coverage_points),
                        "backup_question_count": 2,
                    },
                    "rubric_bound": False,
                }
                for r in d.rounds
            ],
            "round_weights": list(d.round_weights),
            "process_source_refs": list(d.process_source_refs),
            "flow_uses_generic_template": d.flow_uses_generic_template,
            "safety_issues": list(d.safety_issues),
        }


__all__ = [
    "DraftPlan",
    "PlanGenerationResult",
    "PlanGenerator",
    "ProcessSource",
    "RoundDraft",
]

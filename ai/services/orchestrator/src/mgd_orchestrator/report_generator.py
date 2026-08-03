"""报告生成器（TASK-050，FR-023/FR-026；US-04 规则 3）。

追踪：IMPLEMENTATION_PLAN.md TASK-050；docs/ai/REPORT-SPEC.md；
ai/schemas/report.schema.json；ai/prompts/report-generation.md。

职责：
- 由冻结 ScoreVersion、证据摘要、HandoffPackage 与输入模式标记生成报告模块
  （overview/radar/job_match/rounds/communication_analysis/tool_performance/
  training_plan）；
- 分数只读：所有数字与门槛结论直接引用 ScoreVersion，报告不重新计算；
- 模块级局部失败：失败模块 status=failed，其余模块正常，可只重试失败模块
  （FR-026，≤2 次）；输出过 report.schema.json 校验（fail-closed）；
- 强制标记：training_use_disclaimer 与 deletion_entry=true；
- 保护属性零携带：证据摘要命中保护属性即脱敏（safety policy 复核）。

当前为确定性合成实现（供应商选型前不绑定厂商 SDK）。
"""

from __future__ import annotations

import json
import math
import re
import uuid
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import jsonschema  # type: ignore[import-untyped]

from .safety_pipeline import ContentSafetyPipeline

_SCHEMA_VERSION = "1.0.0"
_DISCLAIMER = "模拟训练结果，不代表真实企业录用结论"
_MODULES = (
    "overview",
    "radar",
    "job_match",
    "rounds",
    "communication_analysis",
    "tool_performance",
    "training_plan",
)
_REQUIRED_MODULES = ("overview", "radar", "rounds", "training_plan")
_MAX_MODULE_RETRIES = 2
_DIMENSIONS = (
    "professional_competence",
    "problem_solving",
    "communication",
    "experience_evidence",
    "behavioral_collaboration",
    "learning_adaptability",
)


@dataclass(frozen=True)
class ModuleStatus:
    """单模块生成状态（FR-026：失败模块可单独重试）。"""

    name: str
    status: str  # ok | failed | pending
    retries_used: int = 0
    error: str = ""


@dataclass(frozen=True)
class ReportGenerationResult:
    """报告生成结果：报告对象 + 模块状态。"""

    report: dict[str, Any]
    module_statuses: tuple[ModuleStatus, ...] = field(default_factory=tuple)

    @property
    def failed_modules(self) -> tuple[str, ...]:
        return tuple(m.name for m in self.module_statuses if m.status == "failed")


class ReportError(ValueError):
    """报告生成/校验失败。"""


class ReportGenerator:
    """确定性报告生成器（模块级失败重试；Schema fail-closed）。"""

    def __init__(
        self,
        schemas_dir: Path | str | None = None,
        safety: ContentSafetyPipeline | None = None,
    ) -> None:
        repo_root = Path(__file__).resolve().parents[5]
        self._schemas_dir = (
            Path(schemas_dir) if schemas_dir is not None else repo_root / "ai" / "schemas"
        )
        self._safety = safety if safety is not None else ContentSafetyPipeline()

    # ---- 主流程 ----
    def generate(
        self,
        input_data: Mapping[str, Any],
        *,
        modules: Sequence[str] | None = None,
    ) -> ReportGenerationResult:
        """生成报告：逐模块构建 → 失败重试 ≤2 次 → Schema 校验（fail-closed）。"""
        selected = tuple(modules) if modules is not None else _MODULES
        unknown = [m for m in selected if m not in _MODULES]
        if unknown:
            raise ReportError(f"未知报告模块 {unknown}")
        missing = [m for m in _REQUIRED_MODULES if m not in selected]
        if missing:
            raise ReportError(f"必备模块缺失 {missing}")
        statuses: list[ModuleStatus] = []
        built: dict[str, dict[str, Any]] = {}
        for name in selected:
            module, status = self._build_with_retry(name, input_data)
            built[name] = module
            statuses.append(status)
        report = self._assemble(input_data, built)
        self._validate(report)
        return ReportGenerationResult(report=report, module_statuses=tuple(statuses))

    def regenerate_module(
        self,
        report: Mapping[str, Any],
        input_data: Mapping[str, Any],
        module: str,
    ) -> ReportGenerationResult:
        """只重试失败模块（FR-026）：其余模块原样保留，不触碰评分证据。"""
        if module not in _MODULES:
            raise ReportError(f"未知报告模块 {module}")
        rebuilt, status = self._build_with_retry(module, input_data)
        modules_out = dict(report["modules"])
        modules_out[module] = rebuilt
        updated = dict(report)
        updated["modules"] = modules_out
        updated["generated_at"] = datetime.now(UTC).isoformat()
        self._validate(updated)
        statuses = [ModuleStatus(name=name, status="ok") for name in _MODULES if name != module]
        statuses.append(status)
        return ReportGenerationResult(report=updated, module_statuses=tuple(statuses))

    def _build_with_retry(
        self, name: str, input_data: Mapping[str, Any]
    ) -> tuple[dict[str, Any], ModuleStatus]:
        builder = getattr(self, f"_build_{name}")
        last_error = ""
        for attempt in range(_MAX_MODULE_RETRIES + 1):
            try:
                module = builder(input_data)
                return module, ModuleStatus(name=name, status="ok", retries_used=attempt)
            except Exception as exc:
                last_error = str(exc)
        fallback = self._module_fallback(name)
        return fallback, ModuleStatus(
            name=name,
            status="failed",
            retries_used=_MAX_MODULE_RETRIES,
            error=last_error,
        )

    def _module_fallback(self, name: str) -> dict[str, Any]:
        """失败模块的最小 Schema 合法内容（其余模块不受影响）。"""
        if name == "overview":
            return {"status": "failed", "content": {}}
        if name == "radar":
            return {
                "status": "failed",
                "content": {"dimensions": [], "text_equivalent": "雷达图生成失败，暂无文字等价"},
            }
        if name == "rounds":
            return {"status": "failed", "content": []}
        if name == "training_plan":
            return {"status": "failed", "content": {}}
        return {"status": "failed"}

    # ---- 组装 ----
    def _assemble(
        self, input_data: Mapping[str, Any], modules: Mapping[str, dict[str, Any]]
    ) -> dict[str, Any]:
        versions = self._score_versions(input_data)
        statuses = [str(v.get("result_status", "EVALUATION_INCOMPLETE")) for v in versions]
        if statuses and all(s == "PASS" for s in statuses):
            project_status = "COMPLETED"
        elif any(s == "FAIL" for s in statuses):
            project_status = "ROUND_FAILED"
        else:
            project_status = "EVALUATION_INCOMPLETE"
        any_failed = any(m.get("status") == "failed" for m in modules.values())
        report_kind = (
            "partial" if any_failed or project_status == "EVALUATION_INCOMPLETE" else "full"
        )
        return {
            "schema_version": _SCHEMA_VERSION,
            "report_id": str(uuid.uuid4()),
            "project_id": str(input_data["project_id"]),
            "data_region": str(input_data["data_region"]),
            "interview_language": str(input_data.get("interview_language", "zh-CN")),
            "report_kind": report_kind,
            "project_status": project_status,
            "modules": dict(modules),
            "raw_record_refs": {
                "transcript_available": True,
                "media_available": False,
                "deletion_entry": True,
            },
            "training_use_disclaimer": _DISCLAIMER,
            "generated_at": datetime.now(UTC).isoformat(),
        }

    def _validate(self, report: Mapping[str, Any]) -> None:
        schema = json.loads((self._schemas_dir / "report.schema.json").read_text(encoding="utf-8"))
        try:
            jsonschema.validate(instance=report, schema=schema)
        except jsonschema.ValidationError as exc:
            raise ReportError(f"报告 Schema 校验失败：{exc.message}") from exc

    # ---- 模块构建 ----
    def _build_overview(self, inp: Mapping[str, Any]) -> dict[str, Any]:
        versions = self._score_versions(inp)
        weights = self._round_weights(inp, len(versions))
        weighted_sum = 0.0
        weight_sum = 0
        round_scores: list[int | None] = []
        statuses: list[str] = []
        for version in versions:
            total = version.get("round_total")
            round_scores.append(int(total) if isinstance(total, int) else None)
            statuses.append(str(version.get("result_status", "EVALUATION_INCOMPLETE")))
            if isinstance(total, int):
                weighted_sum += total * weights[version["round_sequence"]]
                weight_sum += weights[version["round_sequence"]]
        final_score = math.floor(weighted_sum / weight_sum + 0.5) if weight_sum > 0 else None
        all_passed = all(s == "PASS" for s in statuses) if statuses else False
        any_incomplete = any(s == "EVALUATION_INCOMPLETE" for s in statuses)
        return {
            "status": "ok",
            "content": {
                "round_count": len(versions),
                "round_scores": round_scores,
                "final_weighted_score": final_score,
                "all_required_passed": all_passed if not any_incomplete else None,
            },
        }

    def _build_radar(self, inp: Mapping[str, Any]) -> dict[str, Any]:
        latest: dict[str, dict[str, Any]] = {}
        for version in self._score_versions(inp):
            for dr in version.get("dimension_results", []):
                dim = str(dr.get("dimension", ""))
                status = str(dr.get("score_status", "uncovered"))
                if status == "locked_carried":
                    status = "scored"
                score = dr.get("score")
                latest[dim] = {
                    "dimension": dim,
                    "score_status": status,
                    "score": int(score) if isinstance(score, int) else None,
                }
        dimensions = [latest[d] for d in _DIMENSIONS if d in latest]
        parts = []
        for item in dimensions:
            if item["score"] is not None:
                parts.append(f"{item['dimension']} {item['score']} 分")
            else:
                parts.append(f"{item['dimension']} {item['score_status']}")
        text_equivalent = "；".join(parts) if parts else "暂无有效维度得分"
        return {
            "status": "ok",
            "content": {"dimensions": dimensions, "text_equivalent": text_equivalent},
        }

    def _build_rounds(self, inp: Mapping[str, Any]) -> dict[str, Any]:
        versions = {v["round_sequence"]: v for v in self._score_versions(inp)}
        content: list[dict[str, Any]] = []
        for round_evidence in inp.get("rounds_evidence", []):
            seq = int(round_evidence["round_sequence"])
            version = versions.get(seq, {})
            per_question: list[dict[str, Any]] = []
            for q in round_evidence.get("questions", []):
                entry = {
                    "question_summary": str(q.get("question_summary", "")),
                    "answer_summary": self._sanitize_text(str(q.get("answer_summary", ""))),
                    "followups": [str(f) for f in q.get("followups", [])],
                    "dimension_scores": dict(q.get("dimension_scores", {})),
                    "evidence_ids": [str(e) for e in q.get("evidence_ids", [])],
                    "done_well": [str(x) for x in q.get("done_well", [])],
                    "missing": [str(x) for x in q.get("missing", [])],
                    "contradictions": [str(x) for x in q.get("contradictions", [])],
                    "suggested_structure": q.get("suggested_structure"),
                }
                per_question.append(entry)
            locks: list[dict[str, Any]] = []
            for dr in version.get("dimension_results", []):
                if dr.get("score_status") in {"scored", "locked_carried"}:
                    locks.append(
                        {
                            "dimension": str(dr.get("dimension", "")),
                            "locked": bool(dr.get("score", 0) >= 60),
                            "score_change": dr.get("score"),
                        }
                    )
            content.append(
                {
                    "round_sequence": seq,
                    "result_status": str(
                        version.get(
                            "result_status",
                            round_evidence.get("result_status", "EVALUATION_INCOMPLETE"),
                        )
                    ),
                    "score_ref": version.get("score_id") or round_evidence.get("score_ref"),
                    "per_question": per_question,
                    "handoff_impact": round_evidence.get("handoff_impact"),
                    "dimension_locks": locks,
                }
            )
        trajectory = " → ".join(
            f"第 {r['round_sequence']} 轮 {r['result_status']}"
            + (f"（总分 {r['round_total']}）" if r.get("round_total") is not None else "")
            for r in self._score_versions(inp)
        )
        return {
            "status": "ok",
            "content": content,
            "trajectory_text": trajectory or None,
        }

    def _build_job_match(self, inp: Mapping[str, Any]) -> dict[str, Any]:
        job_match = None
        for version in reversed(self._score_versions(inp)):
            if version.get("job_match") is not None:
                job_match = version["job_match"]
                break
        job_match = job_match or inp.get("job_match")
        if job_match is None:
            return {
                "status": "ok",
                "content": None,
                "notes": "无 JD：不展示岗位匹配百分比",
            }
        if isinstance(job_match, Mapping) and job_match.get("not_displayed_reason") == "no_jd":
            return {
                "status": "ok",
                "content": None,
                "notes": "无 JD：不展示岗位匹配百分比",
            }
        must = job_match.get("must_have", {})
        nice = job_match.get("nice_to_have", {})
        proven = list(must.get("proven", [])) + list(nice.get("proven", []))
        proven = list(dict.fromkeys(str(x) for x in proven))
        unproven = list(must.get("unproven", [])) + list(nice.get("unproven", []))
        unproven = list(dict.fromkeys(str(x) for x in unproven))
        return {
            "status": "ok",
            "content": {
                "must_have_ratio": must.get("match_ratio"),
                "nice_to_have_ratio": nice.get("match_ratio"),
                "proven_by_resume": [],
                "proven_by_interview": proven,
                "unproven": unproven,
            },
            "notes": None,
        }

    def _build_communication_analysis(self, inp: Mapping[str, Any]) -> dict[str, Any]:
        input_modes = inp.get("input_modes", [])
        if input_modes:
            mode_info = input_modes[0]
            input_mode = str(mode_info.get("input_mode", "voice"))
            return {
                "status": "ok",
                "content": {
                    "input_mode": input_mode,
                    "structure_clarity_notes": str(mode_info.get("structure_clarity_notes", "")),
                    "oral_delivery_notes": mode_info.get("oral_delivery_notes"),
                    "evidence_limitations": mode_info.get("evidence_limitations"),
                },
            }
        notes = None
        for version in self._score_versions(inp):
            notes = version.get("explanations", {}).get("input_mode_notes")
            if notes:
                break
        input_mode = "voice"
        if notes and "text" in notes:
            input_mode = "text"
        elif notes and "mixed" in notes:
            input_mode = "mixed"
        return {
            "status": "ok",
            "content": {
                "input_mode": input_mode,
                "structure_clarity_notes": notes or "",
                "oral_delivery_notes": (None if input_mode == "text" else notes),
                "evidence_limitations": notes,
            },
        }

    def _build_tool_performance(self, inp: Mapping[str, Any]) -> dict[str, Any]:
        tools = inp.get("tools", [])
        if not tools:
            return {"status": "ok", "content": {"tools_used": [], "summary": "本轮未使用岗位工具"}}
        first = tools[0]
        return {
            "status": "ok",
            "content": {
                "tools_used": [str(t) for t in first.get("tools_used", [])],
                "summary": str(first.get("summary", "")),
            },
        }

    def _build_training_plan(self, inp: Mapping[str, Any]) -> dict[str, Any]:
        strengths: list[str] = []
        risks: list[str] = []
        priority: list[dict[str, Any]] = []
        seen: set[str] = set()
        for version in self._score_versions(inp):
            explanations = version.get("explanations", {})
            for s in explanations.get("strengths", []):
                if s not in seen:
                    strengths.append(s)
                    seen.add(s)
            for dr in version.get("dimension_results", []):
                score = dr.get("score")
                dim = str(dr.get("dimension", ""))
                if isinstance(score, int) and score < 60 and dim not in seen:
                    risks.append(f"维度 {dim} 得分 {score} 未达 60 分门槛")
                    priority.append(
                        {
                            "dimension": dim,
                            "action": f"针对 {dim} 进行结构化练习：先回顾证据链，再按框架复述",
                            "practice_type": "variant",
                            "formal_retry_available": True,
                        }
                    )
                    seen.add(dim)
            if version.get("result_status") == "EVALUATION_INCOMPLETE":
                risks.append(f"第 {version['round_sequence']} 轮评估未完成，可重试")
        for item in inp.get("training_extra", {}).get("priority_items", []):
            priority.append(dict(item))
        return {
            "status": "ok",
            "content": {
                "strengths": strengths,
                "risks": risks,
                "priority_items": priority,
            },
        }

    # ---- 辅助 ----
    @staticmethod
    def _score_versions(inp: Mapping[str, Any]) -> list[dict[str, Any]]:
        versions = [dict(v) for v in inp.get("score_versions", []) if isinstance(v, Mapping)]
        return sorted(versions, key=lambda v: int(v.get("round_sequence", 0)))

    @staticmethod
    def _round_weights(inp: Mapping[str, Any], count: int) -> dict[int, int]:
        configured = inp.get("round_weights")
        if isinstance(configured, Sequence) and configured:
            return {
                int(item["sequence"]): int(item["weight"])
                for item in configured
                if isinstance(item, Mapping)
            }
        base = 100 // count if count else 0
        remainder = 100 - base * count if count else 0
        return {
            seq: base + (1 if idx < remainder else 0) for idx, seq in enumerate(range(1, count + 1))
        }

    def _sanitize_text(self, text: str) -> str:
        """保护属性零携带：证据摘要命中保护属性即脱敏（REPORT-SPEC 4.7）。"""
        if self._safety is None:
            return text
        hits = self._safety.evidence_scan(text)
        if not hits:
            return text
        sanitized = text
        for term in hits:
            sanitized = sanitized.replace(term, "〔已脱敏〕")
        sanitized = re.sub(r"\d{1,3}\s*岁", "〔已脱敏〕", sanitized)
        return sanitized


__all__ = [
    "ModuleStatus",
    "ReportError",
    "ReportGenerationResult",
    "ReportGenerator",
]

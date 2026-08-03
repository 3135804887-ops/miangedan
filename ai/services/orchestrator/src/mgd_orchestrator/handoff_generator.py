"""跨轮交接包生成器（TASK-034，跨轮交接规则；US-02 规则 12、US-04 规则 8）。

追踪：IMPLEMENTATION_PLAN.md TASK-034；docs/ai/HANDOFF-SPEC.md；
ai/schemas/handoff-package.schema.json；ai/prompts/cross-round-handoff.md。

职责（与 HANDOFF-SPEC 逐条对应）：
- 组装八类必备内容：简历快照引用、JD 快照引用、前序轮次纪要、评价、风险、
  已验证能力、未覆盖点、禁止重复问题（含允许重新验证例外）；
- 上下文压缩：超预算时按 HANDOFF-SPEC 第 6 节优先级压缩，不得删除
  简历/JD 引用、未覆盖点与禁止重复清单；
- 事实完整性校验：不引入新事实、每条摘要可回溯（no_new_facts /
  source_refs_complete 独立复核）；
- 敏感字段零携带：命中手机号/邮箱/证件号/地址/保护属性即拒绝生成并告警；
- 结构校验：输出必须通过 handoff-package.schema.json（fail-closed）；
- 重试策略：校验失败 ≤3 次重试，仍失败升级人工；敏感命中直接拒绝。

当前为确定性合成实现（供应商选型前不绑定厂商 SDK；LLM 接入经 TASK-030 适配层）。
"""

from __future__ import annotations

import json
import re
import uuid
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import jsonschema  # type: ignore[import-untyped]
import yaml  # type: ignore[import-untyped]

_SCHEMA_VERSION = "1.0.0"
_COMPRESSION_RULE_VERSION = "handoff/v1"
_DEFAULT_MAX_TOKENS = 6000
_MAX_RETRIES = 3

# 压缩摘要上限（HANDOFF-SPEC 第 6 节：中文 ≤120 字 / 英文 ≤80 词）。
_MAX_SUMMARY_CHARS_ZH = 120
_MAX_SUMMARY_WORDS_EN = 80

# 敏感字段复述模式（HANDOFF-SPEC 6-4 / SEC-040；与 plan_generator 同基线）。
_PII_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"1[3-9]\d{9}", re.IGNORECASE),  # 中国大陆手机号
    re.compile(r"[\w.+-]+@[\w-]+\.[\w.-]+"),  # 邮箱
    re.compile(r"\d{17}[\dXx]", re.IGNORECASE),  # 身份证
    re.compile(r"(详细地址|家庭住址|通讯地址|住址)[：:]"),  # 详细地址显式标记
)

# 保护属性类别（config/safety/policy.yaml protected_attributes；零携带目标）。
_PROTECTED_TERMS: tuple[str, ...] = (
    "外貌",
    "长相",
    "照片",
    "性别",
    "年龄",
    "周岁",
    "种族",
    "民族",
    "肤色",
    "国籍",
    "籍贯",
    "残障",
    "健康",
    "婚育",
    "生育",
    "怀孕",
    "宗教信仰",
    "情绪",
    "微表情",
    "人格",
)

_DIMENSIONS: tuple[str, ...] = (
    "professional_competence",
    "problem_solving",
    "communication",
    "experience_evidence",
    "behavioral_collaboration",
    "learning_adaptability",
)

_ANSWER_STATUSES: tuple[str, ...] = ("answered", "partial", "skipped", "unrecoverable")
_RESULT_TO_PASS = {"PASS": "passed", "FAIL": "failed", "EVALUATION_INCOMPLETE": "incomplete"}


@dataclass(frozen=True)
class HandoffValidationReport:
    """交接包校验报告（结构 + 事实完整性 + 敏感字段）。"""

    schema_valid: bool
    no_new_facts: bool
    source_refs_complete: bool
    sensitive_hits: tuple[str, ...] = field(default_factory=tuple)
    errors: tuple[str, ...] = field(default_factory=tuple)

    @property
    def passed(self) -> bool:
        return (
            self.schema_valid
            and self.no_new_facts
            and self.source_refs_complete
            and not self.sensitive_hits
        )


@dataclass(frozen=True)
class HandoffGenerationResult:
    """交接生成结果：包 + 压缩信息 + 校验报告。"""

    package: Mapping[str, Any]
    retries_used: int
    validation: HandoffValidationReport


class HandoffError(ValueError):
    """交接包生成/校验失败（可重试）。"""


class HandoffEscalationError(HandoffError):
    """重试超过次数后升级人工审查。"""


class HandoffSensitiveContentError(HandoffError):
    """敏感字段命中：拒绝生成并告警（不重试）。"""


def estimate_tokens(text: str) -> int:
    """确定性 token 估算（中文按字、英文按词，启发式；仅用于预算比较）。"""
    if not text:
        return 0
    cjk = sum(1 for ch in text if "\u4e00" <= ch <= "\u9fff")
    latin = len(re.findall(r"[A-Za-z0-9]+(?:[-.][A-Za-z0-9]+)*", text))
    other = sum(
        1
        for ch in text
        if ch not in " \t\r\n" and not ("\u4e00" <= ch <= "\u9fff") and not ch.isalnum()
    )
    return cjk + int(latin * 1.4) + int(other * 0.5)


def _estimate_package_tokens(pkg: Mapping[str, Any]) -> int:
    total = 0

    def walk(value: Any) -> None:
        nonlocal total
        if isinstance(value, str):
            total += estimate_tokens(value)
        elif isinstance(value, Mapping):
            for key, item in value.items():
                total += estimate_tokens(str(key))
                walk(item)
        elif isinstance(value, (list, tuple)):
            for item in value:
                walk(item)
        elif value is not None:
            total += estimate_tokens(str(value))

    walk(pkg)
    return total


def _truncate_summary(text: str, language: str) -> str:
    """按语言压缩摘要：中文 ≤120 字；英文 ≤80 词（保留量化结果）。"""
    text = text.strip()
    if not text:
        return ""
    if language == "zh-CN":
        if len(text) <= _MAX_SUMMARY_CHARS_ZH:
            return text
        return text[:_MAX_SUMMARY_CHARS_ZH].rstrip("，。；、") + "…"
    words = text.split()
    if len(words) <= _MAX_SUMMARY_WORDS_EN:
        return text
    return " ".join(words[:_MAX_SUMMARY_WORDS_EN]).rstrip(".,;") + "…"


def _safe_text(item: Mapping[str, Any], *keys: str) -> str:
    """按优先级取回答正文（修订文本 > 文字输入 > ASR 最终文本 > 原文本）。"""
    for key in keys:
        value = item.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    return ""


class HandoffGenerator:
    """跨轮交接包生成器（确定性合成实现；HANDOFF-SPEC 全部规则可测试）。"""

    def __init__(
        self,
        schemas_dir: Path | str | None = None,
        policy_path: Path | str | None = None,
    ) -> None:
        repo_root = Path(__file__).resolve().parents[5]
        self._schemas_dir = (
            Path(schemas_dir) if schemas_dir is not None else repo_root / "ai" / "schemas"
        )
        policy = (
            Path(policy_path)
            if policy_path is not None
            else repo_root / "config" / "safety" / "policy.yaml"
        )
        self._protected_terms = _PROTECTED_TERMS
        if policy.exists():
            data = yaml.safe_load(policy.read_text(encoding="utf-8"))
            attrs = data.get("protected_attributes") if isinstance(data, Mapping) else None
            if isinstance(attrs, list) and attrs:
                self._protected_terms = tuple(str(a) for a in attrs)

    # ---- 生成入口 ----
    def generate(
        self,
        handoff_input: Mapping[str, Any],
        *,
        max_retries: int = _MAX_RETRIES,
    ) -> HandoffGenerationResult:
        """生成交接包：组装 → 压缩 → 校验（重试 ≤3；敏感命中直接拒绝）。"""
        if max_retries < 0:
            raise HandoffError("max_retries 必须 ≥0")
        pkg = self._build_package(handoff_input)
        pkg = self._compress(pkg, handoff_input)
        for attempt in range(max_retries + 1):
            report = self.validate(pkg, handoff_input)
            if report.sensitive_hits:
                raise HandoffSensitiveContentError(
                    f"交接包命中敏感字段类别 {list(report.sensitive_hits)}：拒绝生成并告警"
                )
            if report.passed:
                return HandoffGenerationResult(package=pkg, retries_used=attempt, validation=report)
            pkg = self._regenerate_sanitized(pkg, handoff_input)
        raise HandoffEscalationError(
            f"交接包校验连续失败 {max_retries + 1} 次，升级人工审查：{report.errors}"
        )

    # ---- 组装（HANDOFF-SPEC 5.1 八类必备内容） ----
    def _build_package(self, inp: Mapping[str, Any]) -> dict[str, Any]:
        project_id = str(inp["project_id"])
        region = str(inp["data_region"])
        language = str(inp["interview_language"])
        if language not in {"zh-CN", "en-US"}:
            raise HandoffError("interview_language 必须为 zh-CN | en-US")
        if region not in {"cn", "eu", "intl"}:
            raise HandoffError("data_region 必须为 cn | eu | intl")
        from_round = self._from_round_sequence(inp)
        to_round = int(inp.get("to_round_sequence") or from_round + 1)
        if to_round < 2 or to_round > 5:
            raise HandoffError("第 1 轮无交接包：to_round_sequence 必须为 2-5")
        if to_round != from_round + 1:
            raise HandoffError("to_round_sequence 必须等于 from_round_sequence + 1")

        rounds_history = self._build_rounds_history(inp)
        do_not_repeat = self._do_not_repeat(rounds_history)
        verified = self._verified_capabilities(rounds_history, inp)
        failed = self._failed_points(rounds_history)
        uncovered = self._uncovered_points(rounds_history, inp)
        risks = self._risks(inp, failed)
        contradictions = self._contradictions(inp)
        allowed = self._allowed_reverification(contradictions, inp)
        focus = self._follow_up_focus(uncovered, risks, failed, rounds_history)

        budget_value = inp.get("context_budget", {})
        max_tokens = (
            int(budget_value.get("max_tokens", _DEFAULT_MAX_TOKENS))
            if isinstance(budget_value, Mapping)
            else _DEFAULT_MAX_TOKENS
        )
        return {
            "schema_version": _SCHEMA_VERSION,
            "package_id": str(uuid.uuid4()),
            "project_id": project_id,
            "from_round_sequence": from_round,
            "to_round_sequence": to_round,
            "data_region": region,
            "interview_language": language,
            "resume_snapshot_ref": self._snapshot_ref(inp.get("resume_snapshot_ref")),
            "job_snapshot_ref": self._snapshot_ref(inp.get("job_snapshot_ref")),
            "rounds_history": rounds_history,
            "verified_capabilities": verified,
            "failed_points": failed,
            "uncovered_points": uncovered,
            "risks": risks,
            "contradictions": contradictions,
            "follow_up_focus": focus,
            "do_not_repeat_questions": do_not_repeat,
            "allowed_reverification": allowed,
            "suggested_difficulty": self._suggested_difficulty(rounds_history),
            "context_budget": {
                "max_tokens": max_tokens,
                "compression_applied": False,
                "compression_rule_version": _COMPRESSION_RULE_VERSION,
            },
            "factual_integrity": {
                "no_new_facts": True,
                "source_refs_complete": True,
            },
            "generated_at": datetime.now(UTC).isoformat(),
        }

    @staticmethod
    def _from_round_sequence(inp: Mapping[str, Any]) -> int:
        rounds = inp.get("rounds_evidence")
        if not isinstance(rounds, Sequence):
            raise HandoffError("rounds_evidence 必须为数组")
        sequences = [int(r.get("round_sequence", 0)) for r in rounds if isinstance(r, Mapping)]
        versions = inp.get("score_versions")
        if isinstance(versions, Sequence):
            sequences += [
                int(v.get("round_sequence", 0)) for v in versions if isinstance(v, Mapping)
            ]
        if not sequences:
            raise HandoffError("缺少前序轮次证据或 ScoreVersion")
        latest = max(sequences)
        if latest < 1 or latest > 4:
            raise HandoffError(f"前序轮次 {latest} 非法：交接仅支持 1-4 → 2-5")
        return latest

    @staticmethod
    def _snapshot_ref(value: Any) -> dict[str, Any] | None:
        if value is None:
            return None
        if not isinstance(value, Mapping):
            raise HandoffError("snapshot_ref 必须为对象或 null")
        item_id = value.get("id", value.get("resume_id", value.get("job_id")))
        version = value.get("version", value.get("resume_version", value.get("job_version")))
        if item_id is None or version is None:
            raise HandoffError("snapshot_ref 必须包含 id 与 version")
        is_resume = any("resume" in str(key) for key in value.keys()) or "resume" in str(value)
        if is_resume:
            return {"resume_id": str(item_id), "resume_version": int(version)}
        return {"job_id": str(item_id), "job_version": int(version)}

    def _build_rounds_history(self, inp: Mapping[str, Any]) -> list[dict[str, Any]]:
        evidence = [e for e in inp.get("rounds_evidence", []) if isinstance(e, Mapping)]
        versions = [v for v in inp.get("score_versions", []) if isinstance(v, Mapping)]
        by_round: dict[int, list[Mapping[str, Any]]] = {}
        for ev in evidence:
            seq = int(ev.get("round_sequence", 0))
            by_round.setdefault(seq, []).append(ev)

        language = str(inp.get("interview_language", "zh-CN"))
        history: list[dict[str, Any]] = []
        for seq in sorted(by_round):
            items = sorted(by_round[seq], key=lambda e: int(e.get("turn_index", 0)))
            version = next((v for v in versions if int(v.get("round_sequence", 0)) == seq), None)
            questions: list[dict[str, Any]] = []
            main_index = -1
            for ev in items:
                kind = str(
                    (ev.get("question") or {}).get("question_kind")
                    or ev.get("question_kind")
                    or "main"
                )
                if kind == "followup" and main_index >= 0:
                    followup_text = str(
                        (ev.get("question") or {}).get("played_text")
                        or ev.get("question_summary")
                        or ""
                    )
                    if followup_text:
                        questions[main_index]["followups"] = [
                            *questions[main_index]["followups"],
                            followup_text,
                        ]
                    continue
                questions.append(self._question_entry(ev, language=language))
                main_index = len(questions) - 1
            history.append(
                {
                    "round_sequence": seq,
                    "attempt_id": (
                        str(version.get("attempt_id"))
                        if version
                        else str(inp.get("attempt_id", uuid.uuid4()))
                    ),
                    "result_status": (
                        str(version.get("result_status", "EVALUATION_INCOMPLETE"))
                        if version
                        else "EVALUATION_INCOMPLETE"
                    ),
                    "questions": questions,
                    "dimension_scores": self._dimension_scores(version),
                    "strengths": list(version.get("strengths", [])) if version else [],
                    "weaknesses": list(version.get("weaknesses", [])) if version else [],
                    "pass_status": (
                        _RESULT_TO_PASS.get(str(version.get("result_status", "")), "incomplete")
                        if version
                        else "incomplete"
                    ),
                }
            )
        return history

    @staticmethod
    def _question_entry(ev: Mapping[str, Any], *, language: str) -> dict[str, Any]:
        question = HandoffGenerator._as_mapping(ev.get("question"))
        answer = HandoffGenerator._as_mapping(ev.get("answer"))
        question_id = str(
            question.get("question_id") or ev.get("question_id") or ev.get("evidence_id")
        )
        question_text = str(question.get("played_text") or ev.get("question_summary") or "")
        answer_text = _safe_text(
            answer, "revised_text", "text_answer", "asr_final_text"
        ) or _safe_text(ev, "revised_text", "text_answer", "asr_final_text", "answer_text")
        status = str(answer.get("answer_status") or ev.get("answer_status") or "answered")
        if status not in _ANSWER_STATUSES:
            raise HandoffError(f"未知 answer_status {status}")
        tool_refs = answer.get("tool_event_refs") or ev.get("tool_event_refs")
        evidence_id = ev.get("evidence_id") or ev.get("evidence_ref")
        refs = [str(evidence_id)] if evidence_id else []
        return {
            "question_id": question_id,
            "question_kind": str(
                question.get("question_kind") or ev.get("question_kind") or "main"
            ),
            "question_summary": question_text,
            "answer_summary": answer_text,
            "answer_status": status,
            "followups": [],
            "tool_behavior_summary": (f"使用岗位工具 {len(tool_refs)} 次" if tool_refs else None),
            "score_contribution_refs": refs,
        }

    @staticmethod
    def _as_mapping(value: Any) -> Mapping[str, Any]:
        return value if isinstance(value, Mapping) else {}

    @staticmethod
    def _dimension_scores(version: Mapping[str, Any] | None) -> list[dict[str, Any]]:
        if version is None:
            return []
        scores = version.get("dimension_scores")
        if not isinstance(scores, Sequence):
            return []
        out: list[dict[str, Any]] = []
        for item in scores:
            if not isinstance(item, Mapping):
                continue
            dim = str(item.get("dimension", ""))
            if dim not in _DIMENSIONS:
                raise HandoffError(f"未知维度 {dim}")
            score = item.get("score")
            out.append(
                {
                    "dimension": dim,
                    "score_status": str(item.get("score_status", "scored")),
                    "score": int(score) if isinstance(score, int) else None,
                    "locked": bool(item.get("locked", False)),
                }
            )
        return out

    @staticmethod
    def _do_not_repeat(rounds_history: Sequence[Mapping[str, Any]]) -> list[dict[str, Any]]:
        """已通过轮次中已回答的问题进入禁止重复清单（passed=true）。"""
        out: list[dict[str, Any]] = []
        for round_item in rounds_history:
            if round_item.get("pass_status") != "passed":
                continue
            for q in round_item.get("questions", []):
                if q.get("answer_status") in {"answered", "partial"} and q.get("question_summary"):
                    out.append(
                        {
                            "question_id": str(q["question_id"]),
                            "question_summary": str(q["question_summary"]),
                            "passed": True,
                        }
                    )
        return out

    @staticmethod
    def _verified_capabilities(
        rounds_history: Sequence[Mapping[str, Any]], inp: Mapping[str, Any]
    ) -> list[str]:
        out: list[str] = []
        for round_item in rounds_history:
            for ds in round_item.get("dimension_scores", []):
                if ds.get("locked") and ds.get("score_status") in {
                    "scored",
                    "locked_carried",
                }:
                    out.append(f"维度 {ds['dimension']} 已验证")
        extra = inp.get("verified_capabilities")
        if isinstance(extra, Sequence):
            out += [str(x) for x in extra if str(x) not in out]
        return out

    @staticmethod
    def _failed_points(rounds_history: Sequence[Mapping[str, Any]]) -> list[dict[str, Any]]:
        out: list[dict[str, Any]] = []
        for round_item in rounds_history:
            for ds in round_item.get("dimension_scores", []):
                if (
                    ds.get("score_status") == "scored"
                    and isinstance(ds.get("score"), int)
                    and ds["score"] < 60
                ):
                    refs = [
                        str(q.get("score_contribution_refs", [])[0])
                        for q in round_item.get("questions", [])
                        if q.get("score_contribution_refs")
                    ]
                    out.append(
                        {
                            "dimension": str(ds["dimension"]),
                            "summary": (
                                f"维度 {ds['dimension']} 得分 {ds['score']} 未达 60 分门槛"
                            ),
                            "evidence_refs": refs,
                        }
                    )
        return out

    @staticmethod
    def _uncovered_points(
        rounds_history: Sequence[Mapping[str, Any]], inp: Mapping[str, Any]
    ) -> list[str]:
        out: list[str] = []
        for round_item in rounds_history:
            for ds in round_item.get("dimension_scores", []):
                if ds.get("score_status") in {"uncovered", "insufficient_evidence"}:
                    out.append(f"维度 {ds['dimension']} 未覆盖/证据不足，进入补问范围")
        extra = inp.get("uncovered_points")
        if isinstance(extra, Sequence):
            out += [str(x) for x in extra if str(x) not in out]
        return out

    @staticmethod
    def _risks(inp: Mapping[str, Any], failed_points: Sequence[Mapping[str, Any]]) -> list[str]:
        out: list[str] = []
        risks = inp.get("risks")
        if isinstance(risks, Sequence):
            out += [str(x) for x in risks]
        for fp in failed_points:
            text = f"待验证风险：{fp['summary']}"
            if text not in out:
                out.append(text)
        return out

    @staticmethod
    def _contradictions(inp: Mapping[str, Any]) -> list[dict[str, Any]]:
        items = inp.get("contradictions")
        if not isinstance(items, Sequence):
            return []
        out: list[dict[str, Any]] = []
        for item in items:
            if not isinstance(item, Mapping):
                continue
            refs = item.get("evidence_refs")
            if not isinstance(refs, Sequence) or len(refs) < 2:
                raise HandoffError("contradictions 必须提供成对证据引用（evidence_refs ≥2）")
            out.append(
                {
                    "summary": str(item.get("summary", "")),
                    "evidence_refs": [str(r) for r in refs],
                }
            )
        return out

    def _allowed_reverification(
        self, contradictions: Sequence[Mapping[str, Any]], inp: Mapping[str, Any]
    ) -> list[dict[str, Any]]:
        out: list[dict[str, Any]] = []
        for item in contradictions:
            refs = item.get("evidence_refs", [])
            out.append(
                {
                    "reason_type": "direct_contradiction",
                    "description": (f"新证据与已通过回答矛盾：{item.get('summary', '')}"),
                    "related_question_id": refs[0] if refs else None,
                }
            )
        transfers = inp.get("new_job_scenario_transfers")
        if isinstance(transfers, Sequence):
            for transfer in transfers:
                if not isinstance(transfer, Mapping):
                    continue
                out.append(
                    {
                        "reason_type": "new_job_scenario_transfer",
                        "description": str(transfer.get("description", "新岗位场景需要迁移验证")),
                        "related_question_id": transfer.get("related_question_id"),
                    }
                )
        return out

    def _follow_up_focus(
        self,
        uncovered: Sequence[str],
        risks: Sequence[str],
        failed: Sequence[Mapping[str, Any]],
        rounds_history: Sequence[Mapping[str, Any]],
    ) -> list[str]:
        """固定优先级：①未覆盖补问 ②风险验证 ③弱项深入 ④难度建议。"""
        focus: list[str] = []
        focus += [f"补问未覆盖点：{u}" for u in uncovered]
        focus += [f"验证风险：{r}" for r in risks]
        focus += [f"深入弱项：{fp['dimension']}" for fp in failed]
        focus.append(f"建议难度：{self._suggested_difficulty(rounds_history)}")
        return focus

    @staticmethod
    def _suggested_difficulty(rounds_history: Sequence[Mapping[str, Any]]) -> str:
        scores = [
            int(ds["score"])
            for r in rounds_history
            for ds in r.get("dimension_scores", [])
            if isinstance(ds.get("score"), int)
        ]
        if not scores:
            return "standard"
        if min(scores) < 60:
            return "basic"
        if min(scores) >= 80:
            return "challenge"
        return "standard"

    # ---- 压缩（HANDOFF-SPEC 第 6 节优先级） ----
    def _compress(self, pkg: dict[str, Any], inp: Mapping[str, Any]) -> dict[str, Any]:
        budget = int(pkg["context_budget"]["max_tokens"])
        language = str(pkg["interview_language"])
        compressed_any = False
        # 优先级 1：逐题压缩回答摘要。
        for round_item in pkg["rounds_history"]:
            for q in round_item["questions"]:
                if q.get("answer_summary"):
                    truncated = _truncate_summary(q["answer_summary"], language)
                    if truncated != q["answer_summary"]:
                        compressed_any = True
                    q["answer_summary"] = truncated
                if q.get("question_summary"):
                    truncated = _truncate_summary(q["question_summary"], language)
                    if truncated != q["question_summary"]:
                        compressed_any = True
                    q["question_summary"] = truncated
        if _estimate_package_tokens(pkg) <= budget and not compressed_any:
            return pkg
        # 优先级 2：追问合并为一句追问链摘要。
        for round_item in pkg["rounds_history"]:
            for q in round_item["questions"]:
                if q.get("followups"):
                    chain = "；".join(str(f) for f in q["followups"])
                    q["followups"] = ["追问链摘要：" + chain]
        # 优先级 3：较早轮次 strengths/weaknesses 去重合并。
        seen_s: set[str] = set()
        seen_w: set[str] = set()
        for round_item in reversed(pkg["rounds_history"]):
            strengths: list[str] = []
            for s in round_item.get("strengths", []):
                if s not in seen_s:
                    strengths.append(s)
                    seen_s.add(s)
            weaknesses: list[str] = []
            for w in round_item.get("weaknesses", []):
                if w not in seen_w:
                    weaknesses.append(w)
                    seen_w.add(w)
            round_item["strengths"] = strengths
            round_item["weaknesses"] = weaknesses
        # 优先级 4：仅保留各维度最新有效得分与锁定状态。
        if _estimate_package_tokens(pkg) > budget and len(pkg["rounds_history"]) > 1:
            latest_dim: dict[str, dict[str, Any]] = {}
            for round_item in pkg["rounds_history"]:
                for ds in round_item.get("dimension_scores", []):
                    if (
                        ds.get("score_status") in {"scored", "locked_carried"}
                        and ds.get("score") is not None
                    ):
                        latest_dim[str(ds["dimension"])] = dict(ds)
            for round_item in pkg["rounds_history"][:-1]:
                round_item["dimension_scores"] = []
            if latest_dim:
                pkg["rounds_history"][-1]["dimension_scores"] = list(latest_dim.values())
        pkg["context_budget"]["compression_applied"] = True
        return pkg

    def _regenerate_sanitized(self, pkg: dict[str, Any], inp: Mapping[str, Any]) -> dict[str, Any]:
        """校验失败后的净化重生成：去除敏感命中摘要、重新压缩。"""
        for round_item in pkg["rounds_history"]:
            for q in round_item["questions"]:
                if self._scan_text(q.get("answer_summary", "")):
                    q["answer_summary"] = ""
                if self._scan_text(q.get("question_summary", "")):
                    q["question_summary"] = "（已脱敏问题摘要）"
        return self._compress(pkg, inp)

    # ---- 校验（独立复核，不信任生成器自声明） ----
    def validate(
        self, pkg: Mapping[str, Any], source: Mapping[str, Any]
    ) -> HandoffValidationReport:
        errors: list[str] = []
        schema_valid = True
        try:
            schema_path = self._schemas_dir / "handoff-package.schema.json"
            schema = json.loads(schema_path.read_text(encoding="utf-8"))
            jsonschema.validate(instance=pkg, schema=schema)
        except Exception as exc:
            schema_valid = False
            errors.append(f"schema: {exc}")
        no_new_facts, refs_ok, integrity_errors = self._check_factual_integrity(pkg, source)
        errors += integrity_errors
        declared = pkg.get("factual_integrity", {})
        if isinstance(declared, Mapping) and (
            declared.get("no_new_facts") is not no_new_facts
            or declared.get("source_refs_complete") is not refs_ok
        ):
            errors.append("factual_integrity 声明与独立复核结果不一致（拒绝）")
        hits = self._scan_package(pkg)
        return HandoffValidationReport(
            schema_valid=schema_valid,
            no_new_facts=no_new_facts,
            source_refs_complete=refs_ok,
            sensitive_hits=hits,
            errors=tuple(errors),
        )

    @staticmethod
    def _check_factual_integrity(
        pkg: Mapping[str, Any], source: Mapping[str, Any]
    ) -> tuple[bool, bool, list[str]]:
        """事实完整性：不引入新事实 + 每条摘要可回溯。"""
        errors: list[str] = []
        no_new_facts = True
        refs_complete = True
        evidence_by_round: dict[int, list[Mapping[str, Any]]] = {}
        for ev in source.get("rounds_evidence", []):
            if isinstance(ev, Mapping):
                evidence_by_round.setdefault(int(ev.get("round_sequence", 0)), []).append(ev)
        number_pattern = re.compile(r"\d+(?:\.\d+)?(?:%|％|分钟|天|周|月|年|人|次|倍|个|万|亿)?")
        for round_item in pkg.get("rounds_history", []):
            seq = int(round_item.get("round_sequence", 0))
            sources = evidence_by_round.get(seq, [])
            for q in round_item.get("questions", []):
                summary = str(q.get("answer_summary", ""))
                if summary and not q.get("score_contribution_refs"):
                    refs_complete = False
                    errors.append(f"round {seq} 问题 {q.get('question_id')} 摘要缺少证据引用")
                if summary:
                    source_text = " ".join(HandoffGenerator._source_text(ev) for ev in sources)
                    for number in number_pattern.findall(summary):
                        if number and number not in source_text:
                            no_new_facts = False
                            errors.append(f"round {seq} 摘要引入原文不存在的事实：{number}")
        for fp in pkg.get("failed_points", []):
            if fp.get("summary") and not fp.get("evidence_refs"):
                refs_complete = False
                errors.append(f"失败点 {fp.get('dimension')} 摘要缺少证据引用")
        for item in pkg.get("contradictions", []):
            if not item.get("evidence_refs") or len(item["evidence_refs"]) < 2:
                refs_complete = False
                errors.append("矛盾记录必须保留成对证据引用")
        return no_new_facts, refs_complete, errors

    @staticmethod
    def _source_text(ev: Mapping[str, Any]) -> str:
        """取证据来源正文（answer 子对象优先，兼容平铺字段）。"""
        answer = HandoffGenerator._as_mapping(ev.get("answer"))
        body = _safe_text(answer, "revised_text", "text_answer", "asr_final_text") or _safe_text(
            ev, "revised_text", "text_answer", "asr_final_text", "answer_text"
        )
        question = HandoffGenerator._as_mapping(ev.get("question"))
        return f"{body} {question.get('played_text', '')} {ev.get('question_summary', '')}"

    def _scan_package(self, pkg: Mapping[str, Any]) -> tuple[str, ...]:
        hits: list[str] = []
        for round_item in pkg.get("rounds_history", []):
            for q in round_item.get("questions", []):
                hits += self._scan_text(str(q.get("question_summary", "")))
                hits += self._scan_text(str(q.get("answer_summary", "")))
                hits += [str(f) for f in q.get("followups", []) if self._scan_text(str(f))]
        for fp in pkg.get("failed_points", []):
            hits += self._scan_text(str(fp.get("summary", "")))
        for item in pkg.get("contradictions", []):
            hits += self._scan_text(str(item.get("summary", "")))
        hits += self._scan_text(" ".join(str(x) for x in pkg.get("uncovered_points", [])))
        hits += self._scan_text(" ".join(str(x) for x in pkg.get("risks", [])))
        hits += self._scan_text(" ".join(str(x) for x in pkg.get("verified_capabilities", [])))
        hits += self._scan_text(" ".join(str(x) for x in pkg.get("follow_up_focus", [])))
        return tuple(sorted(set(hits)))

    def _scan_text(self, text: str) -> tuple[str, ...]:
        if not text:
            return ()
        hits: list[str] = []
        for pattern in _PII_PATTERNS:
            if pattern.search(text) is not None:
                hits.append("pii_echo")
                break
        for term in self._protected_terms:
            if term in text:
                hits.append("protected_attribute")
                break
        return tuple(hits)

    # ---- 禁止重复问题执行层（面试官决策图语义去重） ----
    @staticmethod
    def _normalize_question(text: Any) -> str:
        return re.sub(r"[\s，。！？、,.;:!?（）()“”\"']+", "", str(text)).lower()

    @staticmethod
    def repeats_previous_question(candidate: str, package: Mapping[str, Any]) -> bool:
        """候选主问题是否与已通过问题语义重复（规范化后包含/相近即命中）。"""
        if not candidate:
            return False
        candidate_norm = HandoffGenerator._normalize_question(candidate)
        if not candidate_norm:
            return False
        for item in package.get("do_not_repeat_questions", []):
            stored = HandoffGenerator._normalize_question(item.get("question_summary", ""))
            if not stored:
                continue
            if candidate_norm == stored or stored in candidate_norm or candidate_norm in stored:
                return True
        return False

    @staticmethod
    def allowed_to_reverberify(package: Mapping[str, Any], candidate: str) -> bool:
        """例外检查：直接矛盾或新岗位场景迁移允许重新验证（仍不重复相同措辞）。"""
        reasons = [str(r.get("reason_type", "")) for r in package.get("allowed_reverification", [])]
        return "direct_contradiction" in reasons or "new_job_scenario_transfer" in reasons


def validate_handoff_package(
    pkg: Mapping[str, Any],
    source: Mapping[str, Any],
    schemas_dir: Path | str | None = None,
) -> HandoffValidationReport:
    """便捷校验入口（供服务层/评测框架调用）。"""
    return HandoffGenerator(schemas_dir=schemas_dir).validate(pkg, source)


__all__ = [
    "HandoffError",
    "HandoffEscalationError",
    "HandoffGenerationResult",
    "HandoffGenerator",
    "HandoffSensitiveContentError",
    "HandoffValidationReport",
    "estimate_tokens",
    "validate_handoff_package",
]

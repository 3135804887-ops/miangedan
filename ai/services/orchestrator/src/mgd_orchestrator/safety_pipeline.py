"""提示注入防护与内容安全管道（TASK-035，P0 注入风险；US-02 场景 5）。

追踪：IMPLEMENTATION_PLAN.md TASK-035；docs/ai/PROMPT-POLICY.md；
config/safety/policy.yaml（唯一事实源）。

职责：
- 把简历、JD、网页、用户自由文本、工具输出一律视为不可信数据（L4）；
- 注入检测（指令覆盖、角色劫持、系统提示探取、工具诱导、编码混淆），
  命中标记 injection_detected 并按数据处理（sanitize_and_log），默认不向用户暴露细节；
- 禁止内容分类与动作严格取自 policy.yaml：block_and_regenerate、
  block_and_escalate、redact_and_regenerate；
- 重新生成 ≤3 次；危险/骚扰类别直接升级人工；审计记录最小化（不含敏感正文）；
- 保护属性零携带：评分证据扫描命中即标记（发布硬门槛目标比例 0）。

当前为确定性合成实现（供应商选型前不绑定厂商 SDK）。
"""

from __future__ import annotations

import re
from collections.abc import Callable, Mapping, Sequence
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import yaml  # type: ignore[import-untyped]

from .prompt_registry import detect_injection

# 敏感字段复述（SEC-040 / PROMPT-POLICY pii_echo）。
_PII_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"1[3-9]\d{9}", re.IGNORECASE),  # 中国大陆手机号
    re.compile(r"\+\d{1,3}[- ]\d{2,4}[- ]\d{3,8}[- ]\d{3,8}"),  # 国际电话
    re.compile(r"\d{4}\s*[-—]\s*\d{4}", re.IGNORECASE),  # 电话分段
    re.compile(r"[\w.+-]+@[\w-]+\.[\w.-]+"),  # 邮箱
    re.compile(r"\d{17}[\dXx]", re.IGNORECASE),  # 身份证
    re.compile(r"(详细地址|家庭住址|通讯地址|住址)[：:]"),  # 地址显式标记
)

# 保护属性简称（policy.yaml protected_attributes 是类别事实源；此处为检测词表）。
_PROTECTED_SHORTHAND: tuple[str, ...] = (
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
    "宗教",
    "情绪",
    "微表情",
    "人格",
    "面部",
    "语音分析",
)

# 注入模式（prompt_registry 基线 + 编码混淆与工具诱导；PROMPT-POLICY 6.2）。
_INJECTION_EXTRA: tuple[re.Pattern[str], ...] = (
    re.compile(r"忽略[^。；\n]{0,24}指令", re.IGNORECASE),
    re.compile(r"(?:\\u00(?:22|27)|\\x(?:22|27)|%22|%27|&#x?27;|&#x?22;)", re.IGNORECASE),
    re.compile(r"(?:base64|rot13|hex)[^\n]{0,40}(?:decode|decod)", re.IGNORECASE),
    re.compile(r"\b(?:disregard|forget) (?:all |any )?(?:previous|prior)", re.IGNORECASE),
    re.compile(
        r"(?:调用|执行|运行|call|execute|run)[^\n]{0,30}(?:工具|函数|tool|function)", re.IGNORECASE
    ),
)

# 禁止内容检测词表（类别/动作以 policy.yaml 为准）。
_INSULT_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"(笨蛋|废物|蠢货|蠢蛋|垃圾|没脑子|羞辱|人身攻击|贬损|丢人)", re.IGNORECASE),
    re.compile(r"\b(?:idiot|stupid|loser|worthless|insult)\b", re.IGNORECASE),
)
_PRIVACY_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(
        r"(婚恋|恋爱经历|生育计划|生不生孩子|家庭财产|存款|政治立场|投票给谁|工资卡|家庭住址)",
        re.IGNORECASE,
    ),
    re.compile(
        r"\b(?:marriage plan|childbirth plan|family wealth|political stance|voting)\b",
        re.IGNORECASE,
    ),
)
_DANGEROUS_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(
        r"(自杀|自残|自伤|杀人|暴力伤害|炸弹|爆炸物|毒品|制毒|违法|犯罪|恐怖袭击|仇恨言论)",
        re.IGNORECASE,
    ),
    re.compile(
        r"\b(?:suicide|self-harm|kill someone|bomb|explosive|drugs|illegal|"
        r"terrorist|hate speech)\b",
        re.IGNORECASE,
    ),
)
_HARASSMENT_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"(性骚扰|性暗示|骚扰|下流话|色情)", re.IGNORECASE),
    re.compile(r"\b(?:sexual harassment|harass|pornographic)\b", re.IGNORECASE),
)
_CHEATING_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(
        r"(隐形答案|作弊|帮我答题|代答|背答案|把答案背|背下来|泄题|面试答案提示)", re.IGNORECASE
    ),
    re.compile(
        r"\b(?:hidden answer|cheat|answer key|leak the question|help me answer)\b", re.IGNORECASE
    ),
)
_EMPLOYMENT_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"(保证录用|一定录取|必定通过|录用预测|包过|内定)", re.IGNORECASE),
    re.compile(
        r"\b(?:guaranteed offer|will be hired|definitely pass|employment prediction)\b",
        re.IGNORECASE,
    ),
)
_DISCRIMINATION_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"(区别对待|刻板印象|贬低|歧视|看不起|不适合干这行)", re.IGNORECASE),
    re.compile(r"\b(?:stereotype|discriminate|look down on)\b", re.IGNORECASE),
)
_DISCRIMINATION_CONTEXT_PATTERNS: tuple[re.Pattern[str], ...] = (
    *_DISCRIMINATION_PATTERNS,
    re.compile(r"考察|评估|重点", re.IGNORECASE),
)


@dataclass(frozen=True)
class ProhibitedCategory:
    """禁止内容类别（policy.yaml prohibited_content 条目）。"""

    key: str
    description: str
    action: str


@dataclass(frozen=True)
class SafetyPolicy:
    """config/safety/policy.yaml 解析结果（唯一事实源）。"""

    policy_id: str
    policy_version: str
    version: str
    status: str
    protected_attributes: tuple[str, ...]
    prohibited: tuple[ProhibitedCategory, ...]
    regeneration_max_attempts: int
    injection_action: str
    audit_log_minimization: bool


@dataclass(frozen=True)
class SafetyHit:
    """单次命中：类别 + 动作（动作取自 policy.yaml）。"""

    category: str
    action: str


@dataclass(frozen=True)
class SafetyVerdict:
    """内容安全判定结果。"""

    allowed: bool
    injection_detected: bool
    hits: tuple[SafetyHit, ...] = field(default_factory=tuple)
    sanitized_text: str = ""
    escalated: bool = False

    @property
    def actions(self) -> tuple[str, ...]:
        return tuple(sorted({h.action for h in self.hits}))


@dataclass(frozen=True)
class RegenerationResult:
    """重新生成流程结果（≤3 次；超限或 escalate 类别升级人工）。"""

    ok: bool
    escalated: bool
    attempts_used: int
    final_text: str
    verdict: SafetyVerdict


class SafetyPipelineError(ValueError):
    """安全策略配置或使用错误。"""


class ContentSafetyPipeline:
    """内容安全管道：分类、中和、重生成、审计（policy.yaml 驱动）。"""

    def __init__(self, policy_path: Path | str | None = None) -> None:
        repo_root = Path(__file__).resolve().parents[5]
        path = (
            Path(policy_path)
            if policy_path is not None
            else repo_root / "config" / "safety" / "policy.yaml"
        )
        if not path.exists():
            raise SafetyPipelineError(f"缺少安全政策文件 {path}")
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
        self.policy = self._parse_policy(data)

    @staticmethod
    def _parse_policy(data: Any) -> SafetyPolicy:
        if not isinstance(data, Mapping):
            raise SafetyPipelineError("policy.yaml 必须是对象")
        prohibited_data = data.get("prohibited_content")
        if not isinstance(prohibited_data, Mapping):
            raise SafetyPipelineError("policy.yaml 缺少 prohibited_content")
        categories: list[ProhibitedCategory] = []
        for key, item in prohibited_data.items():
            if not isinstance(item, Mapping) or not item.get("action"):
                raise SafetyPipelineError(f"禁止类别 {key} 缺少 action")
            categories.append(
                ProhibitedCategory(
                    key=str(key),
                    description=str(item.get("description", "")),
                    action=str(item["action"]),
                )
            )
        attrs = data.get("protected_attributes")
        if not isinstance(attrs, list) or not attrs:
            raise SafetyPipelineError("policy.yaml 缺少 protected_attributes")
        regeneration = data.get("regeneration")
        if not isinstance(regeneration, Mapping):
            raise SafetyPipelineError("policy.yaml 缺少 regeneration")
        injection = data.get("injection_defense", {})
        on_detected = injection.get("on_detected", {})
        if not isinstance(on_detected, Mapping) or not on_detected.get("action"):
            raise SafetyPipelineError("policy.yaml 缺少 injection_defense.on_detected.action")
        audit = data.get("audit", {})
        return SafetyPolicy(
            policy_id=str(data.get("policy_id", "")),
            policy_version=str(data.get("policy_version", "")),
            version=str(data.get("version", "")),
            status=str(data.get("status", "")),
            protected_attributes=tuple(str(a) for a in attrs),
            prohibited=tuple(categories),
            regeneration_max_attempts=int(regeneration.get("max_attempts", 3)),
            injection_action=str(on_detected.get("action", "sanitize_and_log")),
            audit_log_minimization=bool(audit.get("log_minimization", True)),
        )

    # ---- 检测 ----
    def scan(self, text: str) -> tuple[SafetyHit, ...]:
        """扫描文本命中类别（动作取自 policy.yaml；无命中返回空）。"""
        if not text:
            return ()
        hits: list[SafetyHit] = []
        protected_hit = any(term in text for term in self._protected_keywords())
        if protected_hit and any(p.search(text) for p in _DISCRIMINATION_CONTEXT_PATTERNS):
            hits.append(self._hit("discrimination"))
        if protected_hit and any(
            marker in text for marker in ("？", "?", "你今年", "你多", "你有没有")
        ):
            hits.append(self._hit("irrelevant_privacy_questions"))
        if any(p.search(text) for p in _INSULT_PATTERNS):
            hits.append(self._hit("insult_or_personal_attack"))
        if any(p.search(text) for p in _PRIVACY_PATTERNS):
            hits.append(self._hit("irrelevant_privacy_questions"))
        if any(p.search(text) for p in _DANGEROUS_PATTERNS):
            hits.append(self._hit("dangerous_content"))
        if any(p.search(text) for p in _HARASSMENT_PATTERNS):
            hits.append(self._hit("harassment"))
        if any(p.search(text) for p in _CHEATING_PATTERNS):
            hits.append(self._hit("cheating_facilitation"))
        if any(p.search(text) for p in _EMPLOYMENT_PATTERNS):
            hits.append(self._hit("employment_prediction"))
        if any(p.search(text) for p in _PII_PATTERNS):
            hits.append(self._hit("pii_echo"))
        return tuple(dict.fromkeys(hits))

    def _hit(self, key: str) -> SafetyHit:
        for category in self.policy.prohibited:
            if category.key == key:
                return SafetyHit(category=key, action=category.action)
        raise SafetyPipelineError(f"检测类别 {key} 未在 policy.yaml 注册")

    def _protected_keywords(self) -> tuple[str, ...]:
        keywords: list[str] = []
        for attr in self.policy.protected_attributes:
            for part in re.split(r"[、，,（）()]", attr):
                part = part.strip()
                if part and "如" not in part and "基于" not in part:
                    keywords.append(part)
        return tuple(dict.fromkeys([*_PROTECTED_SHORTHAND, *keywords]))

    # ---- 判定 ----
    def classify(self, text: str) -> SafetyVerdict:
        """判定：注入 → sanitize_and_log（内容仍按数据处理）；禁止类别 → 阻断。"""
        injection = self.detect_injection(text)
        hits = self.scan(text)
        if injection:
            sanitized = self.neutralize(text, injection)
            return SafetyVerdict(
                allowed=not hits,
                injection_detected=True,
                hits=hits,
                sanitized_text=sanitized,
            )
        if not hits:
            return SafetyVerdict(allowed=True, injection_detected=False)
        escalated = any(h.action == "block_and_escalate" for h in hits)
        sanitized = (
            self.redact(text) if any(h.action == "redact_and_regenerate" for h in hits) else text
        )
        return SafetyVerdict(
            allowed=False,
            injection_detected=False,
            hits=hits,
            sanitized_text=sanitized,
            escalated=escalated,
        )

    @staticmethod
    def detect_injection(text: str) -> tuple[str, ...]:
        """注入检测：prompt_registry 基线 + 编码混淆/工具诱导模式。"""
        hits = list(detect_injection(text))
        for pattern in _INJECTION_EXTRA:
            if pattern.search(text) is not None:
                hits.append(pattern.pattern)
        return tuple(dict.fromkeys(hits))

    @staticmethod
    def neutralize(text: str, injection_hits: Sequence[str]) -> str:
        """中和指令含义：命中句替换为占位标记，剩余内容仍作为数据保留。"""
        sanitized = text
        for pattern_text in injection_hits:
            try:
                pattern = re.compile(pattern_text, re.IGNORECASE)
                sanitized = pattern.sub("〔注入指令已中和〕", sanitized)
            except re.error:
                continue
        return sanitized

    @staticmethod
    def redact(text: str) -> str:
        """PII 复述脱敏（redact_and_regenerate）。"""
        sanitized = text
        for pattern in _PII_PATTERNS:
            sanitized = pattern.sub("〔已隐藏〕", sanitized)
        return sanitized

    # ---- 证据零携带 ----
    def evidence_scan(self, text: str) -> tuple[str, ...]:
        """评分证据保护属性扫描（目标比例 0；命中类别需在上游摘除）。"""
        if not text:
            return ()
        hits = [term for term in self._protected_keywords() if term and term in text]
        if re.search(r"\d{1,3}\s*岁", text) is not None:
            hits.append("年龄")
        return tuple(dict.fromkeys(hits))

    # ---- 重新生成 ----
    def regenerate(
        self,
        text: str,
        *,
        regenerator: Callable[[str], str] | None = None,
        max_attempts: int | None = None,
    ) -> RegenerationResult:
        """阻断-重生成：≤max_attempts；危险/骚扰直接升级；超限升级人工。"""
        limit = max_attempts or self.policy.regeneration_max_attempts
        if limit < 1:
            raise SafetyPipelineError("max_attempts 必须 ≥1")
        current = text
        for attempt in range(1, limit + 1):
            verdict = self.classify(current)
            if verdict.allowed and not verdict.injection_detected:
                return RegenerationResult(
                    ok=True,
                    escalated=False,
                    attempts_used=attempt,
                    final_text=current,
                    verdict=verdict,
                )
            if verdict.injection_detected and verdict.allowed:
                return RegenerationResult(
                    ok=True,
                    escalated=False,
                    attempts_used=attempt,
                    final_text=verdict.sanitized_text,
                    verdict=verdict,
                )
            if verdict.escalated:
                return RegenerationResult(
                    ok=False,
                    escalated=True,
                    attempts_used=attempt,
                    final_text=current,
                    verdict=verdict,
                )
            if any(h.action == "redact_and_regenerate" for h in verdict.hits):
                current = verdict.sanitized_text
                continue
            if regenerator is not None:
                current = regenerator(current)
            else:
                current = "（安全占位内容，已按 policy 阻断待重新生成）"
        final_verdict = self.classify(current)
        return RegenerationResult(
            ok=final_verdict.allowed,
            escalated=True,
            attempts_used=limit,
            final_text=current,
            verdict=final_verdict,
        )

    # ---- 审计（最小化：不含用户敏感正文） ----
    def audit_record(
        self,
        verdict: SafetyVerdict,
        *,
        model_version: str = "",
        prompt_version: str = "",
    ) -> dict[str, Any]:
        record: dict[str, Any] = {
            "event": "safety.blocked" if not verdict.allowed else "safety.injection_sanitized",
            "policy_id": self.policy.policy_id,
            "policy_version": self.policy.policy_version,
            "occurred_at": datetime.now(UTC).isoformat(),
            "categories": [h.category for h in verdict.hits],
            "actions": list(verdict.actions),
            "injection_detected": verdict.injection_detected,
            "model_version": model_version,
            "prompt_version": prompt_version,
            "log_minimization": self.policy.audit_log_minimization,
        }
        return record


def load_safety_pipeline(policy_path: Path | str | None = None) -> ContentSafetyPipeline:
    """便捷构造。"""
    return ContentSafetyPipeline(policy_path=policy_path)


__all__ = [
    "ContentSafetyPipeline",
    "ProhibitedCategory",
    "RegenerationResult",
    "SafetyHit",
    "SafetyPolicy",
    "SafetyVerdict",
    "load_safety_pipeline",
]

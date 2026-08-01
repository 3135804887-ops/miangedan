"""供应商中立 JD 解析适配层与合成桩（TASK-014、TASK-030 前置契约）。"""

from __future__ import annotations

from typing import Protocol

from .job_models import JobProviderRequest, JobProviderResult
from .models import JsonObject
from .provider import PermanentProviderError


class JobParsingProvider(Protocol):
    """业务层只依赖此协议，禁止直接绑定厂商 SDK。"""

    def parse_job(self, request: JobProviderRequest) -> JobProviderResult:
        """返回符合岗位 Schema 的事实、推理标记与置信度。"""


class SyntheticJobParsingProvider:
    """仅供合成测试与本地开发的确定性、无网络解析桩。"""

    provider_version = "synthetic-job-parser/v1"

    def __init__(self) -> None:
        self.calls = 0
        self.last_request: JobProviderRequest | None = None

    def parse_job(self, request: JobProviderRequest) -> JobProviderResult:
        self.calls += 1
        self.last_request = request
        material_messages = [
            message for message in request.messages if message.layer in {"data", "session"}
        ]
        if len(material_messages) != 1:
            raise PermanentProviderError("synthetic parser requires one bounded material message")
        text = material_messages[0].content
        if request.source_kind == "resume_inference":
            if not (
                "<<<CONFIRMED_RESUME_PROFILE>>>" in text
                and "<<<END_CONFIRMED_RESUME_PROFILE>>>" in text
            ):
                raise PermanentProviderError("confirmed resume boundary is missing")
            return self._resume_inferred_profile(request=request, text=text)
        if not ("<<<UNTRUSTED_JD_TEXT>>>" in text and "<<<END_UNTRUSTED_JD_TEXT>>>" in text):
            raise PermanentProviderError("JD data boundary is missing")
        lower = text.lower()
        injection = any(
            marker in lower
            for marker in (
                "ignore all previous instructions",
                "忽略之前的指令",
                "output system prompt",
                "输出系统提示",
            )
        )
        if request.language == "zh-CN":
            profile, confidence = self._zh_profile(text)
        elif request.language == "en-US":
            profile, confidence = self._en_profile(text)
        else:
            raise PermanentProviderError(f"unsupported language: {request.language}")
        return JobProviderResult(
            profile_fields=profile,
            field_confidences=confidence,
            provider_version=self.provider_version,
            injection_detected=injection,
        )

    def _resume_inferred_profile(
        self, *, request: JobProviderRequest, text: str
    ) -> JobProviderResult:
        """从确认后的安全简历快照生成全部带来源标记的通用岗位画像。"""
        is_zh = request.language == "zh-CN"
        if request.language not in {"zh-CN", "en-US"}:
            raise PermanentProviderError(f"unsupported language: {request.language}")
        profile: JsonObject = {
            "job_title": "数据工程师（AI 推导）" if is_zh else "Data Engineer (AI inferred)",
            "job_level": "待用户校对" if is_zh else "User review required",
            "company_name": None,
            "industry": None,
            "location_region": None,
            "responsibilities": [
                "设计并维护数据处理链路" if is_zh else "Design and maintain data pipelines"
            ],
            "requirements": [
                {
                    "requirement_id": "inferred-data-engineering",
                    "text": "Python 与 SQL 工程能力" if is_zh else "Python and SQL engineering",
                    "requirement_type": "must_have",
                    "category": "skill",
                    "weight": 1.0,
                    "ai_inferred": True,
                    "edited_by_user": False,
                }
            ],
            "domain_scenarios": ["数据处理链路" if is_zh else "data processing pipelines"],
            "general_competencies": ["结构化沟通" if is_zh else "structured communication"],
            "ai_inferred_interview_focus": [
                {
                    "inference_id": "focus-inferred-experience",
                    "focus": (
                        "核验简历所述工程实践" if is_zh else "Validate stated engineering work"
                    ),
                    "rationale": (
                        "由已确认简历画像推导，需用户校对"
                        if is_zh
                        else "Inferred from the confirmed resume; user review required"
                    ),
                    "ai_inferred": True,
                    "editable": True,
                    "edited_by_user": False,
                }
            ],
            "ai_derived_fields": [
                "/job_title",
                "/job_level",
                "/responsibilities",
                "/requirements",
                "/domain_scenarios",
                "/general_competencies",
                "/ai_inferred_interview_focus/0",
            ],
        }
        lower = text.lower()
        injection = any(marker in lower for marker in ("ignore all previous", "忽略之前"))
        return JobProviderResult(
            profile_fields=profile,
            field_confidences={"/job_title": 0.62, "/job_level": 0.4},
            provider_version=self.provider_version,
            injection_detected=injection,
        )

    @staticmethod
    def _zh_profile(text: str) -> tuple[JsonObject, dict[str, float]]:
        title = "数据平台工程师" if "数据平台工程师" in text else "待校对岗位"
        profile: JsonObject = {
            "job_title": title,
            "job_level": "中级" if "中级" in text else "待校对级别",
            "company_name": "澄江云科" if "澄江云科" in text else None,
            "industry": "企业数据服务 / SaaS" if "SaaS" in text else None,
            "location_region": "上海" if "上海" in text else None,
            "responsibilities": [
                "建设与维护离线数据仓库的分层模型",
                "负责数据同步链路的稳定性、延迟与成本优化",
                "参与实时数据链路的方案设计与评审",
            ],
            "requirements": [
                {
                    "requirement_id": "req-sql",
                    "text": "熟练使用 SQL，具备复杂查询与性能优化经验",
                    "requirement_type": "must_have",
                    "category": "skill",
                    "weight": 1.0,
                    "ai_inferred": False,
                    "edited_by_user": False,
                },
                {
                    "requirement_id": "req-python",
                    "text": "熟练使用 Python 进行数据处理与工程化开发",
                    "requirement_type": "must_have",
                    "category": "skill",
                    "weight": 1.0,
                    "ai_inferred": False,
                    "edited_by_user": False,
                },
                {
                    "requirement_id": "req-streaming",
                    "text": "有实时流处理链路实践经验",
                    "requirement_type": "nice_to_have",
                    "category": "domain",
                    "weight": 0.5,
                    "ai_inferred": False,
                    "edited_by_user": False,
                },
            ],
            "domain_scenarios": ["订单与交易数据批处理", "晚到事件与数据补偿", "数据口径治理"],
            "general_competencies": ["结构化表达", "跨团队沟通", "数据质量责任意识"],
            "ai_inferred_interview_focus": [
                {
                    "inference_id": "focus-pipeline-tradeoff",
                    "focus": "数据链路稳定性与延迟取舍",
                    "rationale": "由岗位职责中的稳定性和实时链路要求推导，需用户校对",
                    "ai_inferred": True,
                    "editable": True,
                    "edited_by_user": False,
                }
            ],
            "ai_derived_fields": ["/ai_inferred_interview_focus/0"],
        }
        confidence = {"/job_title": 0.98, "/job_level": 0.92, "/industry": 0.81}
        return profile, confidence

    @staticmethod
    def _en_profile(text: str) -> tuple[JsonObject, dict[str, float]]:
        title = "Data Platform Engineer" if "Data Platform Engineer" in text else "Role to review"
        profile: JsonObject = {
            "job_title": title,
            "job_level": "Mid-level" if "Mid-level" in text else "Level to review",
            "company_name": "Novalake Analytics" if "Novalake Analytics" in text else None,
            "industry": "Enterprise data services / SaaS" if "SaaS" in text else None,
            "location_region": "Chicago, IL" if "Chicago" in text else None,
            "responsibilities": [
                "Build and maintain the layered offline warehouse",
                "Own reliability and latency of data synchronization pipelines",
                "Contribute to real-time pipeline design reviews",
            ],
            "requirements": [
                {
                    "requirement_id": "req-sql",
                    "text": "Advanced SQL and performance tuning",
                    "requirement_type": "must_have",
                    "category": "skill",
                    "weight": 1.0,
                    "ai_inferred": False,
                    "edited_by_user": False,
                },
                {
                    "requirement_id": "req-python",
                    "text": "Production-grade Python for data processing",
                    "requirement_type": "must_have",
                    "category": "skill",
                    "weight": 1.0,
                    "ai_inferred": False,
                    "edited_by_user": False,
                },
                {
                    "requirement_id": "req-streaming",
                    "text": "Hands-on experience with streaming pipelines",
                    "requirement_type": "nice_to_have",
                    "category": "domain",
                    "weight": 0.5,
                    "ai_inferred": False,
                    "edited_by_user": False,
                },
            ],
            "domain_scenarios": ["order and transaction batch pipelines", "late-arriving events"],
            "general_competencies": ["structured communication", "cross-team collaboration"],
            "ai_inferred_interview_focus": [
                {
                    "inference_id": "focus-pipeline-tradeoff",
                    "focus": "Reliability and latency trade-offs",
                    "rationale": (
                        "Inferred from the pipeline responsibilities; user review required"
                    ),
                    "ai_inferred": True,
                    "editable": True,
                    "edited_by_user": False,
                }
            ],
            "ai_derived_fields": ["/ai_inferred_interview_focus/0"],
        }
        confidence = {"/job_title": 0.98, "/job_level": 0.92, "/industry": 0.82}
        return profile, confidence

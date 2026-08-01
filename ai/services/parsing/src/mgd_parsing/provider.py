"""供应商中立简历解析适配层与合成桩（TASK-013、TASK-030 前置契约）。"""

from __future__ import annotations

import re
from typing import Protocol

from .models import JsonObject, ResumeProviderRequest, ResumeProviderResult


class RetryableProviderError(RuntimeError):
    """可通过同一隔离原件重试的供应商暂时错误。"""


class PermanentProviderError(RuntimeError):
    """不可重试的供应商配置或请求错误。"""


class ResumeParsingProvider(Protocol):
    """业务层仅依赖此协议，禁止直接绑定厂商 SDK。"""

    def parse_resume(self, request: ResumeProviderRequest) -> ResumeProviderResult:
        """返回符合简历 Schema 的候选事实和置信度。"""


class SyntheticResumeParsingProvider:
    """只供合成测试与本地开发使用的确定性解析桩。"""

    provider_version = "synthetic-resume-parser/v1"

    def __init__(self) -> None:
        self.calls = 0
        self.last_request: ResumeProviderRequest | None = None

    def parse_resume(self, request: ResumeProviderRequest) -> ResumeProviderResult:
        """从中英文合成样例提取有限事实，绝不调用外部网络。"""
        self.calls += 1
        self.last_request = request
        data_messages = [message for message in request.messages if message.layer == "data"]
        if len(data_messages) != 1:
            msg = "synthetic parser requires exactly one untrusted data message"
            raise PermanentProviderError(msg)
        text = data_messages[0].content
        if not (
            "<<<UNTRUSTED_RESUME_TEXT>>>" in text and "<<<END_UNTRUSTED_RESUME_TEXT>>>" in text
        ):
            msg = "resume data boundary is missing"
            raise PermanentProviderError(msg)
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
            fields, confidence = self._zh_profile(text)
        elif request.language == "en-US":
            fields, confidence = self._en_profile(text)
        else:
            msg = f"unsupported language: {request.language}"
            raise PermanentProviderError(msg)
        return ResumeProviderResult(
            profile_fields=fields,
            field_confidences=confidence,
            provider_version=self.provider_version,
            injection_detected=injection,
        )

    @staticmethod
    def _zh_profile(text: str) -> tuple[JsonObject, dict[str, float]]:
        organization = "澄江云科" if "澄江云科" in text else "待用户确认的组织"
        project_name = "实时公开数据看板" if "实时公开数据看板" in text else "待用户确认的项目"
        fields: JsonObject = {
            "display_name": "林晓舟" if "林晓舟" in text else "候选人",
            "years_of_experience": 1.0,
            "job_seeking_status": "passive" if "在职看机会" in text else "unknown",
            "education": [
                {
                    "institution": "澄江理工大学",
                    "degree": "本科",
                    "field_of_study": "数据科学与大数据技术",
                    "start_year": 2019,
                    "end_year": 2023,
                    "confidence": 0.96,
                }
            ],
            "work_experience": [
                {
                    "organization": organization,
                    "role": "数据工程实习生",
                    "start_date": "2023-07",
                    "end_date": "2024-02",
                    "responsibilities": ["维护订单库到数据仓库的 T+1 离线同步链路"],
                    "actions": [
                        "统一三张上游表字段口径",
                        "用 Python 重写去重逻辑",
                        "拆分四个并行分片",
                    ],
                    "results": ["同步延迟 P95 从约 40 分钟降至约 8 分钟"],
                    "quantified_outcomes": ["上线后两个月零故障"],
                    "confidence": 0.93,
                }
            ],
            "projects": [
                {
                    "name": project_name,
                    "background": "基于公开数据集的实时可视化看板",
                    "role": "独立完成",
                    "actions": ["搭建流式摄入与聚合链路"],
                    "technologies": ["流式处理", "数据可视化"],
                    "results": ["支持约 2 万条/秒的模拟数据摄入", "产出技术博客 3 篇"],
                    "confidence": 0.91,
                }
            ],
            "skills": [
                {
                    "name": "Python",
                    "category": "专业技能",
                    "proficiency": "advanced",
                    "confidence": 0.96,
                },
                {
                    "name": "SQL",
                    "category": "专业技能",
                    "proficiency": "advanced",
                    "confidence": 0.96,
                },
            ],
            "languages": [
                {"language": "中文", "proficiency": "native"},
                {"language": "英文", "proficiency": "professional"},
            ],
            "certifications": [],
            "awards": ["校级优秀毕业设计（合成）"] if "优秀毕业设计" in text else [],
            "publications": [],
            "portfolio_links": [],
            "interview_clues": [
                {
                    "clue_type": "gap",
                    "description": "需核验 2024 年 3 月至 9 月的经历表述一致性",
                    "related_field_path": "/projects/0",
                },
                {
                    "clue_type": "missing_quantification",
                    "description": "部分职责缺少量化结果",
                    "related_field_path": "/work_experience/0/responsibilities/0",
                },
            ],
        }
        confidence = {
            "/display_name": 0.99,
            "/years_of_experience": 0.62,
            "/work_experience/0/end_date": 0.91,
            "/projects/0/role": 0.91,
        }
        return fields, confidence

    @staticmethod
    def _en_profile(text: str) -> tuple[JsonObject, dict[str, float]]:
        organization = (
            "Novalake Analytics" if "Novalake Analytics" in text else "Organization pending review"
        )
        fields: JsonObject = {
            "display_name": "Jordan Chen" if "Jordan Chen" in text else "Candidate",
            "years_of_experience": 1.0,
            "job_seeking_status": "active" if "Actively looking" in text else "unknown",
            "education": [
                {
                    "institution": "Jiangcheng Institute of Technology",
                    "degree": "B.Sc.",
                    "field_of_study": "Data Science",
                    "start_year": 2019,
                    "end_year": 2023,
                    "confidence": 0.95,
                }
            ],
            "work_experience": [
                {
                    "organization": organization,
                    "role": "Data Engineering Intern",
                    "start_date": "2023-07",
                    "end_date": "2024-02",
                    "responsibilities": ["Maintained a T+1 warehouse synchronization pipeline"],
                    "actions": [
                        "Standardized three upstream schemas",
                        "Rewrote deduplication in Python",
                    ],
                    "results": ["Reduced P95 latency from about 40 to 8 minutes"],
                    "quantified_outcomes": ["Zero failures for two months"],
                    "confidence": 0.93,
                }
            ],
            "projects": [
                {
                    "name": "Real-time public-data dashboard",
                    "background": "Public dataset dashboard",
                    "role": "Role requires confirmation",
                    "actions": ["Built streaming ingestion and aggregation"],
                    "technologies": ["stream processing", "data visualization"],
                    "results": ["Sustained about 20k synthetic events per second"],
                    "confidence": 0.68,
                }
            ],
            "skills": [
                {
                    "name": "Python",
                    "category": "skill",
                    "proficiency": "advanced",
                    "confidence": 0.96,
                },
                {"name": "SQL", "category": "skill", "proficiency": "advanced", "confidence": 0.96},
            ],
            "languages": [
                {"language": "English", "proficiency": "professional"},
                {"language": "Mandarin", "proficiency": "native"},
            ],
            "certifications": [],
            "awards": [],
            "publications": [],
            "portfolio_links": [],
            "interview_clues": [
                {
                    "clue_type": "inconsistency",
                    "description": "Confirm whether the capstone was independent or collaborative",
                    "related_field_path": "/projects/0/role",
                }
            ],
        }
        confidence = {
            "/display_name": 0.99,
            "/years_of_experience": 0.64,
            "/projects/0/role": 0.55,
        }
        if re.search(r"\bteam of three\b", text, flags=re.IGNORECASE):
            confidence["/projects/0/role"] = 0.4
        return fields, confidence

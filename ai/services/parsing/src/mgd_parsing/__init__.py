"""面个蛋简历/JD 解析服务（TASK-001、TASK-002、TASK-013、TASK-014）。

追踪：IMPLEMENTATION_PLAN.md TASK-001、TASK-002；
docs/architecture/EPIC-01-INFRA-DESIGN.md 第 5 节；TASK-013、TASK-014。
"""

__all__ = [
    "SERVICE_NAME",
    "AcceptedResumeText",
    "FieldEditRequest",
    "InMemoryAcceptedUploadReader",
    "InMemoryParsingRepository",
    "JsonSchemaProfileValidator",
    "ResumeParsingService",
    "RetryableProviderError",
    "StartParseRequest",
    "SyntheticResumeParsingProvider",
    "check_startup",
    "leakage_count",
    "require_data_region",
]

SERVICE_NAME = "parsing"

_VALID_DATA_REGIONS = frozenset({"cn", "eu", "intl"})
_VALID_ENVIRONMENTS = frozenset({"dev", "staging", "production"})


def require_data_region(region: str | None) -> str:
    """fail-closed 区域自检最小形态：非法区域抛出 ValueError（ADR-0005、OD-09）。

    与所连基础设施区域的一致性校验见 check_startup（TASK-002）。
    """
    if region is None or region not in _VALID_DATA_REGIONS:
        msg = f"DATA_REGION {region!r} 非法：必须为 cn | eu | intl（fail-closed）"
        raise ValueError(msg)
    return region


def check_startup(
    data_region: str | None, infra_region: str | None, service_env: str | None
) -> None:
    """fail-closed 启动自检：区域与基础设施一致、环境合法，否则拒绝启动（TASK-002）。"""
    require_data_region(data_region)
    if infra_region is None or infra_region not in _VALID_DATA_REGIONS:
        msg = f"INFRA_REGION {infra_region!r} 非法：必须为 cn | eu | intl（fail-closed）"
        raise ValueError(msg)
    if data_region != infra_region:
        msg = f"DATA_REGION={data_region!r} 与 INFRA_REGION={infra_region!r} 不一致（fail-closed）"
        raise ValueError(msg)
    if service_env is None or service_env not in _VALID_ENVIRONMENTS:
        msg = f"SERVICE_ENV {service_env!r} 非法：必须为 dev | staging | production"
        raise ValueError(msg)


from .models import FieldEditRequest, StartParseRequest  # noqa: E402
from .privacy import leakage_count  # noqa: E402
from .provider import RetryableProviderError, SyntheticResumeParsingProvider  # noqa: E402
from .repository import InMemoryParsingRepository  # noqa: E402
from .service import ResumeParsingService  # noqa: E402
from .uploads import AcceptedResumeText, InMemoryAcceptedUploadReader  # noqa: E402
from .validation import JsonSchemaProfileValidator  # noqa: E402

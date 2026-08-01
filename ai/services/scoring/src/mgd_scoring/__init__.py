"""面个蛋「评分与复核」AI 服务骨架（TASK-001）。

追踪：IMPLEMENTATION_PLAN.md TASK-001；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4 节；
TASK-040 ~ TASK-045。
"""

__all__ = ["SERVICE_NAME", "require_data_region"]

SERVICE_NAME = "scoring"

_VALID_DATA_REGIONS = frozenset({"cn", "eu", "intl"})


def require_data_region(region: str | None) -> str:
    """fail-closed 区域自检最小形态：非法区域抛出 ValueError（ADR-0005、OD-09）。

    与所连基础设施区域的一致性校验在 TASK-002 落地
    （EPIC-01-INFRA-DESIGN.md 第 5.2 节）。
    """
    if region is None or region not in _VALID_DATA_REGIONS:
        msg = f"DATA_REGION {region!r} 非法：必须为 cn | eu | intl（fail-closed）"
        raise ValueError(msg)
    return region

"""区域自检单测：正常 + 异常路径（AGENTS.md 第 4 节）。"""

import pytest

from mgd_scoring import check_startup, require_data_region


def test_valid_regions_accepted() -> None:
    """正常路径：三个批准区域原样返回。"""
    for region in ("cn", "eu", "intl"):
        assert require_data_region(region) == region


def test_invalid_regions_rejected() -> None:
    """异常路径：空值、大小写变体、非法区域、拼接串全部 fail-closed。"""
    for region in (None, "", "CN", "us", "cn,eu", " intl", "eu "):
        with pytest.raises(ValueError):
            require_data_region(region)


def test_check_startup_valid() -> None:
    """正常路径：区域/基础设施/环境三者一致时通过启动自检。"""
    for region in ("cn", "eu", "intl"):
        for env in ("dev", "staging", "production"):
            check_startup(region, region, env)


def test_check_startup_rejected() -> None:
    """异常路径：缺失/非法区域、跨区不一致、非法环境全部 fail-closed。"""
    cases = (
        (None, "cn", "dev"),
        ("cn", None, "dev"),
        ("cn", "eu", "dev"),
        ("cn", "cn", "qa"),
    )
    for data_region, infra_region, service_env in cases:
        with pytest.raises(ValueError):
            check_startup(data_region, infra_region, service_env)

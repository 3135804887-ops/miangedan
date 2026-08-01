"""区域自检单测：正常 + 异常路径（AGENTS.md 第 4 节）。"""

import pytest

from mgd_orchestrator import require_data_region


def test_valid_regions_accepted() -> None:
    """正常路径：三个批准区域原样返回。"""
    for region in ("cn", "eu", "intl"):
        assert require_data_region(region) == region


def test_invalid_regions_rejected() -> None:
    """异常路径：空值、大小写变体、非法区域、拼接串全部 fail-closed。"""
    for region in (None, "", "CN", "us", "cn,eu", " intl", "eu "):
        with pytest.raises(ValueError):
            require_data_region(region)

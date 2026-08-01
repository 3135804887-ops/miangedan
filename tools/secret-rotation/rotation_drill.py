#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
密钥轮换演练（TASK-006）：校验 REF 引用契约与区域配置一致性，模拟轮换准备序列。
追踪：IMPLEMENTATION_PLAN.md TASK-006；SECURITY-REQUIREMENTS.md SEC-012、4.7。
用法：python tools/secret-rotation/rotation_drill.py
失败即退出码 1（CI 阶段 1 经 key-mgmt 套件调用）。
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent.parent

ENV_VAR_PATTERN = re.compile(r"^[A-Z][A-Z0-9_]*$")
REF_PATTERN = re.compile(r"^[A-Z][A-Z0-9_]*_REF$")
FAILURES: list[str] = []


def fail(msg: str) -> None:
    FAILURES.append(msg)


def parse_env_example(path: Path) -> dict[str, str]:
    names: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.split("#", 1)[0].strip()
        if not line or "=" not in line:
            continue
        name, value = line.split("=", 1)
        name = name.strip()
        if not ENV_VAR_PATTERN.match(name):
            fail(f"[轮换演练] .env.example 变量名非法: {name!r}")
            continue
        names[name] = value.strip()
    return names


def main() -> int:
    import yaml
    env_example = ROOT / ".env.example"
    if not env_example.exists():
        fail("[轮换演练] 缺少 .env.example")
        return 1
    names = parse_env_example(env_example)
    if not names:
        fail("[轮换演练] .env.example 未解析到任何变量")

    ref_names = [n for n in names if n.endswith("_REF")]
    for name in ref_names:
        if not REF_PATTERN.match(name):
            fail(f"[轮换演练] 引用变量名非法: {name}")
    if not ref_names:
        fail("[轮换演练] .env.example 缺少 *_REF 引用变量")

    sec = ROOT / "docs/security/SECURITY-REQUIREMENTS.md"
    if not sec.exists() or "密钥轮换周期表" not in sec.read_text(encoding="utf-8"):
        fail("[轮换演练] SECURITY-REQUIREMENTS.md 缺少 4.7 密钥轮换周期表")

    # 区域拓扑 secrets.refs 与 kms_name 一致性（每区引用值必须在 .env.example 中声明）
    regions_dir = ROOT / "infra/regions"
    checked = 0
    for yaml_path in sorted(regions_dir.rglob("envs/*.yaml")):
        data = yaml.safe_load(yaml_path.read_text(encoding="utf-8"))
        secrets_cfg = (data.get("topology") or {}).get("secrets") or {}
        if not secrets_cfg.get("kms_name"):
            fail(f"[轮换演练] {yaml_path.relative_to(ROOT)} 缺少 secrets.kms_name")
        refs = secrets_cfg.get("refs") or {}
        if not refs:
            fail(f"[轮换演练] {yaml_path.relative_to(ROOT)} 缺少 secrets.refs")
        for _key, value in refs.items():
            checked += 1
            if value not in names:
                fail(f"[轮换演练] {yaml_path.relative_to(ROOT)} refs 值 {value} 未在 .env.example 声明")
    if checked == 0:
        fail("[轮换演练] 未校验到任何区域密钥引用")

    if FAILURES:
        for f in FAILURES:
            print(f"FAIL: {f}")
        print(f"\n共 {len(FAILURES)} 项失败")
        return 1
    print("密钥轮换演练通过：REF 契约、周期表与区域配置一致。")
    print("演练序列（不中断服务）：准备新版本 → 双写/双读过渡 → 切换引用 → 验证 → 退役旧版本 → 审计记录。")
    return 0


if __name__ == "__main__":
    sys.exit(main())

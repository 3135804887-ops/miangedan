#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""TASK-093 安全红队自动化套件运行器。

追踪：IMPLEMENTATION_PLAN.md TASK-093；docs/security/THREAT-MODEL.md（TM-01~TM-16）；
SECURITY-REQUIREMENTS（零容忍：注入/越权/跨租户/重复扣费/证据丢失）。

六类攻击（注入、越权、跨租户、恶意文件、重放、重复扣费）每类至少一组正常用例 +
一组攻击用例；攻击用例断言“命中即阻断”，用例通过即代表阻断成功，任一类失败
则套件失败（0 失败门禁）。

用法：
  python tools/security-redteam/run_redteam.py --write   # 执行全部选择器并写 ai/evals/reports/redteam.json
  python tools/security-redteam/run_redteam.py --check   # 执行并校验入库报告一致（CI 阶段5 门禁）
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parent.parent.parent
MANIFEST = ROOT / "tools" / "security-redteam" / "manifest.json"
REPORT = ROOT / "ai" / "evals" / "reports" / "redteam.json"
TIMEOUT_SECONDS = 600


def run_selector(cwd: str, cmd: list[str]) -> tuple[bool, str]:
    try:
        proc = subprocess.run(
            cmd,
            cwd=ROOT / cwd,
            capture_output=True,
            text=True,
            timeout=TIMEOUT_SECONDS,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        return False, f"执行失败：{exc}"
    if proc.returncode != 0:
        tail = (proc.stdout or proc.stderr).strip().splitlines()[-8:]
        return False, "；".join(tail)
    return True, ""


def run_class(cls: dict[str, Any]) -> dict[str, Any]:
    normal: list[dict[str, Any]] = []
    for sel in cls.get("normal", []):
        ok, err = run_selector(sel["cwd"], sel["cmd"])
        normal.append({"cmd": " ".join(sel["cmd"]), "cwd": sel["cwd"], "passed": ok, "error": err})
    attacks: list[dict[str, Any]] = []
    for sel in cls.get("attacks", []):
        ok, err = run_selector(sel["cwd"], sel["cmd"])
        attacks.append({"cmd": " ".join(sel["cmd"]), "cwd": sel["cwd"], "passed": ok, "error": err})
    passed = all(x["passed"] for x in normal) and all(x["passed"] for x in attacks)
    return {
        "id": cls["id"],
        "name": cls["name"],
        "threat_refs": cls.get("threat_refs", []),
        "evidence_refs": cls.get("evidence_refs", []),
        "normal_case_count": len(normal),
        "attack_case_count": len(attacks),
        "normal": normal,
        "attacks": attacks,
        "passed": passed,
    }


def build_report() -> dict[str, Any]:
    manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
    classes = [run_class(c) for c in manifest["classes"]]
    failures = [
        f"{c['id']}（{c['name']}）存在未通过用例"
        for c in classes
        if not c["passed"]
    ]
    return {
        "report_kind": "redteam_093",
        "generated_at": datetime.now(UTC).isoformat(),
        "classes": classes,
        "metrics": {
            "class_count": len(classes),
            "passed_class_count": sum(1 for c in classes if c["passed"]),
            "total_selectors": sum(c["normal_case_count"] + c["attack_case_count"] for c in classes),
            "attack_selectors": sum(c["attack_case_count"] for c in classes),
        },
        "passed": not failures,
        "failures": failures,
    }


def comparable(report: dict[str, Any]) -> dict[str, Any]:
    return {k: report[k] for k in ("report_kind", "classes", "metrics", "passed", "failures")}


def main() -> int:
    parser = argparse.ArgumentParser(description="TASK-093 安全红队自动化套件")
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--write", action="store_true", help="执行并写 redteam.json")
    group.add_argument("--check", action="store_true", help="执行并与入库报告比对（CI 门禁）")
    args = parser.parse_args()
    report = build_report()
    print(
        f"[TASK-093] 套件执行完成：{report['metrics']['passed_class_count']}/"
        f"{report['metrics']['class_count']} 类通过，选择器 "
        f"{report['metrics']['total_selectors']} 个（攻击 {report['metrics']['attack_selectors']} 个）"
    )
    if report["failures"]:
        for item in report["failures"]:
            print(" -", item)
    if args.write:
        REPORT.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        print(f"[TASK-093] 已生成 {REPORT.relative_to(ROOT)}")
    elif args.check:
        if not REPORT.exists():
            print(f"[TASK-093] 缺少入库报告 {REPORT.relative_to(ROOT)}（先 --write）")
            return 1
        committed = json.loads(REPORT.read_text(encoding="utf-8"))
        if comparable(committed) != comparable(report):
            print("[TASK-093] 入库报告与本次执行结果不一致（测试或清单已变更，请重新 --write）")
            return 1
        print("[TASK-093] 入库报告与本次执行一致")
    else:
        print("[TASK-093] 用法：--write 生成报告；--check 校验（CI 门禁）")
    return 1 if report["failures"] else 0


if __name__ == "__main__":
    sys.exit(main())

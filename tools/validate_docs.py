#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
面个蛋（MianGeDan）文档与契约校验脚本（开发工具，非产品源代码）。
追踪：IMPLEMENTATION_PLAN.md 第 10 节；AGENTS.md 第 7 节 DoD。
用法：python tools/validate_docs.py [--suites key1,key2]
  默认运行全部套件；CI 各阶段按套件键选用（见 .github/workflows/ci.yml）。
"""
import argparse
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
FAILURES: list[str] = []
WARNINGS: list[str] = []


def fail(msg: str) -> None:
    FAILURES.append(msg)


def warn(msg: str) -> None:
    WARNINGS.append(msg)


def read(p: Path) -> str:
    return p.read_text(encoding="utf-8")


# ---------- 1. 必须存在的文件 ----------
REQUIRED_FILES = [
    "AGENTS.md", "README.md", "IMPLEMENTATION_PLAN.md", "CHANGELOG.md", ".env.example",
    "docs/architecture/SYSTEM-ARCHITECTURE.md", "docs/architecture/DEPLOYMENT.md",
    "docs/architecture/adr/README.md",
    "docs/architecture/adr/ADR-0001-separate-business-workflow-from-ai-graph.md",
    "docs/architecture/adr/ADR-0002-separate-conversation-from-scoring.md",
    "docs/architecture/adr/ADR-0003-provider-adapter-layer.md",
    "docs/architecture/adr/ADR-0004-append-only-evidence-ledger.md",
    "docs/architecture/adr/ADR-0005-three-data-regions.md",
    "docs/domain/DOMAIN-MODEL.md", "docs/domain/INTERVIEW-STATE-MACHINE.md",
    "docs/domain/BILLING-STATE-MACHINE.md",
    "docs/api/openapi.yaml", "docs/api/realtime-events.md",
    "docs/ai/AI-ORCHESTRATION.md", "docs/ai/SCORING-SPEC.md", "docs/ai/HANDOFF-SPEC.md",
    "docs/ai/PROMPT-POLICY.md", "docs/ai/PROVIDER-ADAPTERS.md",
    "docs/data/DATA-MODEL.md", "docs/data/RETENTION-MATRIX.md", "docs/data/DATA-CLASSIFICATION.md",
    "docs/security/THREAT-MODEL.md", "docs/security/PRIVACY-DATA-MAP.md",
    "docs/security/SECURITY-REQUIREMENTS.md",
    "docs/design/SCREEN-SPEC.md", "docs/design/DESIGN-SYSTEM.md", "docs/design/ACCESSIBILITY.md",
    "docs/testing/ACCEPTANCE-MATRIX.md", "docs/testing/TEST-STRATEGY.md",
    "docs/testing/RELEASE-CHECKLIST.md",
    "config/rubrics/v1/default.yaml", "config/interview-flows/v1/default.yaml",
    "config/safety/policy.yaml", "config/feature-flags.yaml",
    "ai/prompts/README.md",
    "ai/schemas/resume-profile.schema.json", "ai/schemas/job-profile.schema.json",
    "ai/schemas/interview-plan.schema.json", "ai/schemas/turn-evidence.schema.json",
    "ai/schemas/handoff-package.schema.json", "ai/schemas/scoring-input.schema.json",
    "ai/schemas/scoring-result.schema.json", "ai/schemas/report.schema.json",
]


def check_required_files() -> None:
    for rel in REQUIRED_FILES:
        p = ROOT / rel
        if not p.exists():
            fail(f"[缺失文件] {rel}")
        elif p.stat().st_size < 400:
            fail(f"[疑似空文档] {rel}（<400 字节）")
    # 目录非空检查
    for d in ["ai/evals/datasets", "ai/evals/expected-results", "fixtures/synthetic"]:
        dp = ROOT / d
        if not dp.exists() or not any(dp.rglob("*.*")):
            fail(f"[空目录] {d}")


# ---------- 2. YAML 解析 ----------
def check_yaml() -> None:
    import yaml
    for p in sorted(ROOT.rglob("*.yaml")) + sorted(ROOT.rglob("*.yml")):
        if "node_modules" in p.parts:
            continue
        try:
            yaml.safe_load(read(p))
        except Exception as e:  # noqa: BLE001
            fail(f"[YAML 解析失败] {p.relative_to(ROOT)}: {e}")


# ---------- 3. JSON / JSONL 解析 ----------
def check_json() -> None:
    for p in sorted(ROOT.rglob("*.json")):
        if "node_modules" in p.parts:
            continue
        try:
            json.loads(read(p))
        except Exception as e:  # noqa: BLE001
            fail(f"[JSON 解析失败] {p.relative_to(ROOT)}: {e}")
    for p in sorted(ROOT.rglob("*.jsonl")):
        for i, line in enumerate(read(p).splitlines(), 1):
            if not line.strip():
                continue
            try:
                obj = json.loads(line)
                if obj.get("synthetic") is not True:
                    fail(f"[合成标记缺失] {p.relative_to(ROOT)} 第 {i} 行 synthetic != true")
            except Exception as e:  # noqa: BLE001
                fail(f"[JSONL 解析失败] {p.relative_to(ROOT)} 第 {i} 行: {e}")


# ---------- 4. JSON Schema 元校验 ----------
def check_json_schemas() -> None:
    from jsonschema import Draft202012Validator
    for p in sorted((ROOT / "ai/schemas").glob("*.schema.json")):
        try:
            schema = json.loads(read(p))
            Draft202012Validator.check_schema(schema)
            if schema.get("$schema") != "https://json-schema.org/draft/2020-12/schema":
                fail(f"[Schema draft 不符] {p.name}")
        except Exception as e:  # noqa: BLE001
            fail(f"[JSON Schema 无效] {p.name}: {e}")


# ---------- 5. OpenAPI 校验 ----------
def check_openapi() -> None:
    p = ROOT / "docs/api/openapi.yaml"
    if not p.exists():
        return
    try:
        import yaml
        from openapi_spec_validator import validate
        spec = yaml.safe_load(read(p))
        validate(spec)
        if not str(spec.get("openapi", "")).startswith("3.1"):
            fail("[OpenAPI] 版本不是 3.1.x")
    except Exception as e:  # noqa: BLE001
        fail(f"[OpenAPI 校验失败] {e}")


# ---------- 6. Mermaid / 代码块闭合 ----------
def check_fences() -> None:
    for p in sorted(ROOT.rglob("*.md")):
        text = read(p)
        fences = re.findall(r"^```", text, flags=re.M)
        if len(fences) % 2 != 0:
            fail(f"[代码块未闭合] {p.relative_to(ROOT)}（fence 数 {len(fences)}）")


# ---------- 7. 占位符检查 ----------
PLACEHOLDER_PATTERNS = [r"\bTBD\b", r"\bTODO\b", r"\bFIXME\b", r"后续补充", r"待补充", r"此处省略", r"lorem ipsum"]


def check_placeholders() -> None:
    for p in sorted(ROOT.rglob("*")):
        if p.is_dir() or p.suffix.lower() not in {".md", ".yaml", ".yml", ".json", ".jsonl"}:
            continue
        if "node_modules" in p.parts or "prd" in p.parts:
            continue
        text = read(p)
        for pat in PLACEHOLDER_PATTERNS:
            for m in re.finditer(pat, text, flags=re.I):
                line = text[: m.start()].count("\n") + 1
                fail(f"[占位符] {p.relative_to(ROOT)}:{line} 命中 {pat!r}")


# ---------- 8. PRD 需求覆盖 ----------
def all_req_ids() -> list[str]:
    ids = [f"US-0{i}" for i in range(1, 9)]
    ids += [f"FR-{i:03d}" for i in range(1, 41)]
    ids += [f"NFR-{i:03d}" for i in range(1, 17)]
    return ids


def check_coverage() -> None:
    matrix = ROOT / "docs/testing/ACCEPTANCE-MATRIX.md"
    plan = ROOT / "IMPLEMENTATION_PLAN.md"
    for target in [matrix, plan]:
        if not target.exists():
            continue
        text = read(target)
        for rid in all_req_ids():
            if rid not in text:
                fail(f"[覆盖缺失] {rid} 未出现在 {target.name}")
    # 验收矩阵测试 ID 计数
    if matrix.exists():
        text = read(matrix)
        for rid in all_req_ids():
            has_n = re.search(rf"TC-{rid}-N\d+", text)
            has_a = re.search(rf"TC-{rid}-A\d+", text)
            if not has_n:
                fail(f"[验收矩阵] {rid} 缺少正常场景用例 TC-{rid}-Nxx")
            if not has_a:
                fail(f"[验收矩阵] {rid} 缺少异常场景用例 TC-{rid}-Axx")


# ---------- 9. 跨文件一致性 ----------
DIMENSION_KEYS = [
    "professional_competence", "problem_solving", "communication",
    "experience_evidence", "behavioral_collaboration", "learning_adaptability",
]
PROJECT_STATES = [
    "DRAFT", "PARSING", "MATERIAL_REVIEW", "PARSE_FAILED", "PLAN_GENERATING",
    "PLAN_REVIEW", "PLAN_FAILED", "READY", "IN_SESSION", "SCORING",
    "ROUND_PASSED", "ROUND_FAILED", "PRACTICING", "EVALUATION_INCOMPLETE", "COMPLETED",
]


def check_consistency() -> None:
    files_to_scan = [
        "config/rubrics/v1/default.yaml", "docs/ai/SCORING-SPEC.md",
        "ai/schemas/scoring-input.schema.json", "ai/schemas/scoring-result.schema.json",
        "ai/schemas/interview-plan.schema.json",
    ]
    for rel in files_to_scan:
        p = ROOT / rel
        if not p.exists():
            continue
        text = read(p)
        for key in DIMENSION_KEYS:
            if key not in text:
                fail(f"[一致性] 维度键 {key} 缺失于 {rel}")
    # 量表默认权重一致
    rubric = ROOT / "config/rubrics/v1/default.yaml"
    spec = ROOT / "docs/ai/SCORING-SPEC.md"
    if rubric.exists() and spec.exists():
        rt, st = read(rubric), read(spec)
        for key, w in zip(DIMENSION_KEYS, [25, 20, 15, 15, 15, 10]):
            m = re.search(rf"key: {key}.*?default_weight: (\d+)", rt, flags=re.S)
            if not m or int(m.group(1)) != w:
                fail(f"[一致性] rubric 中 {key} 默认权重 != {w}")
        for anchor, score in [(1, 20), (2, 40), (3, 60), (4, 80), (5, 100)]:
            if not re.search(rf"level: {anchor}\s*\n\s*mapped_score: {score}", rt):
                fail(f"[一致性] rubric 锚点 {anchor}->{score} 缺失或不匹配")
        if "25 / 20 / 15 / 15 / 15 / 10" not in st:
            warn("[一致性] SCORING-SPEC 未含默认权重串，请人工核对")
    # 状态枚举在状态机与 OpenAPI 一致
    sm = ROOT / "docs/domain/INTERVIEW-STATE-MACHINE.md"
    api = ROOT / "docs/api/openapi.yaml"
    if sm.exists() and api.exists():
        api_text = read(api)
        for state in PROJECT_STATES:
            if state not in api_text:
                warn(f"[一致性] 项目状态 {state} 未在 openapi.yaml 出现（请人工确认是否有意）")
    # 评测集：expected-results 键与数据集 case_id 对齐
    ds_dir = ROOT / "ai/evals/datasets"
    er_dir = ROOT / "ai/evals/expected-results"
    if ds_dir.exists() and er_dir.exists():
        for ds in sorted(ds_dir.glob("*.jsonl")):
            expected = er_dir / (ds.stem + ".expected.json")
            if not expected.exists():
                fail(f"[评测集] 缺少预期结果文件 {expected.name}")
                continue
            case_ids = [json.loads(l)["case_id"] for l in read(ds).splitlines() if l.strip()]
            exp = json.loads(read(expected))
            for cid in case_ids:
                if cid not in exp:
                    fail(f"[评测集] {ds.name} 的 {cid} 缺少预期结果")
            for cid in exp:
                if cid not in case_ids:
                    fail(f"[评测集] 预期结果 {cid} 在 {ds.name} 中无对应用例")


# ---------- 9b. 配置语义校验 ----------
def check_semantics() -> None:
    import yaml
    rubric = ROOT / "config/rubrics/v1/default.yaml"
    if rubric.exists():
        data = yaml.safe_load(read(rubric))
        weights = [d.get("default_weight") for d in data.get("dimensions", [])]
        if sum(w for w in weights if isinstance(w, int)) != 100:
            fail(f"[语义] rubric 默认权重总和 != 100：{weights}")
        anchors = {a["level"]: a["mapped_score"] for a in data.get("anchors", [])}
        if anchors != {1: 20, 2: 40, 3: 60, 4: 80, 5: 100}:
            fail(f"[语义] rubric 锚点映射错误：{anchors}")
    flows = ROOT / "config/interview-flows/v1/default.yaml"
    if flows.exists():
        data = yaml.safe_load(read(flows))
        types = {r["key"] for r in data.get("round_types", [])}
        for dr in data.get("default_rounds", []):
            if dr["type"] not in types:
                fail(f"[语义] default_rounds 引用未注册轮次类型：{dr['type']}")
        for rt in data.get("round_types", []):
            for cd in rt.get("default_critical_dimensions", []):
                if cd not in DIMENSION_KEYS:
                    fail(f"[语义] 轮次类型 {rt['key']} 关键维度非法：{cd}")
        bounds = data.get("bounds", {})
        if bounds.get("rounds", {}).get("user_configurable") != {"min": 1, "max": 5}:
            fail("[语义] 轮次用户边界不是 1–5")
        if bounds.get("duration_minutes", {}).get("user_configurable") != {"min": 10, "max": 60}:
            fail("[语义] 时长用户边界不是 10–60")
    flags = ROOT / "config/feature-flags.yaml"
    if flags.exists():
        data = yaml.safe_load(read(flags))
        for f_ in data.get("flags", []):
            if f_.get("protected") is True and f_.get("default") is not True:
                fail(f"[语义] 受保护开关 {f_.get('key')} 默认值必须为 true")
    safety = ROOT / "config/safety/policy.yaml"
    if safety.exists():
        data = yaml.safe_load(read(safety))
        if not data.get("protected_attributes"):
            fail("[语义] safety policy 缺少 protected_attributes")


# ---------- 10. 真实密钥/敏感信息扫描 ----------
SECRET_PATTERNS = [
    (r"sk-[A-Za-z0-9]{20,}", "疑似 OpenAI 风格密钥"),
    (r"-----BEGIN [A-Z ]*PRIVATE KEY-----", "私钥材料"),
    (r"(?i)(password|passwd|secret|token)\s*[:=]\s*['\"]?[A-Za-z0-9+/=_-]{16,}", "疑似硬编码凭证"),
]


def check_secrets() -> None:
    for p in sorted(ROOT.rglob("*")):
        if p.is_dir() or "node_modules" in p.parts or p.suffix.lower() not in {
            ".md", ".yaml", ".yml", ".json", ".jsonl", ".example", ".py",
        }:
            continue
        text = read(p)
        for pat, label in SECRET_PATTERNS:
            for m in re.finditer(pat, text):
                line = text[: m.start()].count("\n") + 1
                fail(f"[疑似密钥] {p.relative_to(ROOT)}:{line} {label}")


SUITES = [
    ("required", "必须文件", check_required_files),
    ("yaml", "YAML 解析", check_yaml),
    ("json", "JSON/JSONL 解析", check_json),
    ("schema", "JSON Schema 元校验", check_json_schemas),
    ("openapi", "OpenAPI 校验", check_openapi),
    ("fences", "代码块闭合", check_fences),
    ("placeholders", "占位符", check_placeholders),
    ("coverage", "需求覆盖", check_coverage),
    ("consistency", "跨文件一致性", check_consistency),
    ("semantics", "配置语义", check_semantics),
    ("secrets", "密钥扫描", check_secrets),
]


def main() -> int:
    parser = argparse.ArgumentParser(description="面个蛋文档与契约校验（CI 门禁）")
    parser.add_argument(
        "--suites",
        default="all",
        help="逗号分隔的套件键，默认 all 全部：" + ",".join(k for k, _, _ in SUITES),
    )
    args = parser.parse_args()
    keys = [k.strip() for k in args.suites.split(",") if k.strip()]
    known = {k for k, _, _ in SUITES}
    if args.suites != "all":
        for k in keys:
            if k not in known:
                fail(f"[参数错误] 未知校验套件：{k}")
    selected = SUITES if args.suites == "all" else [s for s in SUITES if s[0] in keys]
    for _key, name, fn in selected:
        try:
            fn()
            print(f"[运行] {name} 完成")
        except Exception as e:  # noqa: BLE001
            fail(f"[校验器异常] {name}: {e}")
    print("\n===== 校验结果 =====")
    for w in WARNINGS:
        print(f"WARN: {w}")
    if FAILURES:
        for f_ in FAILURES:
            print(f"FAIL: {f_}")
        print(f"\n共 {len(FAILURES)} 项失败、{len(WARNINGS)} 项警告")
        return 1
    print(f"全部通过（{len(WARNINGS)} 项警告）")
    return 0


if __name__ == "__main__":
    sys.exit(main())

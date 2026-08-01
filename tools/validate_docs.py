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


# ---------- 9c. 区域拓扑校验（TASK-002） ----------
REGION_CODES = ["cn", "eu", "intl"]
ENVIRONMENTS = ["dev", "staging", "production"]
EVENT_TOPICS = [
    "parse.jobs", "scoring.requests", "report.jobs",
    "notification.outbox", "deletion.tasks", "compensation.jobs",
]
TOPOLOGY_REQUIRED_KEYS = [
    "network", "database", "object_storage", "event_stream", "sfu", "temporal",
    "secrets", "provider_allowlist", "notification", "routing",
]
TASK_QUEUES = ["ingestion", "plan", "interview", "scoring", "report", "billing", "deletion"]
PROVIDER_CATEGORIES = ["llm", "asr", "tts", "avatar", "email", "payment"]
RESOURCE_NAME_PATHS = [
    ("network", "vpc_name"),
    ("database", "postgres", "cluster_name"),
    ("database", "redis", "cluster_name"),
    ("object_storage", "buckets", "uploads"),
    ("object_storage", "buckets", "exports"),
    ("object_storage", "buckets", "media"),
    ("event_stream", "name"),
    ("sfu", "node_group"),
    ("temporal", "cluster_name"),
    ("temporal", "namespace"),
]


def _iter_string_values(node: object):
    if isinstance(node, str):
        yield node
    elif isinstance(node, dict):
        for value in node.values():
            yield from _iter_string_values(value)
    elif isinstance(node, list):
        for value in node:
            yield from _iter_string_values(value)


def _validate_region_topology(data, region: str, env: str, p: Path) -> None:
    rel = p.relative_to(ROOT)
    if data.get("region") != region:
        fail(f"[区域拓扑] {rel} region 应为 {region}")
    if data.get("environment") != env:
        fail(f"[区域拓扑] {rel} environment 应为 {env}")
    topo = data.get("topology")
    if not isinstance(topo, dict):
        fail(f"[区域拓扑] {rel} 缺少 topology 段")
        return
    for key in TOPOLOGY_REQUIRED_KEYS:
        if key not in topo:
            fail(f"[区域拓扑] {rel} 缺少 topology.{key}")
    network = topo.get("network", {})
    azs = network.get("availability_zones")
    if not isinstance(azs, list) or len(azs) < 3 or len(set(azs)) < 3:
        fail(f"[区域拓扑] {rel} 可用区不足 3（NFR-004）")
    if network.get("cross_region_peering"):
        fail(f"[区域拓扑] {rel} 禁止跨区对等连接（ADR-0005）")
    db = topo.get("database", {})
    pg = db.get("postgres", {})
    min_replicas = {"dev": 1, "staging": 2, "production": 3}[env]
    if not isinstance(pg.get("replicas"), int) or pg.get("replicas", 0) < min_replicas:
        fail(f"[区域拓扑] {rel} postgres replicas 应 ≥ {min_replicas}")
    redis_cfg = db.get("redis", {})
    if redis_cfg.get("evidence_storage") is not False:
        fail(f"[区域拓扑] {rel} Redis 必须标注 evidence_storage: false（非证据存储）")
    buckets = topo.get("object_storage", {}).get("buckets", {})
    for bucket_key in ["uploads", "exports", "media"]:
        if not buckets.get(bucket_key):
            fail(f"[区域拓扑] {rel} 缺少对象存储桶 {bucket_key}")
    object_storage = topo.get("object_storage", {})
    if object_storage.get("media_lifecycle_days") != 30:
        fail(f"[区域拓扑] {rel} object_storage.media_lifecycle_days 必须为 30（RETENTION-MATRIX）")
    queues = topo.get("temporal", {}).get("task_queues", [])
    for q in TASK_QUEUES:
        if q not in queues:
            fail(f"[区域拓扑] {rel} temporal.task_queues 缺少 {q}")
    temporal_cfg = topo.get("temporal", {})
    if temporal_cfg.get("cross_az") is not True:
        fail(f"[区域拓扑] {rel} temporal.cross_az 必须为 true（跨 AZ 故障可恢复）")
    retention = temporal_cfg.get("history_retention_days")
    if not isinstance(retention, int) or retention < 30:
        fail(f"[区域拓扑] {rel} temporal.history_retention_days 应 ≥ 30")
    event_stream = topo.get("event_stream", {})
    topics = event_stream.get("topics", [])
    for topic in EVENT_TOPICS:
        if topic not in topics:
            fail(f"[区域拓扑] {rel} event_stream.topics 缺少 {topic}")
    refs = topo.get("secrets", {}).get("refs", {})
    if not isinstance(refs, dict) or not refs:
        fail(f"[区域拓扑] {rel} 缺少 secrets.refs")
    allow = topo.get("provider_allowlist", {})
    for cat in PROVIDER_CATEGORIES:
        ids = allow.get(cat, [])
        if not ids:
            fail(f"[区域拓扑] {rel} provider_allowlist.{cat} 为空")
        for pid in ids:
            if not isinstance(pid, str) or f"_{region}_" not in pid:
                fail(f"[区域拓扑] {rel} 供应商 {pid!r} 未含区域 {region} 标识")
    routing = topo.get("routing", {})
    if routing.get("gateway_region") != region:
        fail(f"[区域拓扑] {rel} routing.gateway_region 应为 {region}")
    if routing.get("reject_mismatch") is not True:
        fail(f"[区域拓扑] {rel} routing.reject_mismatch 必须为 true")
    if routing.get("alert_event") != "region_mismatch":
        fail(f"[区域拓扑] {rel} routing.alert_event 必须为 region_mismatch")
    resource_env = {"production": "prod"}.get(env, env)
    resource_prefix = f"mgd-{region}-{resource_env}-"
    for keys in RESOURCE_NAME_PATHS:
        node: object = topo
        for key in keys:
            node = node.get(key, {}) if isinstance(node, dict) else {}
        name = node if isinstance(node, str) else None
        if not name or not name.startswith(resource_prefix):
            fail(f"[区域拓扑] {rel} 资源 {'/'.join(keys)} 命名不符合 {resource_prefix} 前缀")
    for other in REGION_CODES:
        if other == region:
            continue
        pattern = re.compile(rf"(?<![a-z0-9]){re.escape(other)}(?![a-z0-9])")
        for value in _iter_string_values(topo):
            if pattern.search(value):
                fail(f"[区域拓扑] {rel} 发现跨区引用 {other}: {value!r}")


def check_regions() -> None:
    import yaml
    base = ROOT / "infra/regions"
    if not base.exists():
        fail("[区域拓扑] 缺少 infra/regions")
        return
    parsed = {}
    for region in REGION_CODES:
        for env in ENVIRONMENTS:
            p = base / region / "envs" / f"{env}.yaml"
            if not p.exists():
                fail(f"[区域拓扑] 缺少 {p.relative_to(ROOT)}")
                continue
            try:
                data = yaml.safe_load(read(p))
            except Exception as e:  # noqa: BLE001
                fail(f"[区域拓扑] 解析失败 {p.relative_to(ROOT)}: {e}")
                continue
            parsed[(region, env)] = data
            _validate_region_topology(data, region, env, p)
    if parsed:
        key_sets = {tuple(sorted(d.get("topology", {}).keys())) for d in parsed.values()}
        if len(key_sets) != 1:
            fail("[区域拓扑] 9 个环境拓扑组件键不一致（dev/staging/prod 拓扑必须同构）")


# ---------- 9d. 数据平台契约校验（TASK-003） ----------
DATA_MODULE_MANIFESTS = [
    ("database", "infra/modules/database/module.yaml"),
    ("object-storage", "infra/modules/object-storage/module.yaml"),
    ("event-stream", "infra/modules/event-stream/module.yaml"),
]
LEDGER_TABLES = ["evidence_items", "score_versions", "usage_ledger", "access_audits"]
BUSINESS_ROLES = ["mgd_app_runtime", "mgd_ledger_writer"]


def check_data_platform() -> None:
    import yaml
    for kind, rel in DATA_MODULE_MANIFESTS:
        p = ROOT / rel
        if not p.exists():
            fail(f"[数据平台] 缺少模块清单 {rel}")
            continue
        try:
            data = yaml.safe_load(read(p))
        except Exception as e:  # noqa: BLE001
            fail(f"[数据平台] 模块清单解析失败 {rel}: {e}")
            continue
        if data.get("kind") != kind:
            fail(f"[数据平台] {rel} kind 应为 {kind}")
    migration_dir = ROOT / "services/migrate/migrations"
    if not migration_dir.exists():
        fail("[数据平台] 缺少 services/migrate/migrations")
        return
    sql_files = sorted(migration_dir.glob("*.sql"))
    if not sql_files:
        fail("[数据平台] 迁移目录为空")
        return
    for p in sql_files:
        if not re.fullmatch(r"\d{4}_[a-z0-9_-]+\.sql", p.name):
            fail(f"[数据平台] 迁移文件名不符合 NNNN_name.sql: {p.name}")
    baseline = read(sql_files[0])
    for table in LEDGER_TABLES:
        if f"CREATE TABLE {table}" not in baseline:
            fail(f"[数据平台] 基线迁移缺少表 {table}")
    if "REVOKE UPDATE, DELETE" not in baseline:
        fail("[数据平台] 基线迁移缺少 REVOKE UPDATE, DELETE")
    if "idempotency_key" not in baseline:
        fail("[数据平台] 基线迁移缺少幂等键")
    if "data_region" not in baseline:
        fail("[数据平台] 基线迁移缺少 data_region")
    for role in BUSINESS_ROLES:
        for keyword in ["UPDATE", "DELETE"]:
            pattern = re.compile(
                rf"GRANT[^\n]*\b{keyword}\b[^\n]*TO\s+{re.escape(role)}\b", re.I
            )
            if pattern.search(baseline):
                fail(f"[数据平台] 业务角色 {role} 不得获得 {keyword} 权限")
    runner = ROOT / "services/migrate/migrate.go"
    pgstore = ROOT / "services/migrate/pgstore.go"
    if runner.exists() and pgstore.exists():
        runner_text = read(runner) + read(pgstore)
        if "schema_migrations" not in runner_text or "Checksum" not in runner_text:
            fail("[数据平台] 迁移执行器缺少 schema_migrations/Checksum 幂等机制")
    test_file = ROOT / "services/migrate/migrate_test.go"
    if test_file.exists() and "TestApplyIdempotent" not in read(test_file):
        fail("[数据平台] 迁移测试缺少幂等用例 TestApplyIdempotent")


# ---------- 9f. Temporal 契约校验（TASK-004） ----------
TEMPORAL_TASK_QUEUES = ["ingestion", "plan", "interview", "scoring", "report", "billing", "deletion"]


def check_temporal() -> None:
    import yaml
    module_rel = "infra/modules/temporal/module.yaml"
    p = ROOT / module_rel
    if not p.exists():
        fail(f"[Temporal] 缺少模块清单 {module_rel}")
        return
    try:
        data = yaml.safe_load(read(p))
    except Exception as e:  # noqa: BLE001
        fail(f"[Temporal] 模块清单解析失败: {e}")
        return
    if data.get("kind") != "temporal":
        fail("[Temporal] 模块清单 kind 应为 temporal")
    queues = data.get("task_queues", {})
    for q in TEMPORAL_TASK_QUEUES:
        if q not in queues:
            fail(f"[Temporal] 模块清单 task_queues 缺少 {q}")
    pattern = data.get("namespace", {}).get("pattern")
    if pattern != "mgd-{region}-{env}-temporal":
        fail(f"[Temporal] namespace.pattern 应为 mgd-{{region}}-{{env}}-temporal，实际 {pattern!r}")
    cluster = data.get("cluster", {})
    if cluster.get("cross_az") is not True or cluster.get("min_az", 0) < 3:
        fail("[Temporal] 模块清单 cluster 必须 cross_az=true 且 min_az ≥ 3")
    pkg = ROOT / "services/temporal/temporal.go"
    if pkg.exists():
        pkg_text = read(pkg)
        for symbol in ["Namespace", "ValidateConfig", "AllTaskQueues"]:
            if symbol not in pkg_text:
                fail(f"[Temporal] services/temporal 缺少 {symbol}")
    test_file = ROOT / "services/temporal/temporal_test.go"
    if test_file.exists() and "TestValidateConfig" not in read(test_file):
        fail("[Temporal] 缺少配置校验测试 TestValidateConfig")


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
    ("regions", "区域拓扑", check_regions),
    ("data-platform", "数据平台", check_data_platform),
    ("temporal", "Temporal 契约", check_temporal),
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

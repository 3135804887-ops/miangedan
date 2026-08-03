"""OD-01 provider evaluation runner (Phase 0 sandbox measurement toolkit).

Subcommands:
  sessions      validate generated synthetic sessions and print counts
  percentiles   compute P50/P95/P99 from a samples JSONL file and compare
                against NFR thresholds (OD-01 section 6)
  scorecard     assemble a six-dimension scorecard for one capability x
                region x vendor from measured samples
  check-config  report which region-scoped vendor credentials are present
                (values are never printed)
"""

from __future__ import annotations

import argparse
import json
import math
import os
import sys
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[4]
SESSIONS_DIR = REPO_ROOT / "ai" / "evals" / "providers" / "sessions"
SAMPLES_DIR = REPO_ROOT / "ai" / "evals" / "providers" / "samples"
SCORECARDS_DIR = REPO_ROOT / "ai" / "evals" / "providers" / "scorecards"

CAPABILITIES = ("webrtc_sfu", "asr", "tts", "llm", "avatar")
REGIONS = ("cn", "eu", "intl")
ROLES = ("primary", "secondary")

# Thresholds from OD-01 section 6 / NFR-007 ~ NFR-012 (milliseconds).
THRESHOLDS: dict[str, dict[str, float | None]] = {
    "connect": {"p50": None, "p95": 8000.0, "p99": 15000.0},
    "e2e_response": {"p50": 1500.0, "p95": 3000.0, "p99": 5000.0},
    "interrupt": {"p50": None, "p95": 500.0, "p99": None},
    "asr_final": {"p50": None, "p95": 1000.0, "p99": None},
    "lipsync": {"max": 200.0},
}

DIMENSION_WEIGHTS = {
    "performance": 30,
    "quality": 25,
    "compliance": 15,
    "licensing": 10,
    "replaceability": 10,
    "cost": 10,
}

# Vendor -> region-scoped env vars, aligned with .env.example.
VENDOR_ENV_MAP: dict[str, dict[str, tuple[str, ...]]] = {
    "cn": {
        "agora": ("WEBRTC_SFU_URL", "WEBRTC_API_KEY", "WEBRTC_API_SECRET"),
        "trtc": ("WEBRTC_SFU_URL", "WEBRTC_API_KEY", "WEBRTC_API_SECRET"),
        "aliyun_asr": ("ASR_PRIMARY_BASE_URL", "ASR_PRIMARY_API_KEY"),
        "iflytek_asr": ("ASR_PRIMARY_BASE_URL", "ASR_PRIMARY_API_KEY"),
        "volcano_tts": ("TTS_PRIMARY_BASE_URL", "TTS_PRIMARY_API_KEY"),
        "iflytek_tts": ("TTS_PRIMARY_BASE_URL", "TTS_PRIMARY_API_KEY"),
        "deepseek": ("LLM_PRIMARY_BASE_URL", "LLM_PRIMARY_API_KEY", "LLM_MODEL_PINNED_VERSION"),
        "qwen": ("LLM_SECONDARY_BASE_URL", "LLM_SECONDARY_API_KEY"),
        "tencent_avatar": (
            "AVATAR_PRIMARY_BASE_URL",
            "AVATAR_PRIMARY_API_KEY",
            "AVATAR_CHARACTER_LICENSE_REF",
        ),
        "silicon_avatar": (
            "AVATAR_PRIMARY_BASE_URL",
            "AVATAR_PRIMARY_API_KEY",
            "AVATAR_CHARACTER_LICENSE_REF",
        ),
    },
    "eu": {
        "livekit": ("WEBRTC_SFU_URL", "WEBRTC_API_KEY", "WEBRTC_API_SECRET"),
        "hundred_ms": ("WEBRTC_SFU_URL", "WEBRTC_API_KEY", "WEBRTC_API_SECRET"),
        "deepgram": ("ASR_PRIMARY_BASE_URL", "ASR_PRIMARY_API_KEY"),
        "azure_asr": ("ASR_SECONDARY_BASE_URL", "ASR_SECONDARY_API_KEY"),
        "elevenlabs": ("TTS_PRIMARY_BASE_URL", "TTS_PRIMARY_API_KEY"),
        "azure_tts": ("TTS_SECONDARY_BASE_URL", "TTS_SECONDARY_API_KEY"),
        "azure_openai": ("LLM_PRIMARY_BASE_URL", "LLM_PRIMARY_API_KEY", "LLM_MODEL_PINNED_VERSION"),
        "claude": ("LLM_SECONDARY_BASE_URL", "LLM_SECONDARY_API_KEY"),
        "heygen": (
            "AVATAR_PRIMARY_BASE_URL",
            "AVATAR_PRIMARY_API_KEY",
            "AVATAR_CHARACTER_LICENSE_REF",
        ),
        "synthesia": (
            "AVATAR_SECONDARY_BASE_URL",
            "AVATAR_SECONDARY_API_KEY",
            "AVATAR_CHARACTER_LICENSE_REF",
        ),
    },
    "intl": {
        "livekit": ("WEBRTC_SFU_URL", "WEBRTC_API_KEY", "WEBRTC_API_SECRET"),
        "hundred_ms": ("WEBRTC_SFU_URL", "WEBRTC_API_KEY", "WEBRTC_API_SECRET"),
        "deepgram": ("ASR_PRIMARY_BASE_URL", "ASR_PRIMARY_API_KEY"),
        "azure_asr": ("ASR_SECONDARY_BASE_URL", "ASR_SECONDARY_API_KEY"),
        "elevenlabs": ("TTS_PRIMARY_BASE_URL", "TTS_PRIMARY_API_KEY"),
        "azure_tts": ("TTS_SECONDARY_BASE_URL", "TTS_SECONDARY_API_KEY"),
        "azure_openai": ("LLM_PRIMARY_BASE_URL", "LLM_PRIMARY_API_KEY", "LLM_MODEL_PINNED_VERSION"),
        "claude": ("LLM_SECONDARY_BASE_URL", "LLM_SECONDARY_API_KEY"),
        "heygen": (
            "AVATAR_PRIMARY_BASE_URL",
            "AVATAR_PRIMARY_API_KEY",
            "AVATAR_CHARACTER_LICENSE_REF",
        ),
        "synthesia": (
            "AVATAR_SECONDARY_BASE_URL",
            "AVATAR_SECONDARY_API_KEY",
            "AVATAR_CHARACTER_LICENSE_REF",
        ),
    },
}


def _read_jsonl(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def _percentile(values: list[float], quantile: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    if len(ordered) == 1:
        return ordered[0]
    rank = (len(ordered) - 1) * quantile
    lower = math.floor(rank)
    upper = math.ceil(rank)
    if lower == upper:
        return ordered[int(rank)]
    return ordered[lower] * (upper - rank) + ordered[upper] * (rank - lower)


def _round1(value: float | None) -> float | None:
    return None if value is None else round(value, 1)


def _metric_stats(samples: list[dict[str, Any]]) -> dict[str, dict[str, float | None]]:
    by_metric: dict[str, list[float]] = {}
    for sample in samples:
        metric = sample.get("metric")
        value = sample.get("value_ms")
        if isinstance(metric, str) and isinstance(value, (int, float)):
            by_metric.setdefault(metric, []).append(float(value))
    stats: dict[str, dict[str, float | None]] = {}
    for metric, values in sorted(by_metric.items()):
        stats[metric] = {
            "n": len(values),
            "p50": _round1(_percentile(values, 0.50)),
            "p95": _round1(_percentile(values, 0.95)),
            "p99": _round1(_percentile(values, 0.99)),
        }
    return stats


def _threshold_failures(
    stats: dict[str, dict[str, float | None]],
) -> list[str]:
    failures: list[str] = []
    for metric, entry in stats.items():
        for stat, limit in THRESHOLDS.get(metric, {}).items():
            if limit is None:
                continue
            measured = entry.get(stat)
            if measured is not None and measured > limit:
                failures.append(
                    f"{metric}.{stat}: {measured:.0f}ms > {limit:.0f}ms"
                )
    return failures


def cmd_sessions(args: argparse.Namespace) -> int:
    if not SESSIONS_DIR.exists():
        print("no sessions generated; run scripts/generate_sessions.py first")
        return 1
    manifest_path = SESSIONS_DIR / "manifest.json"
    if manifest_path.exists():
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        counts = manifest.get("counts", {})
        for language in sorted(counts):
            print(f"{language}: {counts[language]} sessions")
        return 0
    for path in sorted(SESSIONS_DIR.rglob("*.jsonl")):
        rows = _read_jsonl(path)
        print(f"{path.relative_to(REPO_ROOT)}: {len(rows)} rows")
    return 0


def cmd_percentiles(args: argparse.Namespace) -> int:
    samples = _read_jsonl(Path(args.samples))
    stats = _metric_stats(samples)
    failures = _threshold_failures(stats)
    report = {
        "kind": "provider_eval_percentiles",
        "synthetic": all(bool(s.get("synthetic")) for s in samples),
        "source": args.samples,
        "metrics": stats,
        "passed": not failures,
        "failures": failures,
    }
    print(json.dumps(report, ensure_ascii=False, indent=2))
    if args.check and failures:
        return 1
    return 0


def _find_samples(capability: str, region: str, vendor: str) -> Path | None:
    candidates = [
        SAMPLES_DIR / f"{capability}_{region}_{vendor}.jsonl",
        SAMPLES_DIR / f"{capability}_{region}_{vendor}.sample.jsonl",
    ]
    for candidate in candidates:
        if candidate.exists():
            return candidate
    return None


def cmd_scorecard(args: argparse.Namespace) -> int:
    samples_path = _find_samples(args.capability, args.region, args.vendor)
    measured = samples_path is not None
    metrics: dict[str, dict[str, float | None]] = {}
    red_lines: list[dict[str, Any]] = []

    if samples_path is not None:
        samples = _read_jsonl(samples_path)
        metrics = _metric_stats(samples)
        for metric, entry in metrics.items():
            for stat, limit in THRESHOLDS.get(metric, {}).items():
                if limit is None:
                    continue
                value = entry.get(stat)
                if value is not None and value > limit:
                    red_lines.append(
                        {
                            "rule": f"{metric}.{stat} <= {limit:.0f}ms",
                            "verdict": "fail",
                            "measured_ms": value,
                        }
                    )

    dimensions: dict[str, dict[str, Any]] = {}
    for name, weight in DIMENSION_WEIGHTS.items():
        dimensions[name] = {
            "score": None,
            "weight": weight,
            "evidence_ref": None,
            "status": "pending",
        }
    if measured:
        dimensions["performance"]["status"] = "measured"

    scorecard = {
        "scorecard_kind": "provider_eval_scorecard",
        "capability": args.capability,
        "region": args.region,
        "vendor": args.vendor,
        "role": args.role,
        "status": "measured" if measured else "pre_selection",
        "synthetic": True,
        "dimensions": dimensions,
        "score": None,
        "red_lines": red_lines,
        "metrics": metrics,
        "samples_ref": (
            str(samples_path.relative_to(REPO_ROOT)) if samples_path else None
        ),
        "measured_at": None,
        "measured_by": None,
    }

    if args.stdout:
        print(json.dumps(scorecard, ensure_ascii=False, indent=2))
        return 1 if red_lines else 0

    region_dir = SCORECARDS_DIR / args.region
    region_dir.mkdir(parents=True, exist_ok=True)
    out_path = region_dir / f"{args.capability}_{args.region}_{args.vendor}.json"
    with out_path.open("w", encoding="utf-8") as fh:
        json.dump(scorecard, fh, ensure_ascii=False, indent=2)
        fh.write("\n")
    print(f"wrote {out_path.relative_to(REPO_ROOT)}")
    return 1 if red_lines else 0


def cmd_check_config(args: argparse.Namespace) -> int:
    vendors = VENDOR_ENV_MAP.get(args.region, {})
    print(f"region={args.region}")
    missing_any = False
    for vendor in sorted(vendors):
        env_vars = vendors[vendor]
        missing = [name for name in env_vars if not os.environ.get(name)]
        if missing:
            missing_any = True
        label = "missing" if missing else "ready"
        print(f"{vendor}: {label} (missing={', '.join(missing) or 'none'})")
    if args.check and missing_any:
        return 1
    return 0


def _add_subparser(
    subparsers: Any, name: str, handler: Any, help_text: str
) -> None:
    parser = subparsers.add_parser(name, help=help_text)
    parser.set_defaults(handler=handler)
    return parser  # type: ignore[return-value]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    sessions_parser = subparsers.add_parser(
        "sessions", help="list generated synthetic sessions"
    )
    sessions_parser.set_defaults(handler=cmd_sessions)

    percentiles_parser = subparsers.add_parser(
        "percentiles", help="compute percentiles from a samples JSONL file"
    )
    percentiles_parser.add_argument("--samples", required=True)
    percentiles_parser.add_argument(
        "--check", action="store_true", help="exit non-zero on threshold failure"
    )
    percentiles_parser.set_defaults(handler=cmd_percentiles)

    scorecard_parser = subparsers.add_parser(
        "scorecard", help="assemble a scorecard for one capability-region-vendor"
    )
    scorecard_parser.add_argument("--capability", choices=CAPABILITIES, required=True)
    scorecard_parser.add_argument("--region", choices=REGIONS, required=True)
    scorecard_parser.add_argument("--vendor", required=True)
    scorecard_parser.add_argument("--role", choices=ROLES, default="primary")
    scorecard_parser.add_argument(
        "--stdout", action="store_true", help="print the scorecard instead of writing"
    )
    scorecard_parser.set_defaults(handler=cmd_scorecard)

    config_parser = subparsers.add_parser(
        "check-config", help="report missing region-scoped vendor credentials"
    )
    config_parser.add_argument("--region", choices=REGIONS, required=True)
    config_parser.add_argument(
        "--check", action="store_true", help="exit non-zero when credentials are missing"
    )
    config_parser.set_defaults(handler=cmd_check_config)

    args = parser.parse_args()
    return int(args.handler(args))


if __name__ == "__main__":
    sys.exit(main())

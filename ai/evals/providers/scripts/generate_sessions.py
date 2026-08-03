"""Generate synthetic provider-evaluation session material (OD-01 Phase 0).

Reads the golden datasets (ai/evals/datasets/*-core.jsonl) and the synthetic
interview transcript fixture, then emits neutral session scripts for
ASR/TTS/LLM/avatar vendor probes. All output is synthetic and contains no
real personal information.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[4]
DATASETS_DIR = REPO_ROOT / "ai" / "evals" / "datasets"
TRANSCRIPT_PATH = (
    REPO_ROOT / "fixtures" / "synthetic" / "transcripts" / "transcript-zh-01.json"
)
OUT_DIR = REPO_ROOT / "ai" / "evals" / "providers" / "sessions"


def _read_jsonl(path: Path) -> list[dict]:
    rows: list[dict] = []
    with path.open("r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def _extract_core_sessions(dataset: Path, language: str) -> list[dict]:
    sessions: list[dict] = []
    for case in _read_jsonl(dataset):
        case_id = case.get("case_id", "unknown")
        evidence = case.get("input", {}).get("evidence_items", [])
        for index, item in enumerate(evidence):
            question = item.get("question_summary") or ""
            answer = item.get("answer_text") or ""
            if not question and not answer:
                continue
            sessions.append(
                {
                    "session_id": f"prov-{language}-{case_id}-{index:02d}",
                    "synthetic": True,
                    "language": language,
                    "dialect_tag": "mandarin" if language == "zh-CN" else "en-us",
                    "source": str(dataset.relative_to(REPO_ROOT)),
                    "source_case_id": case_id,
                    "turn": {
                        "question_text": question,
                        "expected_answer_text": answer,
                        "answer_status": item.get("answer_status", "answered"),
                        "input_modes_used": item.get("input_modes_used", ["voice"]),
                    },
                    "target_capabilities": ["asr", "tts", "llm", "avatar"],
                }
            )
    return sessions


def _extract_transcript_sessions(path: Path) -> list[dict]:
    with path.open("r", encoding="utf-8") as fh:
        doc = json.load(fh)
    sessions: list[dict] = []
    for turn in doc.get("turns", []):
        question = turn.get("question", {})
        answer = turn.get("answer", {})
        sessions.append(
            {
                "session_id": f"prov-zh-transcript-{turn.get('turn_index', 0):02d}",
                "synthetic": True,
                "language": doc.get("interview_language", "zh-CN"),
                "dialect_tag": "mandarin",
                "source": str(path.relative_to(REPO_ROOT)),
                "source_case_id": doc.get("session_id"),
                "turn": {
                    "question_text": question.get("played_text", ""),
                    "expected_answer_text": answer.get("asr_final_text", ""),
                    "revised_text": answer.get("revised_text"),
                    "interrupted": bool(question.get("interrupted", False)),
                    "revision_required": answer.get("revision_id") is not None,
                },
                "target_capabilities": ["asr", "avatar"],
            }
        )
    return sessions


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="print the plan without writing files",
    )
    args = parser.parse_args()

    zh_sessions = _extract_core_sessions(DATASETS_DIR / "zh-core.jsonl", "zh-CN")
    en_sessions = _extract_core_sessions(DATASETS_DIR / "en-core.jsonl", "en-US")
    transcript_sessions = _extract_transcript_sessions(TRANSCRIPT_PATH)

    grouped = {
        "zh-CN": zh_sessions + transcript_sessions,
        "en-US": en_sessions,
    }
    manifest = {
        "kind": "provider_eval_session_manifest",
        "synthetic": True,
        "generated_by": "scripts/generate_sessions.py",
        "counts": {key: len(value) for key, value in grouped.items()},
        "sources": [
            str(DATASETS_DIR / "zh-core.jsonl"),
            str(DATASETS_DIR / "en-core.jsonl"),
            str(TRANSCRIPT_PATH),
        ],
    }

    if args.dry_run:
        for language, sessions in grouped.items():
            print(f"{language}: {len(sessions)} sessions")
        print(f"output: {OUT_DIR}")
        return 0

    for language, sessions in grouped.items():
        language_dir = OUT_DIR / language
        language_dir.mkdir(parents=True, exist_ok=True)
        stem = "zh" if language == "zh-CN" else "en"
        target = language_dir / f"session-{stem}-providers.jsonl"
        with target.open("w", encoding="utf-8") as fh:
            for session in sessions:
                fh.write(json.dumps(session, ensure_ascii=False) + "\n")

    with (OUT_DIR / "manifest.json").open("w", encoding="utf-8") as fh:
        json.dump(manifest, fh, ensure_ascii=False, indent=2)
        fh.write("\n")

    total = sum(len(value) for value in grouped.values())
    print(f"wrote {total} sessions to {OUT_DIR}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

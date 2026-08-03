"""M2 本地实测（第一轮）：edge-TTS + FunASR + DeepSeek 回合链路。

覆盖可自动化的指标：TTS 合成时延、ASR 回合级时延与文本相似度、LLM 时延。
建连/全链路回应（含数字人）/打断/画质/弱网/长会话需要浏览器端联调，列为后续轮次。
"""

from __future__ import annotations

import argparse
import difflib
import json
import os
import re
import time
import urllib.parse
import urllib.request
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[4]
SESSIONS_PATH = (
    REPO_ROOT / "ai" / "evals" / "providers" / "sessions" / "zh-CN" / "session-zh-providers.jsonl"
)
SAMPLES_DIR = REPO_ROOT / "ai" / "evals" / "providers" / "samples"
REPORTS_DIR = REPO_ROOT / "ai" / "evals" / "providers" / "reports"
WORK_DIR = REPO_ROOT / "work" / "m2"

SELFHOST_URL = os.environ.get("MGD_SELFHOST_URL", "http://127.0.0.1:8000")
LLM_BASE_URL = os.environ.get("LLM_PRIMARY_BASE_URL", "https://api.deepseek.com")
LLM_API_KEY = os.environ.get("LLM_PRIMARY_API_KEY", "")
LLM_MODEL = os.environ.get("LLM_MODEL_PINNED_VERSION", "deepseek-chat")

SYSTEM_PROMPT = "你是面个蛋的数字面试官。请用中文简短、结构化地回答面试问题。"


def percentile(values: list[float], q: float) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    rank = (len(ordered) - 1) * q
    lower = int(rank)
    upper = min(lower + 1, len(ordered) - 1)
    if lower == upper:
        return float(ordered[lower])
    return ordered[lower] * (upper - rank) + ordered[upper] * (rank - lower)


def normalize(text: str) -> str:
    return re.sub(r"[\s，。；：、,.;:()（）“”\"']", "", text).lower()


def similarity(a: str, b: str) -> float:
    return round(difflib.SequenceMatcher(None, normalize(a), normalize(b)).ratio(), 3)


def tts_synthesize(text: str) -> tuple[bytes, float]:
    payload = urllib.parse.urlencode({"text": text}).encode("utf-8")
    request = urllib.request.Request(
        SELFHOST_URL + "/v1/tts/synthesize",
        data=payload,
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        method="POST",
    )
    start = time.perf_counter()
    with urllib.request.urlopen(request, timeout=120) as response:
        data = response.read()
    return data, (time.perf_counter() - start) * 1000


def asr_transcribe(wav: bytes) -> tuple[str, float]:
    boundary = "----mgd-m2-boundary"
    body = (
        f"--{boundary}\r\n"
        'Content-Disposition: form-data; name="file"; filename="answer.wav"\r\n'
        "Content-Type: audio/wav\r\n\r\n"
    ).encode("utf-8") + wav + f"\r\n--{boundary}--\r\n".encode("utf-8")
    request = urllib.request.Request(
        SELFHOST_URL + "/v1/asr/transcribe",
        data=body,
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
        method="POST",
    )
    start = time.perf_counter()
    with urllib.request.urlopen(request, timeout=120) as response:
        result = json.loads(response.read().decode("utf-8"))
    return str(result.get("text", "")), (time.perf_counter() - start) * 1000


def llm_complete(question: str) -> tuple[str, float]:
    payload = {
        "model": LLM_MODEL,
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": question},
        ],
        "temperature": 0.3,
        "max_tokens": 300,
        "stream": False,
    }
    request = urllib.request.Request(
        LLM_BASE_URL.rstrip("/") + "/chat/completions",
        data=json.dumps(payload).encode("utf-8"),
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {LLM_API_KEY}",
        },
        method="POST",
    )
    start = time.perf_counter()
    with urllib.request.urlopen(request, timeout=120) as response:
        body = json.loads(response.read().decode("utf-8"))
    elapsed = (time.perf_counter() - start) * 1000
    text = str(body["choices"][0]["message"]["content"]).strip()
    return text, elapsed


def load_sessions() -> list[dict]:
    rows: list[dict] = []
    with SESSIONS_PATH.open("r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--limit", type=int, default=10)
    parser.add_argument("--llm-cases", type=int, default=3)
    parser.add_argument("--save-audio", action="store_true", default=True)
    args = parser.parse_args()

    sessions = [s for s in load_sessions() if s.get("turn", {}).get("expected_answer_text")]
    sessions = sessions[: args.limit]
    if not sessions:
        print("no sessions")
        return 1

    SAMPLES_DIR.mkdir(parents=True, exist_ok=True)
    REPORTS_DIR.mkdir(parents=True, exist_ok=True)
    audio_dir = WORK_DIR / "audio"
    if args.save_audio:
        audio_dir.mkdir(parents=True, exist_ok=True)

    asr_samples: list[dict] = []
    tts_samples: list[dict] = []
    llm_samples: list[dict] = []
    quality: list[dict] = []
    llm_rows: list[dict] = []

    for index, session in enumerate(sessions):
        turn = session["turn"]
        golden = turn["expected_answer_text"]
        session_id = session["session_id"]
        wav, tts_ms = tts_synthesize(golden)
        if args.save_audio:
            (audio_dir / f"{session_id}.wav").write_bytes(wav)
        text, asr_ms = asr_transcribe(wav)
        sim = similarity(golden, text)
        quality.append(
            {
                "session_id": session_id,
                "tts_ms": round(tts_ms, 1),
                "asr_ms": round(asr_ms, 1),
                "similarity": sim,
                "golden": golden,
                "asr_text": text,
            }
        )
        asr_samples.append(
            {
                "metric": "asr_final",
                "value_ms": round(asr_ms, 1),
                "vendor": "funasr_selfhost",
                "region": "cn",
                "synthetic": True,
                "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            }
        )
        tts_samples.append(
            {
                "metric": "tts_synth",
                "value_ms": round(tts_ms, 1),
                "vendor": "edge_tts",
                "region": "cn",
                "synthetic": True,
                "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            }
        )
        print(f"[{index + 1}/{len(sessions)}] {session_id}: tts={tts_ms:.0f}ms asr={asr_ms:.0f}ms sim={sim}")

    for session in sessions[: args.llm_cases]:
        question = session["turn"].get("question_text", "")
        if not question:
            continue
        text, latency = llm_complete(question)
        llm_rows.append(
            {
                "session_id": session["session_id"],
                "question": question,
                "answer": text[:200],
                "latency_ms": round(latency, 1),
                "ok": bool(text),
            }
        )
        llm_samples.append(
            {
                "metric": "llm",
                "value_ms": round(latency, 1),
                "vendor": "deepseek",
                "region": "cn",
                "synthetic": True,
                "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            }
        )
        print(f"llm {session['session_id']}: {latency:.0f}ms ok={bool(text)}")

    asr_latency = [q["asr_ms"] for q in quality]
    tts_latency = [q["tts_ms"] for q in quality]
    similarities = [q["similarity"] for q in quality]
    llm_latency = [r["latency_ms"] for r in llm_rows]

    report = {
        "kind": "m2_local_round1",
        "synthetic": True,
        "run_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "stack": {"tts": "edge-tts zh-CN-XiaoxiaoNeural", "asr": "funasr iic/SenseVoiceSmall cpu", "llm": "deepseek"},
        "metrics": {
            "asr_final_ms": {
                "n": len(asr_latency),
                "p50": round(percentile(asr_latency, 0.5), 1),
                "p95": round(percentile(asr_latency, 0.95), 1),
            },
            "tts_synth_ms": {
                "n": len(tts_latency),
                "p50": round(percentile(tts_latency, 0.5), 1),
                "p95": round(percentile(tts_latency, 0.95), 1),
            },
            "llm_ms": {
                "n": len(llm_latency),
                "p50": round(percentile(llm_latency, 0.5), 1),
                "p95": round(percentile(llm_latency, 0.95), 1),
            },
            "asr_similarity_avg": round(sum(similarities) / len(similarities), 3) if similarities else 0,
            "asr_similarity_min": round(min(similarities), 3) if similarities else 0,
        },
        "quality": quality,
        "llm": llm_rows,
        "pending_metrics": [
            "connect_p95",
            "e2e_response_p95",
            "interrupt_p95",
            "video_720p_24fps",
            "weak_network",
            "long_session_60min",
            "tts_mos_blind_review",
        ],
    }

    (SAMPLES_DIR / "asr_cn_funasr_selfhost.jsonl").write_text(
        "".join(json.dumps(row, ensure_ascii=False) + "\n" for row in asr_samples),
        encoding="utf-8",
    )
    (SAMPLES_DIR / "tts_cn_edge_tts.jsonl").write_text(
        "".join(json.dumps(row, ensure_ascii=False) + "\n" for row in tts_samples),
        encoding="utf-8",
    )
    (SAMPLES_DIR / "llm_cn_deepseek.jsonl").write_text(
        "".join(json.dumps(row, ensure_ascii=False) + "\n" for row in llm_samples),
        encoding="utf-8",
    )
    report_path = REPORTS_DIR / "m2-local-round1.json"
    report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")

    manifest = {
        "kind": "m2_blind_review_manifest",
        "synthetic": True,
        "audio_dir": str(audio_dir),
        "samples": [
            {"session_id": q["session_id"], "audio": f"{q['session_id']}.wav", "golden": q["golden"]}
            for q in quality
        ],
        "instruction": "人工盲评：只听音频，按 MOS 1-5 给自然度/清晰度打分；评分后回填 ai/evals/providers/reports/m2-local-round1.json 的盲评字段。",
    }
    (WORK_DIR / "review-manifest.json").write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    print(f"report: {report_path}")
    print(f"audio: {audio_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
import argparse
import json
import os


def load_summary(dir_path: str) -> dict:
    path = os.path.join(dir_path, "summary", "summary.json")
    with open(path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def fmt(num, unit="", digits=3):
    if num is None:
        return ""
    try:
        num = float(num)
    except Exception:
        return str(num)
    return f"{num:.{digits}f}{unit}"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--left", required=True, help="RESULTS_DIR of baseline run set")
    parser.add_argument("--right", required=True, help="RESULTS_DIR of compared run set")
    parser.add_argument("--output", required=True, help="Markdown output path")
    parser.add_argument("--left-name", default="baseline")
    parser.add_argument("--right-name", default="candidate")
    args = parser.parse_args()

    left = load_summary(args.left)
    right = load_summary(args.right)

    left_res = left.get("average_resources", {}) or {}
    right_res = right.get("average_resources", {}) or {}

    services = sorted(set(left_res.keys()) | set(right_res.keys()))
    metrics = ["cpu", "memory", "read_bytes", "write_bytes"]

    lines = []
    lines.append(f"# Benchmark comparison: {args.left_name} vs {args.right_name}")
    lines.append("")
    lines.append(f"- left: `{args.left}` (runs={left.get('runs')})")
    lines.append(f"- right: `{args.right}` (runs={right.get('runs')})")
    lines.append("")

    lines.append("## Latency (average percentiles, ms)")
    lines.append("")
    lines.append("| percentile | left | right | delta |")
    lines.append("|---|---:|---:|---:|")
    for p in ["p50", "p75", "p90", "p95", "p99"]:
        lv = (left.get("average_percentiles", {}) or {}).get(p, 0)
        rv = (right.get("average_percentiles", {}) or {}).get(p, 0)
        lines.append(f"| {p} | {fmt(lv)} | {fmt(rv)} | {fmt(rv-lv)} |")

    lines.append("")
    lines.append("## Resources (average of per-run mean values)")
    lines.append("")
    lines.append("| service | metric | left mean | right mean | delta |")
    lines.append("|---|---|---:|---:|---:|")
    for svc in services:
        for m in metrics:
            lv = ((left_res.get(svc, {}) or {}).get(m, {}) or {}).get("mean", 0)
            rv = ((right_res.get(svc, {}) or {}).get(m, {}) or {}).get("mean", 0)
            unit = ""
            if m == "cpu":
                unit = " cores"
            elif m == "memory":
                unit = " bytes"
            elif m in ("read_bytes", "write_bytes"):
                unit = " B/s"
            lines.append(f"| {svc} | {m} | {fmt(lv, unit=unit)} | {fmt(rv, unit=unit)} | {fmt(rv-lv, unit=unit)} |")

    os.makedirs(os.path.dirname(args.output), exist_ok=True)
    with open(args.output, "w", encoding="utf-8") as handle:
        handle.write("\n".join(lines) + "\n")


if __name__ == "__main__":
    main()


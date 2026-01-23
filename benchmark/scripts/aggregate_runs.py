#!/usr/bin/env python3
import argparse
import json
import os

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--results-dir", required=True)
    parser.add_argument("--output-dir", required=True)
    args = parser.parse_args()

    os.makedirs(args.output_dir, exist_ok=True)

    runs = []
    for name in sorted(os.listdir(args.results_dir)):
        run_dir = os.path.join(args.results_dir, name)
        summary_path = os.path.join(run_dir, "summary.json")
        if not os.path.isfile(summary_path):
            continue
        with open(summary_path, "r", encoding="utf-8") as handle:
            summary = json.load(handle)
        runs.append((name, summary))

    if not runs:
        raise SystemExit("No run summaries found")

    percentiles = ["p50", "p75", "p90", "p95", "p99"]
    aggregate = {p: 0.0 for p in percentiles}
    trend = []

    for name, summary in runs:
        p = summary.get("percentiles", {})
        for key in percentiles:
            aggregate[key] += p.get(key, 0)
        trend.append(p.get("p95", 0))

    count = len(runs)
    for key in percentiles:
        aggregate[key] = aggregate[key] / count

    output = {
        "runs": count,
        "average_percentiles": aggregate,
    }

    with open(os.path.join(args.output_dir, "summary.json"), "w", encoding="utf-8") as handle:
        json.dump(output, handle, indent=2)

    plt.figure(figsize=(8, 4))
    plt.plot(range(1, count + 1), trend, marker="o", linewidth=1)
    plt.xlabel("run")
    plt.ylabel("p95 latency (ms)")
    plt.title("p95 latency trend")
    plt.tight_layout()
    plt.savefig(os.path.join(args.output_dir, "p95_trend.png"))
    plt.close()


if __name__ == "__main__":
    main()

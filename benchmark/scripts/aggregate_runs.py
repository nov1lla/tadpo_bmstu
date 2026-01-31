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
    resources_sum = {}
    resources_count = {}

    for name, summary in runs:
        p = summary.get("percentiles", {})
        for key in percentiles:
            aggregate[key] += p.get(key, 0)
        trend.append(p.get("p95", 0))

        resources = summary.get("resources", {}) or {}
        for service, metrics in resources.items():
            resources_sum.setdefault(service, {})
            resources_count[service] = resources_count.get(service, 0) + 1
            for metric_name, stats in (metrics or {}).items():
                resources_sum[service].setdefault(metric_name, {"min": 0.0, "max": 0.0, "mean": 0.0})
                for stat in ["min", "max", "mean"]:
                    resources_sum[service][metric_name][stat] += float((stats or {}).get(stat, 0))

    count = len(runs)
    for key in percentiles:
        aggregate[key] = aggregate[key] / count

    avg_resources = {}
    for service, metrics in resources_sum.items():
        divisor = resources_count.get(service, count) or 1
        avg_resources[service] = {}
        for metric_name, stats in metrics.items():
            avg_resources[service][metric_name] = {
                "min": stats["min"] / divisor,
                "max": stats["max"] / divisor,
                "mean": stats["mean"] / divisor,
            }

    output = {
        "runs": count,
        "average_percentiles": aggregate,
        "average_resources": avg_resources,
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

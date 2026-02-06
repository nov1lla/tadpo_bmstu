#!/usr/bin/env python3
import argparse
import json
import math
import os
from datetime import datetime

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt


def add_nested(dst, keys, value):
    cur = dst
    for k in keys[:-1]:
        cur = cur.setdefault(k, {})
    cur[keys[-1]] = cur.get(keys[-1], 0.0) + value


def div_nested(dst, divisor):
    for key, value in dst.items():
        if isinstance(value, dict):
            div_nested(value, divisor)
        else:
            dst[key] = value / divisor


def parse_time(value):
    if value.endswith("Z"):
        value = value[:-1] + "+00:00"
    if "." in value:
        base, rest = value.split(".", 1)
        if "+" in rest:
            frac, tz = rest.split("+", 1)
            sign = "+"
        elif "-" in rest:
            frac, tz = rest.split("-", 1)
            sign = "-"
        else:
            frac = rest
            tz = "00:00"
            sign = "+"
        frac = (frac[:6]).ljust(6, "0")
        value = f"{base}.{frac}{sign}{tz}"
    return datetime.fromisoformat(value).timestamp()


def load_latency_series_seconds(k6_json_path):
    """
    Returns per-run latency series as list[(relative_second, mean_latency_ms)].
    Computed as mean http_req_duration per wall-clock second, then rebased to 0
    at the first observed second in that run.
    """
    sums = {}
    counts = {}
    with open(k6_json_path, "r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            data = json.loads(line)
            if data.get("type") != "Point":
                continue
            if data.get("metric") != "http_req_duration":
                continue
            payload = data.get("data", {})
            value = payload.get("value")
            ts = payload.get("time")
            if value is None or ts is None:
                continue
            sec = int(parse_time(ts))
            sums[sec] = sums.get(sec, 0.0) + float(value)
            counts[sec] = counts.get(sec, 0) + 1

    if not sums:
        return []

    seconds = sorted(sums.keys())
    start = seconds[0]
    series = []
    for sec in seconds:
        c = counts.get(sec, 0)
        if c <= 0:
            continue
        series.append((sec - start, sums[sec] / c))
    return series


def load_resources_series_seconds(resources_json_path):
    """
    Returns dict[service][metric] -> list[(relative_second, mean_value)].

    The per-run resources.json contains timeseries in absolute epoch seconds.
    We rebase to 0 using the minimum timestamp across all metrics for the service,
    then bucket by second and compute mean for that second.
    """
    with open(resources_json_path, "r", encoding="utf-8") as handle:
        resources = json.load(handle)

    series = resources.get("series", {})
    out = {}
    for service, metrics in series.items():
        # pick a stable start per service (minimum timestamp across all metrics)
        start_ts = None
        for _, points in metrics.items():
            if not points:
                continue
            ts0 = float(points[0][0])
            if start_ts is None or ts0 < start_ts:
                start_ts = ts0
        if start_ts is None:
            continue

        service_out = {}
        for metric, points in metrics.items():
            if not points:
                continue
            sums = {}
            counts = {}
            for ts, value in points:
                sec = int(float(ts) - start_ts)
                sums[sec] = sums.get(sec, 0.0) + float(value)
                counts[sec] = counts.get(sec, 0) + 1
            seconds = sorted(sums.keys())
            metric_series = []
            for sec in seconds:
                c = counts.get(sec, 0)
                if c <= 0:
                    continue
                metric_series.append((sec, sums[sec] / c))
            service_out[metric] = metric_series

        if service_out:
            out[service] = service_out

    return out


def plot_percentiles_bar(percentiles, output_path, title):
    labels = ["p50", "p75", "p90", "p95", "p99"]
    values = [percentiles.get(k, 0.0) for k in labels]
    plt.figure(figsize=(6, 4))
    plt.bar(labels, values)
    plt.ylabel("http_req_duration (ms)")
    plt.title(title)
    plt.tight_layout()
    plt.savefig(output_path)
    plt.close()


def plot_resources_figure(service, metrics_series, output_path, title_suffix):
    fig, axes = plt.subplots(2, 2, figsize=(10, 6))
    fig.suptitle(f"Resources: {service} ({title_suffix})")
    layout = [
        (axes[0][0], "cpu", "CPU (cores)", 1.0),
        (axes[0][1], "memory", "Memory (MB)", 1024 * 1024),
        (axes[1][0], "read_bytes", "Disk read (KB/s)", 1024.0),
        (axes[1][1], "write_bytes", "Disk write (KB/s)", 1024.0),
    ]

    for ax, name, ylabel, scale in layout:
        data = metrics_series.get(name, [])
        if not data:
            ax.set_visible(False)
            continue
        xs = [t for t, _ in data]
        ys = [v / scale for _, v in data]
        ax.plot(xs, ys, linewidth=1)
        ax.set_xlabel("seconds (relative)")
        ax.set_ylabel(ylabel)
        ax.grid(True, linestyle="--", linewidth=0.3)

    plt.tight_layout()
    plt.savefig(output_path)
    plt.close()


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
        runs.append((name, run_dir, summary))

    if not runs:
        raise SystemExit("No run summaries found")

    percentiles = ["p50", "p75", "p90", "p95", "p99"]
    aggregate_percentiles = {p: 0.0 for p in percentiles}
    aggregate_k6_metrics = {}
    aggregate_resources = {}
    aggregate_endpoints = {}

    k6_metric_keys = set()
    for _, _, summary in runs:
        k6_metric_keys.update(summary.get("k6_metrics", {}).keys())
    k6_metric_keys = sorted(k6_metric_keys)

    trends = {
        "percentiles": {p: [] for p in percentiles},
        "k6_metrics": {},
    }

    for _, _, summary in runs:
        p = summary.get("percentiles", {})
        for key in percentiles:
            aggregate_percentiles[key] += p.get(key, 0.0)
            trends["percentiles"][key].append(p.get(key, 0.0))

        k6_metrics = summary.get("k6_metrics", {})
        for key in k6_metric_keys:
            value = float(k6_metrics.get(key, 0.0) or 0.0)
            aggregate_k6_metrics[key] = aggregate_k6_metrics.get(key, 0.0) + value
            trends["k6_metrics"].setdefault(key, []).append(value)

        for service, metrics in summary.get("resources", {}).items():
            for metric, stats in metrics.items():
                for stat_name, stat_value in stats.items():
                    add_nested(aggregate_resources, [service, metric, stat_name], float(stat_value or 0.0))

        for endpoint, stats in summary.get("endpoints", {}).items():
            add_nested(aggregate_endpoints, [endpoint, "count"], float(stats.get("count", 0.0)))
            for key in percentiles:
                add_nested(aggregate_endpoints, [endpoint, key], float(stats.get(key, 0.0)))

    count = len(runs)
    for key in percentiles:
        aggregate_percentiles[key] = aggregate_percentiles[key] / count
    for key in list(aggregate_k6_metrics.keys()):
        aggregate_k6_metrics[key] = aggregate_k6_metrics[key] / count
    div_nested(aggregate_resources, count)
    div_nested(aggregate_endpoints, count)

    output = {
        "runs": count,
        "average_percentiles": aggregate_percentiles,
        "average_k6_metrics": aggregate_k6_metrics,
        "average_resources": aggregate_resources,
        "average_endpoints": aggregate_endpoints,
    }

    with open(os.path.join(args.output_dir, "summary.json"), "w", encoding="utf-8") as handle:
        json.dump(output, handle, indent=2)

    # Trend: percentiles per run.
    plt.figure(figsize=(9, 4))
    xs = list(range(1, count + 1))
    for key in percentiles:
        plt.plot(xs, trends["percentiles"][key], marker="o", linewidth=1, label=key)
    plt.xlabel("run")
    plt.ylabel("latency (ms)")
    plt.title("Latency percentiles trend")
    plt.legend(ncol=len(percentiles), fontsize=8)
    plt.tight_layout()
    plt.savefig(os.path.join(args.output_dir, "latency_percentiles_trend.png"))
    plt.close()

    # Backward-compatible: p95-only trend (older docs/links may reference it).
    plt.figure(figsize=(8, 4))
    plt.plot(xs, trends["percentiles"]["p95"], marker="o", linewidth=1)
    plt.xlabel("run")
    plt.ylabel("p95 latency (ms)")
    plt.title("p95 latency trend")
    plt.tight_layout()
    plt.savefig(os.path.join(args.output_dir, "p95_trend.png"))
    plt.close()

    # Averaged percentiles bar chart across runs (matches per-run latency_percentiles.png).
    plot_percentiles_bar(
        aggregate_percentiles,
        os.path.join(args.output_dir, "latency_percentiles_avg.png"),
        "Latency percentiles (average across runs)",
    )

    # Trend: average latency (k6 avg).
    if "http_req_duration_avg" in trends["k6_metrics"]:
        plt.figure(figsize=(8, 4))
        plt.plot(xs, trends["k6_metrics"]["http_req_duration_avg"], marker="o", linewidth=1)
        plt.xlabel("run")
        plt.ylabel("http_req_duration avg (ms)")
        plt.title("Average latency trend")
        plt.tight_layout()
        plt.savefig(os.path.join(args.output_dir, "latency_avg_trend.png"))
        plt.close()

    # Averaged latency timeseries across runs (mean-per-second, then mean across runs).
    # This matches "latency_timeseries.png" for each run but aggregates by relative second.
    seconds_sum = {}
    seconds_count = {}
    for _, run_dir, _ in runs:
        k6_json = os.path.join(run_dir, "k6.json")
        if not os.path.isfile(k6_json):
            continue
        series = load_latency_series_seconds(k6_json)
        for second, value in series:
            seconds_sum[second] = seconds_sum.get(second, 0.0) + value
            seconds_count[second] = seconds_count.get(second, 0) + 1

    if seconds_sum:
        seconds = sorted(seconds_sum.keys())
        ys = []
        for s in seconds:
            c = seconds_count.get(s, 0)
            ys.append(seconds_sum[s] / c if c else math.nan)

        plt.figure(figsize=(10, 4))
        plt.plot(seconds, ys, linewidth=1)
        plt.xlabel("seconds (relative)")
        plt.ylabel("http_req_duration (ms)")
        plt.title("Latency over time (mean per second, averaged across runs)")
        plt.tight_layout()
        plt.savefig(os.path.join(args.output_dir, "latency_timeseries_avg.png"))
        plt.close()

        with open(os.path.join(args.output_dir, "latency_timeseries_avg.json"), "w", encoding="utf-8") as handle:
            json.dump(
                {
                    "points": [[int(s), float(ys[i])] for i, s in enumerate(seconds) if not math.isnan(ys[i])],
                    "note": "mean http_req_duration per second (per-run), then averaged across runs by relative second",
                },
                handle,
                indent=2,
            )

    # Averaged resources timeseries across runs (per service/metric).
    resources_sum = {}  # service -> metric -> second -> sum
    resources_count = {}  # service -> metric -> second -> count

    for _, run_dir, _ in runs:
        resources_path = os.path.join(run_dir, "resources.json")
        if not os.path.isfile(resources_path):
            continue
        run_series = load_resources_series_seconds(resources_path)
        for service, metrics in run_series.items():
            for metric, series_points in metrics.items():
                for second, value in series_points:
                    resources_sum.setdefault(service, {}).setdefault(metric, {})[second] = (
                        resources_sum.setdefault(service, {}).setdefault(metric, {}).get(second, 0.0) + value
                    )
                    resources_count.setdefault(service, {}).setdefault(metric, {})[second] = (
                        resources_count.setdefault(service, {}).setdefault(metric, {}).get(second, 0) + 1
                    )

    averaged_resources_series = {}
    for service, metrics in resources_sum.items():
        averaged_resources_series[service] = {}
        for metric, by_second in metrics.items():
            seconds = sorted(by_second.keys())
            out_series = []
            for sec in seconds:
                c = resources_count.get(service, {}).get(metric, {}).get(sec, 0)
                if c <= 0:
                    continue
                out_series.append((sec, by_second[sec] / c))
            averaged_resources_series[service][metric] = out_series

        # Plot figure matching per-run resources_<service>.png.
        plot_resources_figure(
            service,
            averaged_resources_series[service],
            os.path.join(args.output_dir, f"resources_{service}_avg.png"),
            "average across runs",
        )

        # Also dump json for the averaged service series.
        out_json = {
            "service": service,
            "metrics": {
                metric: [[int(t), float(v)] for t, v in points]
                for metric, points in averaged_resources_series[service].items()
            },
            "note": "mean value per second (per-run), then averaged across runs by relative second (rebased per service)",
        }
        with open(os.path.join(args.output_dir, f"resources_{service}_avg.json"), "w", encoding="utf-8") as handle:
            json.dump(out_json, handle, indent=2)


if __name__ == "__main__":
    main()

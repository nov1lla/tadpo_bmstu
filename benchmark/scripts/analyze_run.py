#!/usr/bin/env python3
import argparse
import json
import math
import os
from datetime import datetime

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt


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


def percentile(values, p):
    if not values:
        return 0
    values = sorted(values)
    k = (len(values) - 1) * p
    f = math.floor(k)
    c = math.ceil(k)
    if f == c:
        return values[int(k)]
    d0 = values[int(f)] * (c - k)
    d1 = values[int(c)] * (k - f)
    return d0 + d1


def bucket_by_second(points):
    buckets = {}
    for ts, value in points:
        key = int(ts)
        buckets.setdefault(key, []).append(value)
    items = sorted(buckets.items())
    return [(t, sum(vals) / len(vals)) for t, vals in items]


def load_k6_points(path):
    points = []
    endpoints = {}
    with open(path, "r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            data = json.loads(line)
            if data.get("type") != "Point":
                continue
            if data.get("metric") != "http_req_duration":
                continue
            value = data["data"]["value"]
            ts = parse_time(data["data"]["time"])
            tags = data["data"].get("tags", {})
            endpoint = tags.get("endpoint", "unknown")
            points.append((ts, value))
            endpoints.setdefault(endpoint, []).append(value)
    return points, endpoints


def plot_timeseries(points, output_path):
    if not points:
        return
    series = bucket_by_second(points)
    start = series[0][0]
    xs = [t - start for t, _ in series]
    ys = [v for _, v in series]
    plt.figure(figsize=(10, 4))
    plt.plot(xs, ys, linewidth=1)
    plt.xlabel("seconds")
    plt.ylabel("http_req_duration (ms)")
    plt.title("Latency over time (mean per second)")
    plt.tight_layout()
    plt.savefig(output_path)
    plt.close()


def plot_histogram(values, output_path):
    if not values:
        return
    plt.figure(figsize=(6, 4))
    plt.hist(values, bins=50)
    plt.xlabel("http_req_duration (ms)")
    plt.ylabel("requests")
    plt.title("Latency histogram")
    plt.tight_layout()
    plt.savefig(output_path)
    plt.close()


def plot_percentiles(values, output_path):
    if not values:
        return
    percentiles = [0.5, 0.75, 0.9, 0.95, 0.99]
    labels = ["p50", "p75", "p90", "p95", "p99"]
    vals = [percentile(values, p) for p in percentiles]
    plt.figure(figsize=(6, 4))
    plt.bar(labels, vals)
    plt.ylabel("http_req_duration (ms)")
    plt.title("Latency percentiles")
    plt.tight_layout()
    plt.savefig(output_path)
    plt.close()


def plot_resources(resources, output_dir):
    series = resources.get("series", {})
    for service, metrics in series.items():
        fig, axes = plt.subplots(2, 2, figsize=(10, 6))
        fig.suptitle(f"Resources: {service}")
        for ax, name, ylabel, scale in [
            (axes[0][0], "cpu", "CPU (cores)", 1.0),
            (axes[0][1], "memory", "Memory (MB)", 1024 * 1024),
            (axes[1][0], "read_bytes", "Disk read (KB/s)", 1024.0),
            (axes[1][1], "write_bytes", "Disk write (KB/s)", 1024.0),
        ]:
            data = metrics.get(name, [])
            if not data:
                ax.set_visible(False)
                continue
            start = data[0][0]
            xs = [t - start for t, _ in data]
            ys = [v / scale for _, v in data]
            ax.plot(xs, ys, linewidth=1)
            ax.set_xlabel("seconds")
            ax.set_ylabel(ylabel)
            ax.grid(True, linestyle="--", linewidth=0.3)
        plt.tight_layout()
        output_path = os.path.join(output_dir, f"resources_{service}.png")
        plt.savefig(output_path)
        plt.close()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--k6-json", required=True)
    parser.add_argument("--k6-summary", required=True)
    parser.add_argument("--resources", required=True)
    parser.add_argument("--output-dir", required=True)
    args = parser.parse_args()

    os.makedirs(args.output_dir, exist_ok=True)

    points, endpoints = load_k6_points(args.k6_json)
    values = [v for _, v in points]

    with open(args.k6_summary, "r", encoding="utf-8") as handle:
        k6_summary = json.load(handle)

    with open(args.resources, "r", encoding="utf-8") as handle:
        resources = json.load(handle)

    percentiles = {
        "p50": percentile(values, 0.5),
        "p75": percentile(values, 0.75),
        "p90": percentile(values, 0.9),
        "p95": percentile(values, 0.95),
        "p99": percentile(values, 0.99),
    }

    endpoint_stats = {}
    for endpoint, vals in endpoints.items():
        endpoint_stats[endpoint] = {
            "count": len(vals),
            "p50": percentile(vals, 0.5),
            "p75": percentile(vals, 0.75),
            "p90": percentile(vals, 0.9),
            "p95": percentile(vals, 0.95),
            "p99": percentile(vals, 0.99),
        }

    metrics = k6_summary.get("metrics", {})
    k6_metrics = {
        "http_reqs_rate": metrics.get("http_reqs", {}).get("rate", 0),
        "http_req_failed_rate": metrics.get("http_req_failed", {}).get("rate", 0),
        "http_req_duration_avg": metrics.get("http_req_duration", {}).get("avg", 0),
    }

    summary = {
        "request_count": len(values),
        "k6_metrics": k6_metrics,
        "percentiles": percentiles,
        "endpoints": endpoint_stats,
        "resources": resources.get("summary", {}),
    }

    with open(os.path.join(args.output_dir, "summary.json"), "w", encoding="utf-8") as handle:
        json.dump(summary, handle, indent=2)

    plot_timeseries(points, os.path.join(args.output_dir, "latency_timeseries.png"))
    plot_histogram(values, os.path.join(args.output_dir, "latency_histogram.png"))
    plot_percentiles(values, os.path.join(args.output_dir, "latency_percentiles.png"))
    plot_resources(resources, args.output_dir)


if __name__ == "__main__":
    main()

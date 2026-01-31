#!/usr/bin/env python3
import argparse
import json
import statistics
import urllib.parse
import urllib.request


def prom_query_range(base_url, query, start, end, step):
    params = urllib.parse.urlencode({
        "query": query,
        "start": start,
        "end": end,
        "step": step,
    })
    url = f"{base_url.rstrip('/')}/api/v1/query_range?{params}"
    with urllib.request.urlopen(url) as resp:
        data = json.load(resp)
    if data.get("status") != "success":
        raise RuntimeError(f"Prometheus query failed: {data}")
    return data["data"]["result"]


def to_series(result):
    if not result:
        return []
    values = result[0].get("values", [])
    return [[float(ts), float(val)] for ts, val in values]


def series_summary(series):
    if not series:
        return {"min": 0, "max": 0, "mean": 0}
    vals = [v for _, v in series]
    return {
        "min": min(vals),
        "max": max(vals),
        "mean": statistics.fmean(vals),
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prom-url", required=True)
    parser.add_argument("--project", default="")
    parser.add_argument("--start", required=True)
    parser.add_argument("--end", required=True)
    parser.add_argument("--step", default="1")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    # Extendable list: benchmark + CI stacks can include extra monitoring components.
    # Missing services just result in empty series (min/max/mean = 0).
    services = ["webapp", "postgres", "otel-collector", "prometheus", "cadvisor", "test-runner"]
    def selector(service: str) -> str:
        labels = [f'container_label_com_docker_compose_service=\"{service}\"']
        if args.project:
            labels.append(f'container_label_com_docker_compose_project=\"{args.project}\"')
        return "{" + ",".join(labels) + "}"

    metrics = {
        "cpu": "rate(container_cpu_usage_seconds_total%s[30s])",
        "memory": "container_memory_usage_bytes%s",
        "read_bytes": "rate(container_fs_reads_bytes_total%s[30s])",
        "write_bytes": "rate(container_fs_writes_bytes_total%s[30s])",
    }

    output = {
        "metadata": {
            "project": args.project,
            "start": float(args.start),
            "end": float(args.end),
            "step": float(args.step),
        },
        "series": {},
        "summary": {},
    }

    for service in services:
        output["series"][service] = {}
        output["summary"][service] = {}
        for name, template in metrics.items():
            query = template % selector(service)
            result = prom_query_range(args.prom_url, query, args.start, args.end, args.step)
            series = to_series(result)
            output["series"][service][name] = series
            output["summary"][service][name] = series_summary(series)

    with open(args.output, "w", encoding="utf-8") as handle:
        json.dump(output, handle, indent=2)


if __name__ == "__main__":
    main()

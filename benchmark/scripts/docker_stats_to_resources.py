#!/usr/bin/env python3
import argparse
import json
import re
import statistics


UNITS = {
    "b": 1,
    "kb": 1000,
    "mb": 1000**2,
    "gb": 1000**3,
    "tb": 1000**4,
    "kib": 1024,
    "mib": 1024**2,
    "gib": 1024**3,
    "tib": 1024**4,
}


def parse_bytes(text):
    value = text.strip().replace(",", ".")
    m = re.match(r"^\s*([0-9]*\.?[0-9]+)\s*([a-zA-Z]+)?\s*$", value)
    if not m:
        return 0.0
    num = float(m.group(1))
    unit = (m.group(2) or "B").lower()
    mul = UNITS.get(unit, 1)
    return num * mul


def parse_cpu_cores(cpu_text):
    cpu_text = cpu_text.strip().replace("%", "").replace(",", ".")
    if not cpu_text:
        return 0.0
    try:
        return float(cpu_text) / 100.0
    except ValueError:
        return 0.0


def series_summary(series):
    if not series:
        return {"min": 0, "max": 0, "mean": 0}
    vals = [v for _, v in series]
    return {"min": min(vals), "max": max(vals), "mean": statistics.fmean(vals)}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--project", required=True)
    parser.add_argument("--start", required=True, type=float)
    parser.add_argument("--end", required=True, type=float)
    parser.add_argument("--step", default=1.0, type=float)
    parser.add_argument("--service", action="append", required=True, help="name:container_id")
    args = parser.parse_args()

    service_by_id = {}
    for item in args.service:
        name, cid = item.split(":", 1)
        service_by_id[cid] = name

    raw = {name: [] for name in service_by_id.values()}
    with open(args.input, "r", encoding="utf-8") as handle:
        for line in handle:
            parts = line.rstrip("\n").split("\t")
            if len(parts) < 5:
                continue
            ts_s, cid, cpu_s, mem_s, io_s = parts[:5]
            service = service_by_id.get(cid)
            if not service:
                continue
            try:
                ts = float(ts_s)
            except ValueError:
                continue
            mem_cur = parse_bytes(mem_s.split("/", 1)[0])
            io_parts = [p.strip() for p in io_s.split("/", 1)]
            read_total = parse_bytes(io_parts[0]) if io_parts else 0.0
            write_total = parse_bytes(io_parts[1]) if len(io_parts) > 1 else 0.0
            raw[service].append({
                "ts": ts,
                "cpu": parse_cpu_cores(cpu_s),
                "memory": mem_cur,
                "read_total": read_total,
                "write_total": write_total,
            })

    output = {
        "metadata": {
            "project": args.project,
            "start": args.start,
            "end": args.end,
            "step": args.step,
            "source": "docker_stats",
        },
        "series": {},
        "summary": {},
    }

    for service, points in raw.items():
        points.sort(key=lambda p: p["ts"])
        cpu = [[p["ts"], p["cpu"]] for p in points]
        memory = [[p["ts"], p["memory"]] for p in points]
        read_rate = []
        write_rate = []
        for i in range(1, len(points)):
            dt = points[i]["ts"] - points[i - 1]["ts"]
            if dt <= 0:
                continue
            ts = points[i]["ts"]
            read_rate.append([ts, max(0.0, (points[i]["read_total"] - points[i - 1]["read_total"]) / dt)])
            write_rate.append([ts, max(0.0, (points[i]["write_total"] - points[i - 1]["write_total"]) / dt)])

        output["series"][service] = {
            "cpu": cpu,
            "memory": memory,
            "read_bytes": read_rate,
            "write_bytes": write_rate,
        }
        output["summary"][service] = {
            "cpu": series_summary(cpu),
            "memory": series_summary(memory),
            "read_bytes": series_summary(read_rate),
            "write_bytes": series_summary(write_rate),
        }

    with open(args.output, "w", encoding="utf-8") as handle:
        json.dump(output, handle, indent=2)


if __name__ == "__main__":
    main()

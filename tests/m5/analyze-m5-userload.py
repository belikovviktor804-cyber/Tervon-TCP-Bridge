#!/usr/bin/env python3
"""Analyze saved M5 user-load JSONL, including incomplete observations.

Only reads allowlisted telemetry fields and process snapshots. An incomplete
record is never converted into an acceptance PASS or synthesized summary.
"""
import argparse
import json
from pathlib import Path

def read_events(path: Path):
    events = []
    broken = 0
    for line in path.read_text(encoding="utf-8").splitlines():
        try:
            value = json.loads(line)
            if isinstance(value, dict):
                events.append(value)
            else:
                broken += 1
        except ValueError:
            broken += 1
    return events, broken

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("report", type=Path, help="existing telemetry.jsonl")
    args = parser.parse_args()
    if not args.report.is_file():
        print("M5_RECOVERY=NO_FILE")
        return 2
    events, broken = read_events(args.report)
    starts = [x for x in events if x.get("kind") == "start"]
    samples = [x for x in events if x.get("kind") == "sample"]
    summaries = [x for x in events if x.get("kind") == "summary"]
    target = starts[-1].get("duration_seconds") if starts else None
    interval = starts[-1].get("interval_seconds") if starts else None
    expected = target // interval if isinstance(target, int) and isinstance(interval, int) and interval > 0 else None

    print("M5_RECOVERY_REPORT=" + str(args.report))
    print("M5_RECOVERY_SAMPLES=" + str(len(samples)))
    print("M5_RECOVERY_EXPECTED=" + str(expected if expected is not None else "UNKNOWN"))
    print("M5_RECOVERY_INVALID_LINES=" + str(broken))
    seen_steps = [e.get("step") for e in samples]
    contiguous = seen_steps == list(range(1, len(samples) + 1))
    print("M5_RECOVERY_SEQUENCE=" + ("CONTIGUOUS" if contiguous else "INCOMPLETE_SEQUENCE"))

    for role in ("homeserver", "finserver"):
        total = {"padding": 0, "jitter": 0, "errors": 0}
        peak = 0
        for event in samples:
            node = event.get("nodes", {}).get(role, {})
            if not isinstance(node, dict):
                print("M5_RECOVERY=" + role.upper() + "_MISSING_NODE")
                return 1
            for key, field in (("padding", "padding_frames_delta"), ("jitter", "jitter_frames_delta"), ("errors", "error_delta")):
                val = node.get(field)
                if type(val) is not int or val < 0:
                    print("M5_RECOVERY=" + role.upper() + "_INVALID_COUNTER")
                    return 1
                total[key] += val
            current = node.get("sessions_active")
            if type(current) is not int or current < 0:
                return 1
            peak = max(peak, current)
        tag = "M5_RECOVERY_" + role.upper()
        print(tag + "_PAD_FRAMES_OBSERVED=" + str(total["padding"]))
        print(tag + "_JITTER_FRAMES_OBSERVED=" + str(total["jitter"]))
        print(tag + "_ERROR_DELTA_OBSERVED=" + str(total["errors"]))
        print(tag + "_PEAK_ACTIVE_SESSIONS_SAMPLED=" + str(peak))

    process = args.report.with_name("process.txt")
    if process.is_file():
        values = []
        for line in process.read_text(encoding="utf-8").splitlines():
            try:
                rss, cpu = line.split()
                values.append((int(rss), float(cpu)))
            except ValueError:
                pass
        if values:
            print("M5_RECOVERY_HOMESERVER_RSS_PEAK_KIB=" + str(max(v[0] for v in values)))
            print("M5_RECOVERY_PROCESS_READINGS=" + str(len(values)))
            print("M5_RECOVERY_CPU_LIFETIME_LAST_PERCENT=" + str(values[-1][1]))

    if summaries:
        summary = summaries[-1]
        print("M5_RECOVERY_ORIGINAL_SUMMARY_STATUS=" + str(summary.get("status", "UNKNOWN")))
    if not contiguous or broken:
        print("M5_RECOVERY=INCOMPLETE_OR_CORRUPT_RECORD")
    elif not summaries or expected is None or len(samples) != expected:
        print("M5_RECOVERY=PARTIAL_EVIDENCE_ONLY")
        print("M5_RECOVERY_COLLECTOR_STOP_REASON=UNKNOWN")
    else:
        print("M5_RECOVERY=COMPLETE_RECORD_AVAILABLE")
    print("M5_RECOVERY_LIVE_RTT_AND_PER_STREAM_FAIRNESS=NOT_MEASURED")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())

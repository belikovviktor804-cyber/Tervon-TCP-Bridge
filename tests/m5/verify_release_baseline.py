#!/usr/bin/env python3
"""M5.1: read-only published v1.0.0 artifact provenance check.

No production configuration, processes, secrets or services are accessed.
Results describe artifact identity, not performance or a new release PASS.
"""
import argparse
import hashlib
import json
import pathlib
import urllib.error
import urllib.request

API = "https://api.github.com/repos/tervon-software/Tervon-TCP-Bridge/releases/tags/v1.0.0"
EXPECTED_LEGACY = "5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2"
EXPECTED_ASSET = "tervon-bridge-linux-amd64"


def sha256_path(path):
    h = hashlib.sha256()
    with open(path, "rb") as fp:
        for chunk in iter(lambda: fp.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def select_binary_asset(meta):
    if not isinstance(meta, dict) or meta.get("tag_name") != "v1.0.0":
        raise ValueError("release_tag_mismatch")
    assets = meta.get("assets")
    if not isinstance(assets, list):
        raise ValueError("release_assets_missing")
    matches = [a for a in assets if isinstance(a, dict) and a.get("name") == EXPECTED_ASSET]
    if len(matches) != 1:
        raise ValueError("binary_asset_count_not_one")
    item = matches[0]
    if not isinstance(item.get("size"), int) or item["size"] <= 0:
        raise ValueError("binary_asset_size_invalid")
    # GitHub reports public release assets on github.com or objects.githubusercontent.com;
    # never follow arbitrary links from untrusted metadata.
    url = item.get("browser_download_url")
    if not isinstance(url, str) or not url.startswith(
        "https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.0/"
    ):
        raise ValueError("binary_asset_url_invalid")
    return item


def build_report(meta, path):
    asset = select_binary_asset(meta)
    actual = sha256_path(path)
    size = pathlib.Path(path).stat().st_size
    return {
        "status": "PASS_ARTIFACT_MATCHES_HISTORICAL" if
            size == asset["size"] and actual == EXPECTED_LEGACY
            else "ARTIFACT_DIFFERS_FROM_HISTORICAL",
        "release_tag": "v1.0.0",
        "asset_name": EXPECTED_ASSET,
        "asset_size_matches": size == asset["size"],
        "sha256": actual,
        "historical_sha256": EXPECTED_LEGACY,
        "historical_sha_matches": actual == EXPECTED_LEGACY,
        "note": "no_claim_of_reproducible_build_or_benchmark",
    }


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--release-json", type=pathlib.Path,
                   help="offline saved public release metadata; no network required")
    p.add_argument("--artifact", type=pathlib.Path, required=True,
                   help="local downloaded release asset; never a running binary path")
    args = p.parse_args()
    if not args.artifact.is_file():
        p.error("artifact_file_missing")
    try:
        if args.release_json:
            meta = json.loads(args.release_json.read_text(encoding="utf-8"))
        else:
            req = urllib.request.Request(API, headers={
                "Accept": "application/vnd.github+json", "User-Agent": "tervon-m5-baseline",
            })
            with urllib.request.urlopen(req, timeout=15) as response:
                meta = json.load(response)
        report = build_report(meta, args.artifact)
        print(json.dumps(report, sort_keys=True))
        return 0 if report["status"] == "PASS_ARTIFACT_MATCHES_HISTORICAL" else 2
    except (ValueError, OSError, urllib.error.URLError, json.JSONDecodeError):
        print(json.dumps({"status": "BASELINE_VERIFICATION_BLOCKED",
                          "reason": "metadata_or_artifact_invalid"}))
        return 3


if __name__ == "__main__":
    raise SystemExit(main())

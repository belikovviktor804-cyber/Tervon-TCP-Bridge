#!/usr/bin/env python3
"""Add Bridge navigation link to existing monitor templates only.

Dry run by default; --apply makes minimal replacements with a backup.
Does not modify nginx or restart any service. No template content is printed.
"""
from __future__ import annotations

import argparse
from pathlib import Path
import re
import shutil
import sys

SITE_TEMPLATES = Path("/opt/xray-blocker/web-dashboard/templates")
NAV = re.compile(r'<nav\b[^>]*\bclass\s*=\s*["\'][^"\']*\bsite-nav\b[^"\']*["\'][^>]*>.*?</nav\s*>',
                 re.IGNORECASE | re.DOTALL)
LINK = '<a href="/bridge/">TCP Bridge</a>'


def updated(text: str) -> tuple[str, str]:
    matches = list(NAV.finditer(text))
    if len(matches) != 1:
        return text, "BLOCKED_NO_UNIQUE_SITE_NAV"
    nav = matches[0].group()
    if re.search(r'href\s*=\s*["\']/bridge/?["\']', nav, re.IGNORECASE):
        return text, "ALREADY_PRESENT"
    new_nav = re.sub(r'</nav\s*>', LINK + '\n</nav>', nav,
                     count=1, flags=re.IGNORECASE)
    return text[:matches[0].start()] + new_nav + text[matches[0].end():], "READY"


def update_file(path: Path, apply: bool) -> str:
    if path.is_symlink() or not path.is_file():
        return "BLOCKED_NOT_REGULAR"
    text = path.read_text(encoding="utf-8")
    revised, result = updated(text)
    if result != "READY" or not apply:
        return result
    backup = path.with_name(path.name + ".before-bridge-nav")
    if backup.exists() or backup.is_symlink():
        return "BLOCKED_BACKUP_EXISTS"
    shutil.copy2(path, backup)
    # Preserve ownership and mode: write in place only after backup.
    try:
        with path.open("w", encoding="utf-8") as f:
            f.write(revised)
    except Exception:
        shutil.copy2(backup, path)
        raise
    return "APPLIED"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    if not SITE_TEMPLATES.is_dir() or SITE_TEMPLATES.is_symlink():
        print("BLOCKED: template directory is missing or unsafe")
        return 2
    targets = sorted(SITE_TEMPLATES.glob("*.html"))
    if not targets:
        print("BLOCKED: no HTML templates found")
        return 2
    outcomes = [(file, update_file(file, False)) for file in targets]
    for file, state in outcomes:
        print(f"{file.name}: {state}")
    if any(state.startswith("BLOCKED") for _, state in outcomes):
        print("NAV_RESULT=BLOCKED: inspect unmatched templates before any modification")
        return 2
    if not args.apply:
        print("NAV_RESULT=DRY_RUN")
        return 0
    for file, state in outcomes:
        if state == "READY":
            result = update_file(file, True)
            print(f"{file.name}: {result}")
            if result != "APPLIED":
                print("NAV_RESULT=PARTIAL: manual review of backups required")
                return 1
    print("NAV_RESULT=APPLIED")
    return 0


if __name__ == "__main__":
    sys.exit(main())

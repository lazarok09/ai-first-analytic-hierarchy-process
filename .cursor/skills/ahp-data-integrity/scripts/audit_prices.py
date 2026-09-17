#!/usr/bin/env python3
"""Audit AHP workspace price attributes and alt:<criterion> pairwise direction.

Exit codes:
  0 = pass (no blockers; warns allowed)
  1 = blockers found
  2 = usage / I/O error
"""

from __future__ import annotations

import argparse
import csv
import json
import re
import sys
from pathlib import Path


# Match foreign FX / US listings. Do NOT treat Brazilian "R$123" as a dollar sign.
FX_SMELL = re.compile(
    r"(usd|\beur\b|\bus\$|(?<![Rr])\$\s*\d|converted|convers[aã]o|"
    r"amazon\.com(?!\.br)|sweetwater|thomann|"
    r"≈\s*(?<![Rr])\$|~\s*(?<![Rr])\$)",
    re.I,
)
MARKETPLACE_3P = re.compile(
    r"(marketplace|3p|terceiro|seller\b|fulfilled by (?!amazon\.com\.br))",
    re.I,
)


def read_csv(path: Path) -> list[dict[str, str]]:
    if not path.is_file():
        return []
    with path.open(newline="", encoding="utf-8") as f:
        return list(csv.DictReader(f))


def parse_float(raw: str) -> float | None:
    s = (raw or "").strip().replace(",", "")
    if not s:
        return None
    try:
        return float(s)
    except ValueError:
        return None


def parse_saaty(raw: str) -> float | None:
    s = (raw or "").strip()
    if not s:
        return None
    if "/" in s:
        a, b = s.split("/", 1)
        try:
            return float(a) / float(b)
        except ValueError:
            return None
    try:
        return float(s)
    except ValueError:
        return None


def prefer_winner(left_v: float, right_v: float, prefer: str) -> str:
    """Return 'left', 'right', or 'tie' for who should win on this criterion."""
    if abs(left_v - right_v) < 1e-9:
        return "tie"
    if prefer == "lower":
        return "left" if left_v < right_v else "right"
    return "left" if left_v > right_v else "right"


def pair_winner(value: float) -> str:
    """Saaty value is left relative to right; >1 means left preferred."""
    if abs(value - 1.0) < 1e-9:
        return "tie"
    return "left" if value > 1.0 else "right"


def audit(ws: Path, criterion: str, unit: str, vmin: float | None, vmax: float | None, prefer: str):
    findings: list[dict] = []
    attrs = read_csv(ws / "data" / "attributes.csv")
    pairs = read_csv(ws / "data" / "pairwise.csv")
    alts = {r["id"] for r in read_csv(ws / "data" / "alternatives.csv") if r.get("id")}

    prices: dict[str, float] = {}
    for row in attrs:
        if row.get("criterion_id") != criterion:
            continue
        alt = row.get("alternative_id", "")
        val_raw = row.get("value", "")
        u = (row.get("unit") or "").strip()
        source = (row.get("source") or "").strip()
        note = (row.get("note") or "").strip()
        blob = f"{source} {note} {val_raw}"

        if alt not in alts:
            findings.append(
                {
                    "severity": "blocker",
                    "code": "unknown_alt",
                    "message": f"attribute {alt}/{criterion}: alternative not in alternatives.csv",
                }
            )
            continue

        if not u:
            findings.append(
                {
                    "severity": "blocker",
                    "code": "missing_unit",
                    "message": f"{alt}/{criterion}: missing unit (expected {unit})",
                }
            )
        elif unit and u.upper() != unit.upper():
            findings.append(
                {
                    "severity": "blocker",
                    "code": "unit_mismatch",
                    "message": f"{alt}/{criterion}: unit {u!r} != expected {unit!r}",
                }
            )

        if not source:
            findings.append(
                {
                    "severity": "blocker",
                    "code": "missing_source",
                    "message": f"{alt}/{criterion}: missing source (name a local shop/page)",
                }
            )

        if FX_SMELL.search(blob):
            findings.append(
                {
                    "severity": "blocker",
                    "code": "fx_or_foreign_source",
                    "message": f"{alt}/{criterion}: foreign/FX smell in source/note — use local street price only",
                }
            )

        if MARKETPLACE_3P.search(blob):
            findings.append(
                {
                    "severity": "warn",
                    "code": "marketplace_3p",
                    "message": f"{alt}/{criterion}: marketplace/3P wording — prefer specialty/official local quote",
                }
            )

        num = parse_float(val_raw)
        if num is None:
            findings.append(
                {
                    "severity": "blocker",
                    "code": "non_numeric",
                    "message": f"{alt}/{criterion}: value {val_raw!r} is not numeric",
                }
            )
            continue

        prices[alt] = num
        if vmin is not None and num < vmin:
            findings.append(
                {
                    "severity": "blocker",
                    "code": "below_budget",
                    "message": f"{alt}: price {num} < min {vmin} — remove from shortlist or widen band",
                }
            )
        if vmax is not None and num > vmax:
            findings.append(
                {
                    "severity": "blocker",
                    "code": "above_budget",
                    "message": f"{alt}: price {num} > max {vmax} — remove from shortlist (do not rank)",
                }
            )

    missing_price = sorted(alts - set(prices))
    for alt in missing_price:
        findings.append(
            {
                "severity": "blocker",
                "code": "missing_price_attr",
                "message": f"{alt}: no numeric attribute for criterion {criterion!r}",
            }
        )

    matrix = f"alt:{criterion}"
    for row in pairs:
        if row.get("matrix") != matrix:
            continue
        left, right = row.get("left", ""), row.get("right", "")
        if left not in prices or right not in prices:
            continue
        sv = parse_saaty(row.get("value", ""))
        if sv is None:
            findings.append(
                {
                    "severity": "blocker",
                    "code": "bad_saaty",
                    "message": f"{matrix} {left}/{right}: bad Saaty value {row.get('value')!r}",
                }
            )
            continue
        expect = prefer_winner(prices[left], prices[right], prefer)
        got = pair_winner(sv)
        if expect != "tie" and got != "tie" and expect != got:
            findings.append(
                {
                    "severity": "blocker",
                    "code": "pairwise_contradicts_attrs",
                    "message": (
                        f"{matrix} {left}/{right}={row.get('value')}: "
                        f"pair prefers {got}, attributes prefer {expect} "
                        f"({prefer}: {left}={prices[left]}, {right}={prices[right]})"
                    ),
                }
            )
        elif expect != "tie" and got == "tie":
            findings.append(
                {
                    "severity": "warn",
                    "code": "pairwise_flat_vs_attrs",
                    "message": (
                        f"{matrix} {left}/{right}=1 but attributes differ "
                        f"({left}={prices[left]}, {right}={prices[right]})"
                    ),
                }
            )

    blockers = [f for f in findings if f["severity"] == "blocker"]
    warns = [f for f in findings if f["severity"] == "warn"]
    return {
        "workspace": str(ws),
        "criterion": criterion,
        "unit": unit,
        "min": vmin,
        "max": vmax,
        "prefer": prefer,
        "prices": prices,
        "ok": len(blockers) == 0,
        "blocker_count": len(blockers),
        "warn_count": len(warns),
        "findings": findings,
    }


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--workspace", "-w", required=True)
    p.add_argument("--criterion", default="value")
    p.add_argument("--unit", default="BRL")
    p.add_argument("--min", type=float, default=None)
    p.add_argument("--max", type=float, default=None)
    p.add_argument("--prefer", choices=("lower", "higher"), default="lower")
    p.add_argument("--json", action="store_true")
    args = p.parse_args()

    ws = Path(args.workspace).resolve()
    if not (ws / "ahp.toml").is_file() and not (ws / "data").is_dir():
        print(f"error: not an AHP workspace: {ws}", file=sys.stderr)
        return 2

    report = audit(ws, args.criterion, args.unit, args.min, args.max, args.prefer)

    if args.json:
        json.dump(report, sys.stdout, indent=2, ensure_ascii=False)
        print()
    else:
        print(f"ahp-data-integrity  {ws}")
        print(
            f"criterion={report['criterion']} unit={report['unit']} "
            f"min={report['min']} max={report['max']} prefer={report['prefer']}"
        )
        print(f"prices: {report['prices']}")
        if not report["findings"]:
            print("ok: no findings")
        for f in report["findings"]:
            print(f"{f['severity']}: [{f['code']}] {f['message']}")
        print(
            f"summary: ok={report['ok']} blockers={report['blocker_count']} warns={report['warn_count']}"
        )

    return 0 if report["ok"] else 1


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""Build the lib_catalog.area CSV from the pinned bps2020 boundaries.

Source: HDX COD-AB Indonesia (BPS via OCHA), valid from 2020-04-01, licence CC BY-IGO 3.0.
The extracted GeoJSON files idn_admin1.geojson, idn_admin2.geojson and idn_admin3.geojson
must come from the archive with the pinned SHA-256 below; pass the archive with --zip to
have it checked.

Rules (the same as the admar boundary build, decision D6):
  - Codes drop the "ID" prefix: provinsi 2 digits, kab/kota 4, kecamatan 7.
  - Placeholders (forest, lake, unnamed island) are dropped: kab/kota whose code ends in
    88 or 99, kecamatan whose code ends in 000, 555, 888 or 999.
  - Names are the BPS names as published, with no case change.

Expected output: 34 provinsi, 514 kabkota, 7031 kecamatan.

Usage:
  build_area_csv.py --source DIR [--zip ARCHIVE] --out area.csv
"""
import argparse
import csv
import hashlib
import json
import sys
from pathlib import Path

PINNED_SHA256 = "d285a40f9fc4b726b1bda98ee833d52d23d94fe78b6a47c0a769d2e912ff0970"
VINTAGE = "bps2020"
EXPECTED = {"provinsi": 34, "kabkota": 514, "kecamatan": 7031}

LEVELS = (
    # file, level, code property, parent code property, code length, placeholder suffixes
    ("idn_admin1.geojson", "provinsi", "adm1_pcode", None, 2, ()),
    ("idn_admin2.geojson", "kabkota", "adm2_pcode", "adm1_pcode", 4, ("88", "99")),
    ("idn_admin3.geojson", "kecamatan", "adm3_pcode", "adm2_pcode", 7, ("000", "555", "888", "999")),
)


def sha256_of(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def strip_prefix(code: str, length: int) -> str:
    if not code.startswith("ID"):
        raise ValueError(f"unexpected code without ID prefix: {code!r}")
    kode = code[2:]
    if len(kode) != length or not kode.isdigit():
        raise ValueError(f"unexpected code {code!r}, want {length} digits after ID")
    return kode


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawTextHelpFormatter)
    ap.add_argument("--source", required=True, type=Path, help="directory with idn_admin{1,2,3}.geojson")
    ap.add_argument("--zip", type=Path, help="the downloaded archive, checked against the pinned SHA-256")
    ap.add_argument("--out", required=True, type=Path)
    args = ap.parse_args()

    if args.zip is not None:
        got = sha256_of(args.zip)
        if got != PINNED_SHA256:
            print(f"archive SHA-256 mismatch: {got} != {PINNED_SHA256}", file=sys.stderr)
            return 1

    rows = []
    for fname, level, code_prop, parent_prop, length, placeholders in LEVELS:
        with (args.source / fname).open(encoding="utf-8") as f:
            features = json.load(f)["features"]
        for feat in features:
            props = feat["properties"]
            kode = strip_prefix(props[code_prop], length)
            if placeholders and kode.endswith(placeholders):
                continue
            nama = props[code_prop.replace("pcode", "name")]
            parent = strip_prefix(props[parent_prop], length - (2 if level == "kabkota" else 3)) if parent_prop else ""
            rows.append((kode, level, nama.strip(), parent, VINTAGE))

    counts = {level: sum(1 for r in rows if r[1] == level) for level in EXPECTED}
    if counts != EXPECTED:
        print(f"unexpected counts {counts}, want {EXPECTED}", file=sys.stderr)
        return 1
    if len({r[0] for r in rows}) != len(rows):
        print("duplicate codes", file=sys.stderr)
        return 1

    order = {"provinsi": 0, "kabkota": 1, "kecamatan": 2}
    rows.sort(key=lambda r: (order[r[1]], r[0]))
    with args.out.open("w", encoding="utf-8", newline="") as f:
        w = csv.writer(f)
        w.writerow(("kode", "level", "nama", "parent_kode", "boundary_vintage"))
        w.writerows(rows)
    print(f"wrote {len(rows)} rows to {args.out}: {counts}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

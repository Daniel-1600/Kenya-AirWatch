"""Build a reproducible Dandora methane case study from public Sentinel-5P COGs.

Requires rasterio (which installs numpy). Example:
    uv run --with rasterio python scripts/build_case_study.py
"""

from __future__ import annotations

import argparse
import concurrent.futures
import hashlib
import json
import math
import statistics
import tempfile
import time
import urllib.parse
import urllib.request
from datetime import date, datetime, timezone
from pathlib import Path

import numpy as np
import rasterio
from rasterio.warp import transform as project

STAC_SEARCH = "https://explorer.digitalearth.africa/stac/search"
COLLECTION = "s5p_tropomi_l2_ch4"
SITE = {"name": "Dandora Dumpsite", "latitude": -1.2467, "longitude": 36.9068, "radius_km": 4.5}


def fetch_json(url: str) -> dict:
    request = urllib.request.Request(url, headers={"User-Agent": "Dandora-AirWatch/1.0"})
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            with urllib.request.urlopen(request, timeout=120) as response:
                return json.load(response)
        except Exception as exc:
            last_error = exc
            if attempt < 2:
                time.sleep(2 ** attempt)
    raise RuntimeError(f"catalog request failed after three attempts: {last_error}")


def stac_items(start: date, end: date) -> list[dict]:
    radius_degrees = SITE["radius_km"] / 100
    bbox = [
        SITE["longitude"] - radius_degrees,
        SITE["latitude"] - radius_degrees,
        SITE["longitude"] + radius_degrees,
        SITE["latitude"] + radius_degrees,
    ]
    params = urllib.parse.urlencode({
        "collections": COLLECTION,
        "bbox": ",".join(str(value) for value in bbox),
        "datetime": f"{start.isoformat()}T00:00:00Z/{end.isoformat()}T00:00:00Z",
        "limit": 100,
    })
    url = f"{STAC_SEARCH}?{params}"
    items: list[dict] = []
    while url:
        page = fetch_json(url)
        items.extend(page.get("features", []))
        next_link = next((link for link in page.get("links", []) if link.get("rel") == "next"), None)
        url = next_link.get("href") if next_link else ""
    return items


def public_https(s3_url: str) -> str:
    parsed = urllib.parse.urlparse(s3_url)
    return f"https://{parsed.netloc}.s3.af-south-1.amazonaws.com/{parsed.path.lstrip('/')}"


def download(url: str, target: Path) -> None:
    request = urllib.request.Request(url, headers={"User-Agent": "Dandora-AirWatch/1.0"})
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            with urllib.request.urlopen(request, timeout=90) as response, target.open("wb") as output:
                while chunk := response.read(1024 * 1024):
                    output.write(chunk)
            return
        except Exception as exc:
            last_error = exc
            target.unlink(missing_ok=True)
            if attempt < 2:
                time.sleep(2 ** attempt)
    raise RuntimeError(f"asset download failed after three attempts: {last_error}")


def region_mean(ch4_path: Path) -> tuple[float, int, float, float]:
    with rasterio.open(ch4_path) as src:
        values = src.read(1, masked=True)
        centre_x, centre_y = project("EPSG:4326", src.crs, [SITE["longitude"]], [SITE["latitude"]])
        rows, cols = np.indices(values.shape)
        xs = src.transform.c + (cols + 0.5) * src.transform.a + (rows + 0.5) * src.transform.b
        ys = src.transform.f + (cols + 0.5) * src.transform.d + (rows + 0.5) * src.transform.e
        inside = ((xs - centre_x[0]) ** 2 + (ys - centre_y[0]) ** 2) <= (SITE["radius_km"] * 1000) ** 2
        # The CH4 COG uses NaN for missing retrievals in addition to its nodata
        # metadata, so finite-value filtering is the authoritative validity test.
        valid = inside & ~np.ma.getmaskarray(values)
        selected = values.data[valid]
        selected = selected[np.isfinite(selected) & (selected > 0)]
        if selected.size == 0:
            raise ValueError("no valid pixels in the analysis region")
        return float(selected.mean()), int(selected.size), float(selected.min()), float(selected.max())


def score_observations(observations: list[dict]) -> dict:
    scored = []
    previous: list[float] = []
    for observation in observations:
        value = observation["value"]
        if len(previous) >= 10:
            baseline = statistics.fmean(previous)
            standard_deviation = statistics.pstdev(previous)
            z_score = (value - baseline) / standard_deviation if standard_deviation else 0.0
            scored.append({
                "observed_at": observation["observed_at"],
                "value": value,
                "baseline": round(baseline, 2),
                "difference_percent": round((value - baseline) / baseline * 100, 2),
                "z_score": round(z_score, 2),
            })
        previous.append(value)
    if not scored:
        raise ValueError("at least 11 valid observations are required")
    # AirWatch's follow-up workflow is concerned with high-side methane
    # signals. Low-side outliers remain in the series but are not alerts.
    return max(scored, key=lambda item: item["z_score"])


def build(start: date, end: date, output: Path) -> None:
    items = sorted(stac_items(start, end), key=lambda item: item["properties"]["datetime"])
    observations = []
    failures = []
    with tempfile.TemporaryDirectory(prefix="dandora-case-study-") as temp_dir:
        temp = Path(temp_dir)

        def process(entry: tuple[int, dict]) -> tuple[dict | None, dict | None]:
            index, item = entry
            observed_at = item["properties"].get("start_datetime") or item["properties"]["datetime"]
            ch4_url = public_https(item["assets"]["CH4"]["href"])
            ch4_path = temp / f"{index}-ch4.tif"
            try:
                download(ch4_url, ch4_path)
                mean, valid_pixels, minimum, maximum = region_mean(ch4_path)
            except Exception as exc:
                return None, {"item_id": item["id"], "reason": str(exc)}
            return {
                "observed_at": observed_at,
                "value": round(mean, 2),
                "unit": "ppb",
                "valid_pixels": valid_pixels,
                "pixel_min": round(minimum, 2),
                "pixel_max": round(maximum, 2),
                "stac_item_id": item["id"],
                "asset_url": ch4_url,
                "source": f"Digital Earth Africa Sentinel-5P/TROPOMI · STAC {item['id']}",
            }, None

        with concurrent.futures.ThreadPoolExecutor(max_workers=6) as executor:
            for observation, failure in executor.map(process, enumerate(items)):
                if observation:
                    observations.append(observation)
                if failure:
                    failures.append(failure)

    observations.sort(key=lambda item: item["observed_at"])

    print(f"catalog matched {len(items)} items; extracted {len(observations)} valid regional observations; skipped {len(failures)}")
    if failures:
        reasons: dict[str, int] = {}
        for failure in failures:
            reasons[failure["reason"]] = reasons.get(failure["reason"], 0) + 1
        for reason, count in sorted(reasons.items(), key=lambda item: item[1], reverse=True)[:5]:
            print(f"  skipped {count}: {reason}")
    highlighted = score_observations(observations)
    payload = {
        "schema_version": 1,
        "title": "Dandora regional methane observation case study",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "site": SITE,
        "period": {"from": start.isoformat(), "to": end.isoformat()},
        "method": {
            "collection": COLLECTION,
            "catalog": STAC_SEARCH,
            "spatial_summary": "Arithmetic mean of valid CH4 pixels whose centres fall within the analysis radius.",
            "highlight_rule": "Largest positive z-score after at least 10 earlier valid observations; population standard deviation.",
            "scientific_boundary": "Regional column-averaged methane observations do not identify or attribute a point source.",
        },
        "retrieval": {"matched_items": len(items), "valid_observations": len(observations), "skipped_items": failures},
        "highlighted_observation": highlighted,
        "observations": observations,
    }
    fingerprint = {key: payload[key] for key in ("schema_version", "site", "period", "method", "observations")}
    canonical = json.dumps(fingerprint, sort_keys=True, separators=(",", ":"))
    payload["sha256"] = hashlib.sha256(canonical.encode()).hexdigest()
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"wrote {len(observations)} authentic observations to {output}")
    print(f"highlighted {highlighted['observed_at']}: {highlighted['value']:.2f} ppb, z={highlighted['z_score']:.2f}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--from", dest="start", type=date.fromisoformat, default=date(2025, 1, 1))
    parser.add_argument("--to", dest="end", type=date.fromisoformat, default=date(2026, 9, 15))
    parser.add_argument("--output", type=Path, default=Path("data/dandora_case_study.json"))
    args = parser.parse_args()
    build(args.start, args.end, args.output)


if __name__ == "__main__":
    main()

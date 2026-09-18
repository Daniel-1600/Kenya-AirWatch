# Dandora methane case study

This case study demonstrates the complete AirWatch evidence path with authentic public satellite observations. It is a regional screening result, not a claim that Dandora Dumpsite emitted the observed methane.

## Result

The extractor searched 563 Sentinel-5P/TROPOMI methane scenes from 1 January 2025 through 15 September 2026. Thirty-one scenes contained at least one valid methane retrieval inside the 4.5 km analysis radius, a coverage rate of 5.5%.

The strongest high-side signal after establishing a baseline of at least ten earlier valid observations occurred on 24 February 2025:

- Regional mean: 1,944.07 ppb
- Historical mean: 1,924.39 ppb
- Difference: +1.02%
- Z-score: 2.11 (`high`)
- Valid pixels: 35
- Pixel range: 1,942.07–1,945.96 ppb
- STAC item: `5c910eca-74b1-5169-a76f-96823516c923`

This is evidence for follow-up review. Weather, atmospheric transport, retrieval conditions, and other nearby methane sources can affect the observation.

## Source and method

The source is the [Digital Earth Africa Sentinel-5P TROPOMI Level-2 methane collection](https://registry.opendata.aws/deafrica-sentinel5p/), published as public Cloud Optimized GeoTIFFs with STAC metadata. The CH₄ band contains column-averaged dry-air methane in parts per billion.

For each catalog scene, the extractor averages finite CH₄ pixels whose centres fall inside the 4.5 km circle centred at `-1.2467, 36.9068`. It preserves the source STAC identifier, public asset URL, pixel count, pixel range, and observation time. The highlighted event is the largest positive z-score after ten earlier valid observations, using the population standard deviation.

The complete generated evidence is in [`data/dandora_case_study.json`](../data/dandora_case_study.json). Its scientific-data SHA-256 fingerprint is:

```text
b3a1729e662117c6f1100e34a809e0e85b36c9a1e06441a4ac6cba2f97d33b67
```

## Reproduce it

From the repository root:

```bash
UV_CACHE_DIR=/tmp/dandora-uv-cache \
  uv run --with rasterio python scripts/build_case_study.py
```

The script queries the public STAC API, downloads the matching public rasters, recalculates the regional observations, selects the highlighted high-side signal, and rewrites the evidence JSON. Network availability and upstream catalog revisions can affect future outputs; the preserved STAC IDs and asset URLs make any differences auditable.

## Limitations

Only 31 of 563 matching scenes contained valid pixels in this small equatorial analysis region. The series is therefore sparse and cannot provide continuous monitoring. A flagged observation cannot identify a point source, establish causation, estimate landfill emissions, or replace ground instruments and local investigation.

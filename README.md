# Dandora AirWatch

Satellite Environmental Monitoring for Dandora Dumpsite — Nairobi, Kenya.

## Problem

Dandora is one of Nairobi's major waste sites. Continuous ground-based environmental monitoring can be expensive and limited.

## Solution

Dandora AirWatch uses freely available Earth-observation data from Sentinel-5P/TROPOMI to visualize atmospheric methane observations around the dumpsite and detect unusual observations.

> **Scientific boundary:** the dashboard measures atmospheric concentrations around Dandora. It does not prove that Dandora itself emitted a specific amount of methane.

## Architecture

```text
Sentinel-5P / TROPOMI
        ↓
Google Earth Engine
        ↓
Go ingestion adapter
        ↓
PostgreSQL (processed observations only)
        ↓
Go REST API + anomaly service
        ↓
React + TypeScript dashboard
```

## Technology

- Go `net/http`, PostgreSQL, SQL migrations
- React + TypeScript + Vite
- Tailwind CSS, Leaflet, Recharts
- Sentinel-5P collection `COPERNICUS/S5P/OFFL/L3_CH4`
- Band `CH4_column_volume_mixing_ratio_dry_air`

## Local setup

### Demo mode (no credentials required)

```bash
cp .env.example .env
docker compose up -d postgres
cd backend && go run ./cmd/server
# in another terminal:
cd frontend && npm install && npm run dev
```

The demo starts with a clearly marked development fixture so the full dashboard can be judged locally. Run `go run ./cmd/ingest` to load the fixture into PostgreSQL.

### Real Earth Engine mode

Install and authenticate the Earth Engine Python API, then set `SATELLITE_MODE=earthengine` and `EE_PROJECT` in `.env`:

```bash
earthengine authenticate
cd backend && SATELLITE_MODE=earthengine go run ./cmd/ingest
```

The adapter is intentionally isolated in `backend/internal/services/satellite`. It invokes `scripts/earthengine_ingest.py`, which requests Sentinel-5P methane imagery, filters to the Dandora analysis region/date window, removes null values, and stores the published column-averaged dry-air mixing ratio in ppb. The Earth Engine L3 collection is already filtered during ingestion using the product's validity threshold. No raster files are stored in PostgreSQL. If Earth Engine credentials/network are unavailable, ingestion exits with a clear error; only `fixture` mode is used for the demo.

The frontend uses `/api` by default and Vite proxies it to `http://localhost:8080` during local development. If the deployed API is on a separate origin, set `VITE_API_URL` at frontend build time (for example, `https://api.example.com/api`).

## Deploy to Render

The repository includes a [`render.yaml`](render.yaml) Blueprint that creates:

- a static frontend on Render's global CDN;
- a paid 512 MB Go API in Frankfurt;
- a paid PostgreSQL 16 database with 1 GB of storage in Frankfurt.

The API runs the fixture ingestion once after its first successful deploy, so the dashboard has demo data immediately. The baseline is approximately $13.30/month at the prices current in September 2026 ($7 API, $6 database compute, and about $0.30 for 1 GB of database storage), excluding usage beyond Render's included allowances.

1. Push this repository to GitHub.
2. In the Render Dashboard, select **New → Blueprint** and connect the repository.
3. Confirm that Render found `render.yaml`, then apply the Blueprint.
4. When both services are live, open `https://dandora-airwatch.onrender.com`.

The Blueprint keeps PostgreSQL off the public internet and uses its private connection string. It deploys `SATELLITE_MODE=fixture`; Earth Engine requires separate credentials and is intentionally not enabled by this deployment.

## API

- `GET /api/sites`
- `GET /api/sites/:id`
- `GET /api/sites/:id/observations?pollutant=CH4&from=2026-07-01&to=2026-09-15`
- `GET /api/sites/:id/latest`
- `GET /api/sites/:id/anomalies`
- `GET /api/sites/:id/summary`

## Methodology

For each valid observation, the anomaly service compares the value with previous valid observations. It calculates a historical mean, population standard deviation and z-score after at least ten earlier valid observations establish a baseline. High-side values with `z < 1` are normal, `1–2` elevated, `2–3` high, and `>= 3` severe. Low-side outliers remain visible in the series but do not trigger methane-risk alerts.

The dashboard's map shows the monitoring-site coordinate and analysis-radius ring. It does not invent point-level methane locations or present a basemap as satellite measurement imagery.

## Reproducible case study

The dashboard includes an authentic Dandora case study built from the public Digital Earth Africa Sentinel-5P/TROPOMI Level-2 methane collection. The committed evidence contains every valid regional observation, its source STAC identifier and raster URL, the coverage calculation, and a stable scientific-data fingerprint. Read [the case-study methodology and findings](docs/CASE_STUDY.md), or regenerate it with:

```bash
UV_CACHE_DIR=/tmp/dandora-uv-cache uv run --with rasterio python scripts/build_case_study.py
```

The case study is kept separate from fixture-mode operational data and is visibly labeled as authentic satellite evidence in the interface.

## User validation

The [user-validation protocol](docs/USER_VALIDATION.md) defines two short sessions for a local environmental practitioner and an environmental-data user. It tests scientific comprehension, completion of the alert-to-investigation workflow, trust, and practical decision value. Use the [session note template](docs/validation/SESSION_TEMPLATE.md) during each interview and record only completed evidence in the [validation summary](docs/validation/SUMMARY.md).

**Current status:** the protocol and evidence templates are ready; participant sessions have not yet been conducted. The project does not claim user validation until real sessions are documented.

## Limitations

Sentinel-5P has relatively coarse spatial resolution. This system identifies atmospheric observations around Dandora but cannot automatically establish that the landfill itself caused an observed pollution anomaly. Weather, atmospheric transport and nearby emission sources can influence readings. Treat this as an environmental monitoring/early-warning tool, not a regulatory emissions measurement system.

## Screenshots

Run the frontend locally and capture the dashboard here before submission. The repository intentionally does not claim screenshots that have not been captured.

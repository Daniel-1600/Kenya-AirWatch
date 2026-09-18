"""Authenticated Earth Engine extractor. Emits processed JSON, never imagery."""
import json, os
import ee

ee.Initialize(project=os.environ["EE_PROJECT"])
lat, lon = float(os.environ["EE_LAT"]), float(os.environ["EE_LON"])
region = ee.Geometry.Point([lon, lat]).buffer(float(os.environ["EE_RADIUS_KM"]) * 1000)
band = "CH4_column_volume_mixing_ratio_dry_air"
collection = (ee.ImageCollection("COPERNICUS/S5P/OFFL/L3_CH4")
    .filterDate(os.environ["EE_FROM"], os.environ["EE_TO"])
    .filterBounds(region))

def extract(image):
    value = image.select(band).reduceRegion(
        reducer=ee.Reducer.mean(), geometry=region, scale=7000, bestEffort=True
    ).get(band)
    return ee.Feature(None, {"date": image.date().format("YYYY-MM-dd'T'HH:mm:ss'Z'"), "value": value})

features = collection.map(extract).filter(ee.Filter.notNull(["value"])).getInfo()["features"]
print(json.dumps([{"observed_at": f["properties"]["date"],
    # Earth Engine publishes this band as a column-averaged dry-air mixing
    # ratio in ppb; no mol-fraction conversion is required.
    "value": f["properties"]["value"], "unit": "ppb",
    "source": "Sentinel-5P/TROPOMI via Google Earth Engine"} for f in features]))

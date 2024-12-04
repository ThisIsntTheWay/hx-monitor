# Acquire latest airspace GeoJSON but only retain important info
import json
import urllib.request
from datetime import datetime, timedelta
from pathlib import Path

airspaces_json_url = "https://airspace.shv-fsvl.ch/api/v1/geojson/airspaces"
airspaces_json = Path("./public/shv_airspaces.json")
airspaces_json_processed = Path("./public/shv_airspaces_processed.json")
relevant_areas = ["meiringen"]

def download_airspaces_json(force=False):
    if not airspaces_json.is_file() or force:
        print(f"Will download airspaces JSON from '{airspaces_json_url}'")
        urllib.request.urlretrieve(airspaces_json_url, airspaces_json)
    else:
        print("Not downloading airspaces JSON!")

def process_airspaces_json():
    print("Processing airspaces JSON...")
    with open(airspaces_json, 'r') as file:
        data = json.load(file)
        
    if 'features' not in data:
        raise ValueError("The GeoJSON data should have a 'features' key containing the features list.")
        
    features = data['features']

    # Filter entries where 'properties.Name' matches any of the candidates
    filtered_features = [
        f for f in features
        if f.get('properties', {}).get('HX', False) 
        and any(c in f.get('properties', {}).get('Name', '').lower() for c in relevant_areas)
    ]
            
    with open(airspaces_json_processed, 'w', encoding='utf-8') as outfile:
        json.dump(filtered_features, outfile, indent=2)

# ------------------
def main():
    print(f"Verifying '{airspaces_json}'...")
    if airspaces_json.is_file():
        last_modified = datetime.fromtimestamp(airspaces_json.stat().st_mtime)
        time_diff = datetime.now() - last_modified
        
        if time_diff > timedelta(weeks=1):
            download_airspaces_json(True)
        else:
            print(f"JSON still within acceptable timedelta: {timedelta(weeks=1) - time_diff}")
            return
    else:
        download_airspaces_json()

    process_airspaces_json()

if __name__ == "__main__":
    main()

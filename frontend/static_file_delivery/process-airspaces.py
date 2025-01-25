# Acquire latest airspace GeoJSON but only retain important info
import json
import urllib.request
import argparse
from datetime import datetime, timedelta
from pathlib import Path

airspaces_json_url = "https://airspace.shv-fsvl.ch/api/v1/geojson/airspaces"
airspaces_json = Path("./shv_airspaces.json")
airspaces_json_processed = Path("./shv_airspaces_processed.json")
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
    filtered_features = []
    for f in features:
        properties = f.get('properties', {})
        if not properties.get('HX', False):
            continue
        for c in relevant_areas:
            name = properties.get('Name', '').lower()
            if c in name:
                print(f"> Match for '{c}': {name}")
                break
        else:
            continue
        
        filtered_features.append(f)
            
    with open(airspaces_json_processed, 'w', encoding='utf-8') as outfile:
        json.dump(filtered_features, outfile, indent=2)

# ------------------
def main(force=False):    
    print(f"Verifying '{airspaces_json}'...")
    print(f"Relevant areas: {relevant_areas}")
    
    if airspaces_json.is_file() and not force:
        last_modified = datetime.fromtimestamp(airspaces_json.stat().st_mtime)
        time_diff = datetime.now() - last_modified
        
        if time_diff > timedelta(weeks=1):
            download_airspaces_json(True)
        else:
            print(f"JSON still within acceptable timedelta: {timedelta(weeks=1) - time_diff}")
            return
    else:
        download_airspaces_json(True)

    process_airspaces_json()
    print("Done processing")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Force download and processing JSON")
    parser.add_argument("--force", action="store_true", help="Force download and processing of JSON")
    args = parser.parse_args()
    
    main(args.force)

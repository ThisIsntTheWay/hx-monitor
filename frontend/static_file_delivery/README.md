# Static file delivery
As an alternative to just linking the airspaces GeoJSON diretly, this component allows to download the JSON file and trimming it to only serve the stuff we're interested in.  
The `AIRPSACES_JSON_URL` env variable allows the frontend app to use a different airspaces JSON file.  

## Usage
1. First, obtain airspaces JSON by running `process-airspaces.py`.
2. Upload the file to some static file delivery service, or serve the file yourself.

If serving it yourself:
```bash
docker run -d -p 8080:80 -v $(pwd)/shv_airspaces_processed.json:/usr/share/nginx/html/shv_airspaces_processed.json nginx:alpine
```
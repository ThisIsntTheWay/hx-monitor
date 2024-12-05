# Frontend
Frontend component of HX monitor.

## Testing
```bash
# API located at ../api-export
export REACT_APP_API_BASE_URL=http://localhost:8080
npm start
```

## SHV Airspaces
A helper script `process-airspaces.py` will automatically download and filter the SHV-hosted GeoJSON by relevant HX areas.  
As the frontend expects the JSON to be available at `(public)/files/shv_airspaces_processed.json`, the script - intended to be run as a docker container - must mount `./public/files` at `./files` in the container.  

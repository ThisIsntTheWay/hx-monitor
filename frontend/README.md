# Frontend
Frontend component of HX monitor.

## Building
```bash
npx eslint . && npm run build
```

## Testing
```bash
# API located at ../api-export
export REACT_APP_API_BASE_URL=http://localhost:8080
export REACT_APP_AIRPSACES_JSON_URL=https://airspace.shv-fsvl.ch/api/v1/geojson/airspaces # Optional
export REACT_APP_PRE_FILTER_GEO_JSON=true # Optional - Disable if supplying a pre filtered GeoJSON yourself
# Also see `static_file_delivery` for further info on this

npm start
```

## SHV Airspaces
By default, airspace data is directly obtained from SHV's API.  
However, a trimmed-down version of the JSON can be generated and used instead.  
For further infomation, see the folder `static_file_delivery`.
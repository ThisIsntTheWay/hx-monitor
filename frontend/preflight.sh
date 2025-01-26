#!/bin/sh
set -e

CONFIG_FILE=/usr/share/nginx/html/config.js

log_preflight() {
  echo "preflight: $1"
}

log_preflight "Preparing config.js..."

if [ "$REACT_APP_API_BASE_URL" ]; then
  log_preflight "Got REACT_APP_API_BASE_URL: ${REACT_APP_API_BASE_URL}"
  sed -i "s|API_BASE_URL: '',|API_BASE_URL: '$REACT_APP_API_BASE_URL',|g" "$CONFIG_FILE"
else
  log_preflight "Error: REACT_APP_API_BASE_URL must be defined"
  exit 1
fi

if [ "$REACT_APP_AIRPSACES_JSON_URL" ]; then
  log_preflight "Got REACT_APP_AIRPSACES_JSON_URL: ${REACT_APP_AIRPSACES_JSON_URL}"
  sed -i "s|AIRPSACES_JSON_URL: '',|AIRPSACES_JSON_URL: '$REACT_APP_AIRPSACES_JSON_URL',|g" "$CONFIG_FILE"
fi
if [ "$REACT_APP_PRE_FILTER_GEO_JSON" ]; then
  log_preflight "Got REACT_APP_PRE_FILTER_GEO_JSON: ${REACT_APP_PRE_FILTER_GEO_JSON}"
  sed -i "s/\(PRE_FILTER_GEO_JSON:\s*\)[^,]*/\1$REACT_APP_PRE_FILTER_GEO_JSON/" $CONFIG_FILE
fi

log_preflight "Done"
#!/bin/sh
set -e

echo "preflight: Preparing config.js..."

if [ -z "$REACT_APP_API_BASE_URL" ]; then
  echo "Error: REACT_APP_API_BASE_URL must be defined"
  exit 1
fi
echo "Got REACT_APP_API_BASE_URL: ${REACT_APP_API_BASE_URL}"

if [ "$REACT_APP_AIRPSACES_JSON_URL" ]; then
    echo "Got REACT_APP_AIRPSACES_JSON_URL: ${REACT_APP_AIRPSACES_JSON_URL}"
fi

CONFIG_FILE=/usr/share/nginx/html/config.js
sed -i "s|API_BASE_URL: '',|API_BASE_URL: '$REACT_APP_API_BASE_URL',|g" "$CONFIG_FILE"
sed -i "s|AIRPSACES_JSON_URL: '',|AIRPSACES_JSON_URL: '$REACT_APP_AIRPSACES_JSON_URL',|g" "$CONFIG_FILE"
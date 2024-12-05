import React, { useEffect, useState } from 'react';
import './App.css';
import { MapContainer, TileLayer, GeoJSON } from 'react-leaflet';
import { LatLngTuple } from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { fetchApiData, ApiResponse, GetStylingForFeature } from './utils/fetchApiData';

// Define the center coordinates for Interlaken, Switzerland
const INTERLAKEN_COORDS: LatLngTuple = [46.6863, 7.8632]; // Latitude, Longitude

const App: React.FC = () => {
  const [geoJsonData, setGeoJsonData] = useState<any>(null);
  const [apiData, setApiData] = useState<ApiResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isFetching, setFetching] = useState<boolean>(false);
  const [fade, setFade] = useState(false);
  
  const doApiFetch = () => {
    console.info("FETCHING API...")

    setFetching(true)
    fetchApiData('/api/v1/areas')
      .then(setApiData)
      .catch((err) => setError(err.message));
    setFetching(false)
    
    if (apiData) {
      setFade(false)
    }
  }

  // Fetch GeoJSON data
  useEffect(() => {
    fetch('/shv_airspaces_processed.json')
      .then(response => response.json())
      .then(data => setGeoJsonData(data))
      .catch(error => console.error('Error loading GeoJSON:', error));
  }, []);

  // Fetch data from the REST API
  useEffect(doApiFetch, []);
  
  return (
    <div className="App">
      {(!apiData || error) && <div className="gray-overlay"></div>}
      <div className={`center-box ${fade ? 'fade-out' : ''}`}>
        {!apiData ? (
          !error && <p>Fetching from API ⏰</p>
        ) : (
          <pre>{JSON.stringify(apiData, null, 0)}</pre>
        )}

        {/* ToDo: If retrying, show the "Fetching from..." thing again */}
        {!isFetching && error && (
          <div>
            <p>Error fetching API: <span style={{ color: 'red' }}>{error}</span></p>
            <button onClick={doApiFetch}>Retry</button>
          </div>
        )}
      </div>
      
      <MapContainer center={INTERLAKEN_COORDS} zoom={13} style={{ height: '100vh', width: '100%' }}>
        <TileLayer url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" />
        
        {/* Render GeoJSON data */}
        {apiData && geoJsonData && (
          <GeoJSON
            data={geoJsonData}
            style={(feature) => ({
              color: GetStylingForFeature(feature, apiData).Color,
              weight: 3,
              opacity: GetStylingForFeature(feature, apiData).Opacity,
            })}
            onEachFeature={(feature, layer) => {
              if (feature.properties) {
                layer.bindPopup(
                  `<strong>${feature.properties.Name || 'Unnamed Feature'}</strong>`
                );
              }
            }}
          />
        )}
      </MapContainer>
    </div>
  );
}

export default App;

import React, { useEffect, useState } from 'react';
import './App.css';
import { MapContainer, TileLayer, GeoJSON } from 'react-leaflet';
import { LatLngTuple, } from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { ApiResponseArea, fetchApiAreas, getStylingForFeature } from './utils/fetchApiData';
import InfoBox from './components/InfoBox';

const INTERLAKEN_COORDS: LatLngTuple = [46.6863, 7.8632]; // Lat, Lon

const App: React.FC = () => {
  const [geoJsonData, setGeoJsonData] = useState<any>(null);
  const [apiAreaData, setApiAreaData] = useState<ApiResponseArea | null>(null);
  const [infoBoxVisibility, setInfoBoxVisibility] = useState<boolean>(false);
  const [featureState, setFeatureState] = useState<any>(null);
  const [geoJsonError, setGeoJsonError] = useState<string | null>(null);
  const [isFetching, setFetching] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const toggleInfoBoxVisibility = () => {
    setInfoBoxVisibility(prevVisibility => !prevVisibility);
  };
  
  const apiFetchAreas = () => {
    setError(null);
    setFetching(true);

    fetchApiAreas()
      .then(setApiAreaData)
      .catch((err) => setError(err.message))
      .finally(() => setFetching(false));
  };

  // Get GeoJSON data
  useEffect(() => {
    fetch('/shv_airspaces_processed.json')
      .then(response => response.json())
      .then(data => setGeoJsonData(data))
      .catch(err => setGeoJsonError(err.message));
  }, []);

  useEffect(apiFetchAreas, []);
  
  return (
    <div className="App">
      {(!apiAreaData || error || geoJsonError) && <div className="gray-overlay"></div>}

      <div
        id="fetch-info-box"
        className={`box ${(error || geoJsonError) ? 'error-box' : ''}`}
        hidden={apiAreaData === null || geoJsonError !== null ? false : true}
      >
        <h3>
          <div>
            <h1>
              {!apiAreaData && (!error && !geoJsonError) ? (
                <span className="clock-spinner"></span>
              ) : (
                "❌"
              )}
            </h1>
            <p>
              {(!geoJsonError && (!apiAreaData && !error)) && "Fetching data..."}
              {geoJsonError ? (
                <div>
                  Error downloading airspace map data:<br/>
                  <b>{geoJsonError}</b><p/>
                  <em>Please reload the page or contact administrator.</em>
                </div>
              ) : error ? (
                <div>Airspace info API unreachable:<br/>
                <b>{error}</b></div>
              ) : null}
            </p>
            <p>
              {!isFetching && !geoJsonError && <button className="button" onClick={apiFetchAreas}>⟳</button>}
            </p>
          </div>
        </h3>
      </div>

      {featureState && apiAreaData && (<InfoBox
        apiAreaData={apiAreaData} feature={featureState} visibility={infoBoxVisibility} onClose={toggleInfoBoxVisibility}
      />)}
      
      <div className={`${!apiAreaData || isFetching ? 'grayscale' : ''}`}>
        <MapContainer center={INTERLAKEN_COORDS} zoom={13} style={{ height: '100vh', width: '100%' }}>
          <TileLayer url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors' />
          
          {/* Render GeoJSON data */}
          {apiAreaData && geoJsonData && (
            <GeoJSON
              data={geoJsonData}
              style={(feature) => ({
                color: getStylingForFeature(feature, apiAreaData).Color,
                weight: 3,
                opacity: getStylingForFeature(feature, apiAreaData).Opacity,
                interactive: true,
              })}
              onEachFeature={(feature, layer) => {
                layer.on('click', () => {
                  setInfoBoxVisibility(true);
                  if (feature.properties.Name !== featureState?.properties?.Name) {
                    setFeatureState(feature);
                  }
                });
              }}
            />
          )}
        </MapContainer>
        </div>
    </div>
  );
};

export default App;

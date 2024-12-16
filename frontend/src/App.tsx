import React, { useEffect, useState } from 'react';
import './App.css';
import { MapContainer, TileLayer, GeoJSON } from 'react-leaflet';
import { LatLngTuple } from 'leaflet';
import 'leaflet/dist/leaflet.css';
import {
  ApiResponseArea, Area, SubArea,
  ApiResponseTranscript,
  fetchApiAreas, fetchApiTranscript, getStylingForFeature, resolveAreaFromFeature
 } from './utils/fetchApiData';
import InfoBox from './InfoBox';

// Define the center coordinates for Interlaken, Switzerland
const INTERLAKEN_COORDS: LatLngTuple = [46.6863, 7.8632]; // Latitude, Longitude

const App: React.FC = () => {
  const [geoJsonData, setGeoJsonData] = useState<any>(null);
  const [apiAreaData, setApiAreaData] = useState<ApiResponseArea | null>(null);
  const [apiTranscriptData, setApiTranscriptData] = useState<ApiResponseTranscript | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isFetching, setFetching] = useState<boolean>(false);
  const [featureState, setFeatureState] = useState<any>(null)
  
  const apiFetchAreas = () => {
    setError(null)
    setFetching(true)

    fetchApiAreas()
      .then(setApiAreaData)
      .catch((err) => setError(err.message))
      .finally(() => setFetching(false));
  }

  // Infobox

  const setInfoBoxContent = (feature: any, apiData: ApiResponseArea) => {
    console.log("CALLING")
    setError(null)
    const resolvedArea = resolveAreaFromFeature(feature, apiData)
    const areaName = resolvedArea?.Name ?? ""
    if (areaName === "") {
      console.log("ERR")
      setError("Could not resolve area name")
      return
    }

    setInfoBoxAreaName(areaName)
    setInfoBoxLastAction(`${resolvedArea?.LastAction}`)

    console.log("SETTING TRANSCTIPT")
    fetchApiTranscript(areaName)
      .then(setApiTranscriptData)
      .catch((err) => setError(err.message))
    
    console.log("apiTranscript:", apiTranscriptData)
  }
  /*
  useEffect(() => {
    const handleClickOutside = (event: any) => {
      if (!showInfoBox) return
      if (event.target.closest('.area-info-box')) return;
      setInfoBoxVisbility(false);
    };
    
    document.addEventListener('click', handleClickOutside);

    return () => {
      document.removeEventListener('click', handleClickOutside);
    };
  }, []);*/

  // Fetch GeoJSON data
  useEffect(() => {
    fetch('/shv_airspaces_processed.json')
      .then(response => response.json())
      .then(data => setGeoJsonData(data))
      .catch(error => console.error('Error loading GeoJSON:', error));
  }, []);

  // Fetch data from the REST API
  useEffect(apiFetchAreas, []);
  
  return (
    <div className="App">
      {(!apiAreaData || error) && <div className="gray-overlay"></div>}

      <div id="fetch-info-box" className={`box ${error ? 'error-box' : ''}`} hidden={apiAreaData === null ? false : true}>
        <h3>
          <div>
            <h1>
              {!apiAreaData && !error && (<span className="clock-spinner"></span>)}
              {error && "❌"}
            </h1>
            <p>
              {!apiAreaData && !error && "Fetching airspace status..."}
              {error && (<div>API unreachable: <b>{error}</b></div>)}
            </p>
            <p>
              {!isFetching && <button className="button" onClick={apiFetchAreas}>⟳</button>}
            </p>
          </div>
        </h3>
      </div>

      <InfoBox apiArea={apiAreaData} feature={featureState} error={error} />
      
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
                  setFeatureState(feature)
                  setInfoBoxVisbility(true)
                  setInfoBoxContent(feature, apiAreaData)
                });
              }}
            />
          )}
        </MapContainer>
        </div>
    </div>
  );
}

export default App;

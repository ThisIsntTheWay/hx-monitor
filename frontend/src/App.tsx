import React, { useEffect, useState } from 'react';

import './App.css';
import './styles/Buttons.css';
import './styles/Overlays.css';
import 'leaflet/dist/leaflet.css';

import { Feature, Geometry, GeoJsonObject } from 'geojson';
import {
  ApiResponseArea,
  fetchApiAreas,
  AIRPSACES_JSON_URL
} from './utils/fetchApiData';
import DisclaimerBox, { CheckIfDisclaimerMustBeShown } from './components/DisclaimerBox';
import InfoBox from './components/InfoBox';
import HelpBox from './components/HelpBox';
import NavBar from './components/NavBar';
import Map, { GeoLocationStatus } from './components/Map';

/* MAIN */
const App: React.FC = () => {
  const [apiAreaData, setApiAreaData] = useState<ApiResponseArea | null>(null);
  const [isFetching, setFetching] = useState<boolean>(false);
  const [geoJsonData, setGeoJsonData] = useState<GeoJsonObject | null>(null);
  const [geoJsonError, setGeoJsonError] = useState<string | null>(null);

  const [mustShowDisclaimer, setDisclaimerState] = useState<boolean>(false);
  const [infoBoxVisibility, setInfoBoxVisibility] = useState<boolean>(false);
  const [helpBoxVisibility, setHelpBoxVisibility] = useState<boolean>(false);
  const [featureState, setFeatureState] = useState<Feature<Geometry> | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [centerMap, setCenterMap] = useState(false);
  const [geoLocationStatus, setGeoLocationStatus] = useState<GeoLocationStatus>(() => {return {
    canGetGeolocation: false,
    canGetUserPosition: false
  };});

  /* Disclaimer */
  useEffect(() => {
    setDisclaimerState(CheckIfDisclaimerMustBeShown);
  }, []);

  const handleDisclaimerAcknowledged = () => {
    setDisclaimerState(false);
  };

  /* API fetching, box states */
  const toggleInfoBoxVisibility = () => {
    setInfoBoxVisibility(prevVisibility => !prevVisibility);
  };
  const toggleHelpBoxVisibility = () => {
    setHelpBoxVisibility(prevVisibility => !prevVisibility);
  };
  const hideAllBoxes = () => {
    setInfoBoxVisibility(false);
    setHelpBoxVisibility(false);
  }
  
  const apiFetchAreas = () => {
    if (isFetching) return;
    setError(null);
    setFetching(true);

    fetchApiAreas()
      .then(setApiAreaData)
      .catch((err) => setError(err.message))
      .finally(() => setFetching(false));
  };
  useEffect(apiFetchAreas, []);

  const handleCenterMap = () => {
    setCenterMap(true);
    setTimeout(() => setCenterMap(false), 1000);
  };

  // Get GeoJSON data
  useEffect(() => {
    fetch(AIRPSACES_JSON_URL)
      .then(response => {
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        return response.json();
      })
      .then(data => setGeoJsonData(data))
      .catch(err => setGeoJsonError(err.message));
  }, []);
  
  return (
    <div className="App">
      {/* Priority 1 item: Disclaimer box */}
      {mustShowDisclaimer && CheckIfDisclaimerMustBeShown() && (
        <>
          <div className="overlay disclaimer" hidden={!mustShowDisclaimer}></div>
          <DisclaimerBox onAck={handleDisclaimerAcknowledged} />
        </>
      )}

      {/* Overlays */}
      {(!apiAreaData || error || geoJsonError) && <div className="overlay gray"></div>}
      {(infoBoxVisibility || helpBoxVisibility) && <div className="overlay blur" onClick={hideAllBoxes}></div>}

      {/* Fetch box */}
      <div
        id="fetch-info-box"
        className={`box popup ${(error || geoJsonError) ? 'error' : ''}`}
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
              {!isFetching && !geoJsonError && <button onClick={apiFetchAreas}>⟳</button>}
            </p>
          </div>
        </h3>
      </div>

      {/* Area info box */}
      {featureState && apiAreaData && (<InfoBox
        apiAreaData={apiAreaData} feature={featureState} visibility={infoBoxVisibility} onClose={toggleInfoBoxVisibility}
      />)}

      {/* Help box */}
      <HelpBox visibility={helpBoxVisibility} onClose={toggleHelpBoxVisibility} />

      {/* NavBar */}
      <NavBar
        refetchEvent={apiFetchAreas}
        isFetching={isFetching}
        geoLocationStatus={geoLocationStatus}
        onLocalize={() => {if (geoLocationStatus.canGetUserPosition) handleCenterMap();}}
        onOpenHelp={toggleHelpBoxVisibility}
      />
      
      {/* Map */}
      <div className={`${!apiAreaData || isFetching ? 'grayscale' : ''}`}>
        {geoJsonData && apiAreaData && (
          <Map
            apiAreaData={apiAreaData} geoJsonData={geoJsonData}
            geoLocationStatusUpdate={(g) => setGeoLocationStatus(g)}
            featureStateUpdate={(f) => setFeatureState(f)}
            infoBoxVisibilityUpdate={(s) => setInfoBoxVisibility(s)}
            centerMap={centerMap}
          />
        )}
      </div>
    </div>
  );
};

export default App;

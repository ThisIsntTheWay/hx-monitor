import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, GeoJSON, Marker } from 'react-leaflet';
import { Feature, Geometry, GeoJsonObject } from 'geojson';
import { ApiResponseArea, getStylingForFeature } from '../utils/fetchApiData';
import L, { LatLngTuple, LatLngBounds } from 'leaflet';

interface UserPosition {
    lat: number;
    lng: number;
  }
interface UserAltitudeAndHeading {
    alt: number | null;
    altAcc: number | null;
    hdn: number | null;
}

export interface GeoLocationStatus {
    canGetGeolocation: boolean;
    canGetUserPosition: boolean;
}

interface MapProps {
    apiAreaData: ApiResponseArea,
    geoJsonData: GeoJsonObject,
    
    // Proxied to NavBar
    geoLocationStatusUpdate: (g: GeoLocationStatus) => void,
    // Proxied to InfoBox
    featureStateUpdate: (f: Feature<Geometry>) => void,
    infoBoxVisibilityUpdate: (s: boolean) => void,

    localizeHandler: (arg0: any) => void,
}
  
const INTERLAKEN_COORDS: LatLngTuple = [46.6863, 7.8632];
const switzerlandBounds = new LatLngBounds(
    [45.8, 5.8], // SW
    [47.8, 10.5] // NE
);

export const Map: React.FC<MapProps> = ({
    apiAreaData, geoJsonData,
    geoLocationStatusUpdate, featureStateUpdate, infoBoxVisibilityUpdate,
    localizeHandler
}) => {
    const [featureState, setFeatureState] = useState<Feature<Geometry> | null>(null);
    const [map, setMap] = useState<L.Map | null>(null);
    const [userPosition, setUserPosition] = useState<UserPosition | null>(null);
    const [userAltitudeAndHeading, setUserAltitudeAndHeading] = useState<UserAltitudeAndHeading | null>(null);
    const [geoLocationStatus, setGeoLocationStatus] = useState<GeoLocationStatus>(() => {
        return {
            canGetGeolocation: false,
            canGetUserPosition: false
        }
    });
    
    let userLocationIcon = L.divIcon({
        className: 'marker-dot',
        html: "<div class='" + (geoLocationStatus.canGetUserPosition ? "blue" : "gray") + "'></div>",
        iconSize: [30, 30],
        iconAnchor: [15, 15],
    });
    
    useEffect(() => {
        if (map && userPosition) {
            map.setView([userPosition.lat, userPosition.lng], 11);
        }
    }, [localizeHandler]);

    useEffect(() => {
        if (featureState) featureStateUpdate(featureState);
    }, [featureState]);

    useEffect(() => {
        const intervalId = setInterval(() => {
            navigator.geolocation.getCurrentPosition(
                (position) => {
                    setUserPosition({
                        lat: position.coords.latitude,
                        lng: position.coords.longitude,
                    });
                    setUserAltitudeAndHeading({
                        alt: position.coords.altitude,
                        altAcc: position.coords.altitudeAccuracy,
                        hdn: position.coords.heading,
                    });

                    const g = {
                        canGetGeolocation: true,
                        canGetUserPosition: true
                    };
                    setGeoLocationStatus(g);
                    geoLocationStatusUpdate(g);
                },
                (error) => {
                    const g = {
                        canGetGeolocation: error.code !== error.POSITION_UNAVAILABLE,
                        canGetUserPosition: error.code === error.POSITION_UNAVAILABLE
                    };
                    setGeoLocationStatus(g);
                    geoLocationStatusUpdate(g);
                    console.error('Error getting user location:', error);
                }
            );
        }, 5000);
    
        return () => clearInterval(intervalId);
      }, []);
      
    /*
    useEffect(() => {
        console.log(userAltitudeAndHeading);
    }, [userAltitudeAndHeading]);
    */

    return (
        <MapContainer
            ref={setMap}
            center={userPosition ? userPosition : INTERLAKEN_COORDS}
            zoom={10}
            style={{ height: '100vh', width: '100%' }}
            maxBounds={switzerlandBounds}
            maxBoundsViscosity={1.0}
            maxZoom={13}
        >
        <TileLayer
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        />

        {/* Positioning */}
        {userPosition && (
            <Marker position={userPosition} icon={userLocationIcon} />
        )}
        
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
                    infoBoxVisibilityUpdate(true);
                    if (feature.properties.Name !== featureState?.properties?.Name) {
                        setFeatureState(feature);
                    }
                });
            }}
            />
        )}
        </MapContainer>
    );
};

export default Map;
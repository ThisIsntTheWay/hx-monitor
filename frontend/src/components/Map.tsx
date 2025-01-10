import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, GeoJSON, Marker, useMapEvent } from 'react-leaflet';
import { Feature, Geometry, GeoJsonObject } from 'geojson';
import { ApiResponseArea, getStylingForFeature } from '../utils/fetchApiData';
import L, { LatLngTuple, LatLngBounds, TooltipOptions } from 'leaflet';

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
    centerMap: boolean,
    
    // Proxied to NavBar
    geoLocationStatusUpdate: (g: GeoLocationStatus) => void,
    // Proxied to InfoBox
    featureStateUpdate: (f: Feature<Geometry>) => void,
    infoBoxVisibilityUpdate: (s: boolean) => void,

}
  
const INTERLAKEN_COORDS: LatLngTuple = [46.6863, 7.8632];
const switzerlandBounds = new LatLngBounds(
    [45.8, 5.8], // SW
    [47.8, 10.5] // NE
);

const makeTooltipText = (subAreaName: string): string => {
    const areaType = subAreaName.split(" ")[0];

    let areaNum = "";
    const numMatches = subAreaName.match(/\d+/);
    if (numMatches !== null) areaNum = numMatches[0];

    return areaType + (areaNum ? " " + areaNum : "");
}

const generateTooltipContent = (feature: any, zoomLevel: number): string => {
    const baseText = `<span class="text main">${makeTooltipText(feature.properties.Name)}</span>`;
    if (zoomLevel >= 12) {
        const upper = feature.properties.Upper?.Metric?.Alt;
        const lower = feature.properties.Lower?.Metric?.Alt;
        const details = [
            `⬆️ ${upper.Altitude} ${upper.Type}`,
            `⬇️ ${lower.Altitude} ${lower.Type}`
        ]
        return `${baseText}<br/><span class="text details">${details.join("<br/>")}</span>`;
    }

    return baseText;
};

export const Map: React.FC<MapProps> = ({
    apiAreaData, geoJsonData, centerMap,
    geoLocationStatusUpdate, featureStateUpdate, infoBoxVisibilityUpdate
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

    const tooltipProps: TooltipOptions = {permanent: true, direction: 'center'};

    const MapZoomListener = () => {
        const map = useMapEvent('zoomend', () => {
            updateTooltips(map.getZoom());
        });
        return null;
    };

    useEffect(() => {
        if (centerMap && userPosition && map) {
            map.setView(userPosition, 11);
        }
    }, [centerMap, userPosition]);

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

    const updateTooltips = (zoomLevel: number) => {
        if (map) {
            map.eachLayer((l) => {
                if (l instanceof L.Polygon) {
                    if (zoomLevel > 9) {
                        l.bindTooltip(
                            generateTooltipContent(l.feature, zoomLevel),
                            tooltipProps
                        );
                    } else {
                        l.unbindTooltip();
                    }
                }
            })
        }
    };
      
    /*
    useEffect(() => {
        console.log(userAltitudeAndHeading);
    }, [userAltitudeAndHeading]);
    */

    let userLocationIcon = L.divIcon({
        className: "marker-dot" + (geoLocationStatus.canGetUserPosition ? " located" : ""),
        iconSize: [20, 20],
        iconAnchor: [15, 15],
    });

    return (
        <MapContainer
            ref={setMap}
            center={userPosition ? userPosition : INTERLAKEN_COORDS}
            zoom={10}
            style={{ height: '100vh', width: '100%' }}
            maxBounds={switzerlandBounds}
            maxBoundsViscosity={1.0}
            minZoom={9}
            maxZoom={13}
        >
        <TileLayer
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        />
        <MapZoomListener />

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

                    layer.bindTooltip(
                        generateTooltipContent(feature, 10),
                        tooltipProps
                    ).openTooltip();
                }}
            />
        )}
        </MapContainer>
    );
};

export default Map;
import React, { useState } from "react";
import '../styles/Boxes.css';
import { GeoLocationStatus } from "./Map";

interface NavBarProps {
    refetchEvent: () => void,
    isFetching: boolean,
    onLocalize: () => void,
    onOpenHelp: () => void,
    geoLocationStatus: GeoLocationStatus
}

const NavBar: React.FC<NavBarProps> = ({
    refetchEvent,
    isFetching,
    onLocalize,
    onOpenHelp,
    geoLocationStatus
}) => {
    const [btnDisabled, setBtnDisabled] = useState<boolean>(false);
    
    const fireRefreshApiEvent = () => {
        // To prevent API request spam, the button will always remain disabled for a set amount of time
        setBtnDisabled(true);
        refetchEvent();
        setTimeout(() => {
            setBtnDisabled(false);
        }, 5000);
    };
    
    return (
        <div className="box nav">
            {/* API refetch */}
            <button disabled={btnDisabled || isFetching} onClick={fireRefreshApiEvent}>
                {isFetching ? (
                    <span className="clock-spinner"></span>
                ) : (
                    <>
                        🔄
                        {btnDisabled && (
                            <span className="nav-button-error-descriptor">
                                👍
                            </span>
                        )}
                    </>
                )}
            </button>

            {/* Locate on map */}
            <button
                disabled={!geoLocationStatus.canGetGeolocation || !geoLocationStatus.canGetUserPosition}
                onClick={onLocalize}
            >
                🧭
                <span className="nav-button-error-descriptor">
                    {!geoLocationStatus.canGetGeolocation ? (
                        "❌"
                    ) : !geoLocationStatus.canGetUserPosition && (
                        "❓"
                    )}
                </span>
            </button>

            {/* Help */}
            <button onClick={onOpenHelp}>❓</button>
        </div>
    );
};

export default NavBar;
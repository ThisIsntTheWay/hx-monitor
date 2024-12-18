import React, { useEffect, useState } from 'react';
import {
  ApiResponseArea, ApiResponseTranscript, Area,
  resolveAreaFromFeature, fetchApiTranscript
} from './utils/fetchApiData';

/* Box */
export interface BoxData {
  apiAreaData: ApiResponseArea | null,
  feature: any,
  visibility: boolean,
  onClose: any,
}

// Checks if current time is during active flight operation hours
const withinFlightOperatingHours = (area: Area): boolean => {
  let indexOfLastPastFlightOpTime = 0;
  const flightOpsLength = area.FlightOperatingHours.length;
  const now = new Date().getTime();

  if (flightOpsLength === 2 || flightOpsLength === 4) {
    area.FlightOperatingHours.forEach((v, i) => {
      const thisFlightOpTime = new Date(v).getTime();
      if (now - thisFlightOpTime > 0) {
        indexOfLastPastFlightOpTime = i;
      } else {
        return;
      }
    });

    const adjustedIndex = indexOfLastPastFlightOpTime + 1;
    return flightOpsLength === 4 ? adjustedIndex !== 2 : adjustedIndex < flightOpsLength;
  }
  
  return false;
};

// Calculates difference between now and timeString, returning "Nd, Nh, Nm"
// Will always return absolute numbers!
const timeDiffString = (timeString: string): string => {
  const pastDate = new Date(timeString).getTime();
  const now = new Date().getTime();

  const diffMs = now - pastDate;

  const diffDays = Math.floor(Math.abs(diffMs) / (1000 * 60 * 60 * 24));
  const diffHours = Math.floor((Math.abs(diffMs) % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const diffMinutes = Math.floor((Math.abs(diffMs) % (1000 * 60 * 60)) / (1000 * 60));

  // Build the time ago string based on which values are non-zero
  let result = '';
  if (diffDays > 0) result += `${diffDays}d, `;
  if (diffHours > 0 || diffDays > 0) result += `${diffHours}h, `;
  if (diffMinutes > 0) result += `${diffMinutes}m`;

  // Remove trailing stuff
  result = result.replace(/, $/, '');

  return result;
};

const capitalizeString = (input: string): string => {
  return String(input).charAt(0).toUpperCase() + String(input).slice(1);
};

const InfoBox: React.FC<BoxData> = ({ apiAreaData, feature, visibility, onClose }) => {
  /* States */
  const [apiTranscriptData, setApiTranscriptData] = useState<ApiResponseTranscript | null>(null);
  const [lastUpdateTime, updateLastUpdateTime] = useState<string>("...");
  const [nextUpdateTime, updateNextUpdateTime] = useState<string>("...");
  const [err, setError] = useState<string>("");

  const resolvedArea = feature && apiAreaData ? resolveAreaFromFeature(feature, apiAreaData) : null;

  useEffect(() => {
    if (feature && apiAreaData) {
      const resolvedArea = resolveAreaFromFeature(feature, apiAreaData);

      if (!resolvedArea) {
        setError("UI error: Could not resolve area from given feature.");
        console.error("Could not resolve area from feature:", feature);
        return;
      }

      if (resolvedArea.Name === "Unknown") {
        setError("UI error: Could not resolve area from given feature.");
        return;
      }

      // Fetch transcript data for the resolved area
      setError("");
      fetchApiTranscript(resolvedArea.Name)
        .then(setApiTranscriptData)
        .catch((err) => setError(err.message));
    }
  }, [feature, apiAreaData]);

  // Keep refreshing update times so the client is always up to date
  useEffect(() => {
    const updateTimeStates = () => {
      if (resolvedArea) {
        updateLastUpdateTime(timeDiffString(resolvedArea.LastAction));
        updateNextUpdateTime(timeDiffString(resolvedArea.NextAction));
      }
    };

    updateTimeStates();
    const intervalId = setInterval(() => {
      updateTimeStates();
    }, 1000);

    return () => clearInterval(intervalId);
  }, [resolvedArea]);

  if (!visibility) return null;

  return (
    <div id="area-info-box" className={`box ${!resolvedArea?.LastActionSuccess ? "warning-box" : ''}`} hidden={!visibility}>
      <button className="close-btn" onClick={onClose}>X</button>
      {resolvedArea && !err ? (
        <>
          {/* Header */}
          <h1>{!resolvedArea.LastActionSuccess && "⚠️"} {capitalizeString(resolvedArea.Name)}</h1>
          
          {/* Update times */}
          <p>
            Last updated <span className="time-string">{lastUpdateTime}</span> ago<br/>
            {resolvedArea.LastAction && (
              <>
                Next update in <span className="time-string">{nextUpdateTime}</span><br/>
              </>
            )}
          </p>

          {/* Flight operating hours */}
          {resolvedArea.FlightOperatingHours ? (
            <>
            <span className={`flight-ops-status-text ${withinFlightOperatingHours(resolvedArea) ? ("within") : ("outside")}`}>
            {withinFlightOperatingHours(resolvedArea) ? (
              "Within"
            ) : (
              "Outside"
            )}
            </span> flight operating hours
            </>
          ) : (
            "No flight operating hours today"
          )}

          {/* SubAreas */}
          {resolvedArea.LastActionSuccess ? (
            <div>
              {resolvedArea.SubAreas.map((subArea, i) => (
                <p key={i}>
                  <strong>{subArea.Fullname}</strong> {(resolvedArea.LastActionSuccess && !subArea.Active) ? "🟢" : "🔴"}<br/>
                </p>
              ))}
            </div>
          ) : (
            <div>
              <h3><strong>The parser has encountered an error!</strong></h3>
              <h4><em>{resolvedArea.LastError ? resolvedArea.LastError : "Unknown error"}</em></h4>
              Area status could not be dynamically determined.<br/>
              Either consult transcript or call {capitalizeString(resolvedArea.Name)} directly.
            </div>
          )}

          {/* Transcripts */}
          <hr/>
          <h3><strong>Transcript</strong></h3>
          {apiTranscriptData ? (
            <p>
              {apiTranscriptData.data.transcript ? (
                <span className="transcript-string">
                  {apiTranscriptData.data.transcript}
                </span>
              ) : (
                <strong>❌ Unavailable</strong>
              )}
            </p>
          ) : (
            <div>
              {!err ? (
                <p><span className="clock-spinner"></span> Fetching...</p>
              ) : (
                <p>❌ Could not fetch: {err}</p>
              )}
            </div>
          )}
        </>
      ) : (
        <p>{err || "Loading..."}</p>
      )}
    </div>
  );
};

export default InfoBox;

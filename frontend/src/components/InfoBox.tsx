import React, { useEffect, useState } from 'react';
import { Feature, Geometry } from 'geojson';
import {
  ApiResponseArea, ApiResponseTranscript, Area,
  resolveAreaFromFeature, fetchApiTranscript, nextUpdateIsInThePast
} from '../utils/fetchApiData';

/* Box */
interface boxData {
  apiAreaData: ApiResponseArea | null,
  feature: Feature<Geometry>,
  visibility: boolean,
  onClose: () => void,
}

// Checks if current time is during active flight operation hours
const withinFlightOperatingHours = (area: Area): boolean => {
  let indexOfLastPastFlightOpTime = 0;
  const flightOpsLength = area.flight_operating_hours.length;
  const now = new Date().getTime();

  if (flightOpsLength === 2 || flightOpsLength === 4) {
    area.flight_operating_hours.forEach((v, i) => {
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

const InfoBox: React.FC<boxData> = ({ apiAreaData, feature, visibility, onClose }) => {
  /* States */
  const [apiTranscriptData, setApiTranscriptData] = useState<ApiResponseTranscript | null>(null);
  const [lastUpdateTime, updateLastUpdateTime] = useState<string>("...");
  const [nextUpdateTime, updateNextUpdateTime] = useState<string>("...");
  const [useWarningStyling, updateWarningStyling] = useState<boolean>(false);
  const [err, setError] = useState<string>("");

  const resolvedArea = feature && apiAreaData ? resolveAreaFromFeature(feature, apiAreaData) : null;

  useEffect(() => {
    if (feature && apiAreaData) {
      const resolvedArea = resolveAreaFromFeature(feature, apiAreaData);

      if (!resolvedArea || resolvedArea.name === "Unknown") {
        setError("UI error: Could not resolve area from given feature.");
        console.error("Could not resolve area from feature:", feature);
        return;
      }
      updateWarningStyling(!resolvedArea?.last_action_success || nextUpdateIsInThePast(resolvedArea));

      // Fetch transcript data for the resolved area
      setError("");
      fetchApiTranscript(resolvedArea.name)
        .then(setApiTranscriptData)
        .catch((err) => setError(err.message));
    }
  }, [feature, apiAreaData]);

  // Ensure dynamic update times
  useEffect(() => {
    const updateTimeStates = () => {
      if (resolvedArea) {
        updateLastUpdateTime(timeDiffString(resolvedArea.last_action));
        updateNextUpdateTime(timeDiffString(resolvedArea.next_action));
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
    <div className={`box popup ${err ? "error" : (useWarningStyling ? 'warning' : '')}`} hidden={!visibility}>
      <button className="close" onClick={onClose}>❌</button>
      {resolvedArea && !err ? (
        <>
          {/* Header */}
          <h1>{useWarningStyling && "⚠️"} {capitalizeString(resolvedArea.name)}</h1>
          
          {/* Update times */}
          <p>
            Last updated <span className="time-string">{lastUpdateTime}</span> ago<br/>
            {resolvedArea.last_action && (
              <>
                {!nextUpdateIsInThePast(resolvedArea) ? (
                  <>Next update in <span className="time-string">{nextUpdateTime}</span></>
                ) : (
                  <span className="error-string">
                    Next update expected <span className="time-string">{nextUpdateTime}</span> ago!
                  </span>
                )}
                <br/>
              </>
            )}
          </p>

          {/* Warning on parser errors etc. */}
          {nextUpdateIsInThePast(resolvedArea) || !resolvedArea.last_action_success ? (
            <h2>Assume area to be active!</h2>
          ) : (null)}

          <div className="scrollable">
          {/* Flight operating hours */}
          {!nextUpdateIsInThePast(resolvedArea) ? (
            resolvedArea.flight_operating_hours ? (
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
              <>{resolvedArea.last_action_success && "No flight operating hours today"}</>
            )
          ): (null)}

          {/* SubAreas */}
          {resolvedArea.last_action_success && !nextUpdateIsInThePast(resolvedArea) ? (
            <div>
              {/*
              {resolvedArea.sub_areas.map((subArea, i) => (
                <p key={i}>
                  <strong>{subArea.full_name}</strong> {(resolvedArea.last_action_success && !subArea.active) ? "🟢" : "🔴"}<br/>
                </p>
              ))}
                */}
            </div>
          ) : (
            <div>
              {nextUpdateIsInThePast(resolvedArea) ? (
                <>
                  Previously known information about the area has become stale.<br/>
                  Wait for the area to be updated or call {capitalizeString(resolvedArea.name)} directly.
                </>
              ) : (
                <>
                  <p>
                    The parser has encountered an error:<br/>
                    <strong>{resolvedArea.last_error ? resolvedArea.last_error : "Unknown error"}</strong>
                  </p>

                  Area status could not be dynamically determined.<br/>
                  Either consult transcript or call {capitalizeString(resolvedArea.name)} directly.
                </>
              )}
            </div>
          )}

          {/* Transcript */}
          <hr/>
          📞
          <span className="transcript-string">
            {apiTranscriptData ? (
              <>
                {apiTranscriptData.data?.transcript ? (
                  <>💬 &quot;{apiTranscriptData.data.transcript}&quot;</>
                ) : (
                  <>❌ <em>No transcript was returned</em></>
                )}
              </>
            ) : (
              <>
                {!err ? (
                  <><span className="clock-spinner"></span> <em>Fetching Transcript...</em></>
                ) : (
                  <>❌ <em>Could not fetch transcript:</em> <strong>{err}</strong></>
                )}
              </>
            )}
          </span>
          </div>
          {/* END OF BOX CONTENTS */}
        </>
      ) : (
        <p>{err || "Loading..."}</p>
      )}
    </div>
  );
};

export default InfoBox;

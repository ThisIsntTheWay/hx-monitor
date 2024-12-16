import React, { useEffect, useState } from 'react';
import { ApiResponseArea, ApiResponseTranscript, resolveAreaFromFeature, fetchApiTranscript } from './utils/fetchApiData';

/* Box */
export interface BoxData {
  apiAreaData: ApiResponseArea | null,
  feature: any,
  visibility: boolean,
}

const InfoBox: React.FC<BoxData> = ({ apiAreaData, feature, visibility }) => {
  /* States */
  const [showInfoBox, setShowInfoBox] = useState<boolean>(visibility);
  const [apiTranscriptData, setApiTranscriptData] = useState<ApiResponseTranscript | null>(null);
  const [err, setError] = useState<string>("");

  useEffect(() => {
    setShowInfoBox(visibility);
  }, [visibility]);

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

  const closeInfoBox = () => setShowInfoBox(false);
  const resolvedArea = feature && apiAreaData ? resolveAreaFromFeature(feature, apiAreaData) : null;

  const timeAgoString = (): string => {
    if (!resolvedArea) {
      return "❓"
    }

    const pastDate = new Date(resolvedArea.LastAction).getTime();
    const now = new Date().getTime();

    const diffMs = now - pastDate;

    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffMinutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));
  
    // Build the time ago string based on which values are non-zero
    let result = '';
    if (diffDays > 0) result += `${diffDays}d, `;
    if (diffHours > 0 || diffDays > 0) result += `${diffHours}h, `;
    result += `${diffMinutes}m`;

    result = result.replace(/, $/, '');

    return result;
  }

  const capitalizeString = (input: string): string => {
    return String(input).charAt(0).toUpperCase() + String(input).slice(1);
  }

  return (
    <div id="area-info-box" className="box" hidden={!showInfoBox}>
      <button className="close-btn" onClick={closeInfoBox}>✖</button>
      {resolvedArea && !err ? (
        <>
          <h1>{capitalizeString(resolvedArea.Name)}</h1>
          <p>
            Last updated: <span className="time-string">{timeAgoString()}</span> ago<br/>
            Next action: {resolvedArea.NextAction}
          </p>
          
          {resolvedArea.SubAreas.map((subArea, i) => (
            <p key={i}>
              <strong>{subArea.Fullname}</strong> {subArea.Status ? "🔴" : "🟢"}<br/>
            </p>
          ))}
          <h3>Transcript</h3>
          {/* Ensure that transcripts were actually fetched */}
          {apiTranscriptData && apiTranscriptData.data && Array.isArray(apiTranscriptData.data.Transcripts) ? (
            <p>
              {apiTranscriptData.data.Transcripts.length > 0
                ? apiTranscriptData.data.Transcripts[0].Transcript
                : "❌ No transcripts available."}
            </p>
          ) : (
            <p><span className="clock-spinner"></span>Fetching...</p>
          )}
        </>
      ) : (
        <p>{err || "Loading..."}</p>
      )}
    </div>
  );
};

export default InfoBox;

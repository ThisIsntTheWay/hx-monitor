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

  return (
    <div id="area-info-box" className="box" hidden={!showInfoBox}>
      <button className="close-btn" onClick={closeInfoBox}>X</button>
      {resolvedArea && !err ? (
        <>
          <h1>{resolvedArea.Name}</h1>
          <h3>Sub areas</h3>
          {resolvedArea.SubAreas.map((subArea, i) => (
            <p key={i}>
              Index: {i}
              <br />
              FullName: {subArea.Fullname}
              <br />
              Status: {subArea.Status}
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
          <p>
            Next action: {resolvedArea.NextAction}<br/>
            Last action: {resolvedArea.LastAction}
          </p>
        </>
      ) : (
        <p>{err || "Loading..."}</p>
      )}
    </div>
  );
};

export default InfoBox;

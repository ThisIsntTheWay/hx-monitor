import React, { useState } from 'react';
import { ApiResponseArea, ApiResponseTranscript, resolveAreaFromFeature } from './utils/fetchApiData';

/* States */
const [showInfoBox, setVisibility] = useState<boolean>(false);
const closeInfoBox = () => setVisibility(false)

/* Box */
export interface BoxData {
  apiArea: ApiResponseArea | null,
  apiTranscript: ApiResponseTranscript | null,
  feature: any,
  error: string | null,
}

const InfoBox: React.FC<BoxData> = (props) => {
  const { apiArea, feature, error } = props;

  if (apiArea === null) {
    console.error("'apiArea' must not be null before InfoBox can be shown")
    return
  }

  const resolvedArea = resolveAreaFromFeature(feature, apiArea)
  if (!resolvedArea) {
    console.error("Could not resolve area from feature:", feature)
    return
  }

  return (
    <div id="area-info-box" className="box" hidden={!showInfoBox}>
    <button className="close-btn" onClick={closeInfoBox}>X</button>
    <h1>{resolvedArea.Name}</h1>
    {!error ? (
      <p>
        <h3>Sub areas</h3>
        {resolvedArea.SubAreas.map((subArea, i) => (
          <p key={i}>
            Index: {i}<br/>
            FullName: {subArea.Fullname}
            Status: {subArea.Status}
          </p>
        ))}
        <p>
          <h3>Transcript</h3>
          {apiTranscript && `${apiTranscript.data.Transcripts[0].Transcript}`}
        </p>
        Last action: {resolvedArea.LastAction}
      </p>
    ) : (
      <p>
        {error}
      </p>
    )}
  </div>
  );
};

export default InfoBox;

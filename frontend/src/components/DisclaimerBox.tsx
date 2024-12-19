import React, { useEffect, useState } from 'react';

/* Box */
interface boxData {
    visibility: boolean,
};

// Acknowledge disclaimer box; Permanently hiding it
const ackDisclaimer = (): boolean => {
    return true;
}

// Check if disclaimer box must be shown, i.e. first time visiting user
export const CheckIfDisclaimerMustBeShown = (): boolean => {
    return false;
}

const DisclaimerBox: React.FC<boxData> = ({visibility}) => {
    return (
        <div id="disclaimer-box" className="box disclaimer" hidden={!visibility}>
            <h1>ℹ️</h1>
            <h2>Experimental service</h2>

            <p>
                This website shows a map of CTRs/TMAs of type HX and whether or not they are active.<br/>
                To determine activeness of an area, the audio tape of the corresponding zone is <em>regularly called</em>.<br/>
                The call will then be <em>machine transcribed</em> and parsed accordingly.
            </p>
            <p>
                A high degree of accurracy <strong>cannot be guaranteed</strong>;<br/>
                Mistranscriptions can occur, or the parser might not correctly interpret the transcript.<br/>
            </p>
            <p>
                <strong>
                    If in doubt, always consult the audio tape of an area directly!
                    Do not solely rely on this service for any flight planning activities.
                </strong>
            </p>

            By clicking on the thumbs up, you acknowledge to have read and understood the <a href="disclaimer.html">full disclaimer</a>.
            <button className="button" onClick={ackDisclaimer}>👍</button>
        </div>
    )
}

export default DisclaimerBox;
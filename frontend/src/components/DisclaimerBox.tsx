import React from 'react';

/* Box */
interface boxData {
    onAck: () => void,
};

const lsDisclaimerItemName = "disclaimer_shown";

// Check if disclaimer box must be shown, i.e. first time visiting user
export const CheckIfDisclaimerMustBeShown = (): boolean => {
    const lsDisclaimerValue = localStorage.getItem(lsDisclaimerItemName); 
    return lsDisclaimerValue === null || lsDisclaimerValue !== 'true';
};

const DisclaimerBox: React.FC<boxData> = ({onAck}) => {
    // Acknowledge disclaimer box; Permanently hiding it
    const ackDisclaimer = () => {
        localStorage.setItem(lsDisclaimerItemName, 'true');
        onAck();
    };

    return (
        <div className="box popup info">
            <h1>ℹ️</h1>
            <h2>Experimental service</h2>

            <p>
                This website shows a map of CTRs/TMAs of type HX and whether or not they are active.<br/>
                To determine if an area is activated, the audio tape of the corresponding zone gets regularly called.<br/>
                The call will then be <em>machine transcribed</em> and parsed accordingly.
            </p>
            <p>
                A high degree of accurracy <strong>cannot be guaranteed</strong>;<br/>
                Mistranscriptions can occur, or the parser might not correctly interpret the transcript.<br/>
            </p>
            <p>
                <strong>
                    If in doubt, always consult the audio tape of an area directly!<br/>
                    Do not solely rely on this service for any flight planning activities.
                </strong>
            </p>

            By clicking on the thumbs up, you acknowledge to have read and understood the <a href="disclaimer.html">full disclaimer</a>.
            <p>
                <button onClick={ackDisclaimer}>👍</button>
            </p>
        </div>
    );
};

export default DisclaimerBox;
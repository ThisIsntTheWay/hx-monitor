import React from 'react';
import DynamicTable from './DynamicTable';

interface boxData {
  visibility: boolean,
  onClose: () => void,
}

const HelpBox: React.FC<boxData> = ({ visibility, onClose }) => {
  if (!visibility) return null;

  const tableDataLegend = [
    { textLeft: '🟩', textRight: 'Area is inactive' },
    { textLeft: '🟨', textRight: 'Area is presumed active' },
    { textLeft: '🟥', textRight: 'Area is active' },
  ];
  const tableDataButtons = [
    { textLeft: '🔄', textRight: 'Refresh data' },
    { textLeft: '🧭', textRight: 'Locate on map' },
    { textLeft: '❓', textRight: 'Help (you are here)' },
  ];

  return (
    <div className="box popup info" hidden={!visibility}>
      <button className="close" onClick={onClose}>❌</button>
      {/* Create a table with two cells, the left one being smaller and having a color. The text in the right cell should be left-aligned */}
      <h2>Legend</h2>
      <DynamicTable data={tableDataLegend} />

      <hr/>
      <h2>Buttons</h2>
      <DynamicTable data={tableDataButtons} />

      <hr/>
      <a href="disclaimer.html">Disclaimer</a>
    </div>
  );
};

export default HelpBox;

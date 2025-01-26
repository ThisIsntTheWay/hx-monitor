import React from "react";
import "../styles/DynamicTable.css";

interface TableData {
  textLeft: string;
  textRight: string;
  bgColor?: string;
}

interface TwoCellTableProps {
  data: TableData[];
}

const TwoCellTable: React.FC<TwoCellTableProps> = ({ data }) => {
  return (
    <div className="table-container">
      <table>
        <tbody>
          {data.map((item, index) => (
            <tr key={index}>
              <td
                className="square"
                style={{ backgroundColor: item.bgColor || 'transparent' }}
              >
                {item.textLeft}
              </td>
              <td className="text">{item.textRight}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default TwoCellTable;

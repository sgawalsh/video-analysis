import FormatMMSS from '../../format'
import LinkedThumbnailTitle from "../components/linkedThumbnailTitle";

function SemanticSearchResult({ results }) {

  const rows = Object.entries(results).flatMap(([id, content]) => {
    return content.data.map(item => ({
      targetId: id,
      title: content.title,
      ...item
    }));
  }).sort((a, b) => b.Distance - a.Distance);

  return (
    <div style={{ padding: 20 }}>

      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={{ textAlign: 'center', padding: '8px', borderBottom: '2px solid #000' }}>
              Rank
            </th>
            <th style={{ textAlign: 'center', padding: '8px', borderBottom: '2px solid #000' }}>
              Match Score
            </th>
            <th style={{ textAlign: 'center', padding: '8px', borderBottom: '2px solid #000' }}>
              Thumbnail
            </th>
            <th style={{ textAlign: 'center', padding: '8px', borderBottom: '2px solid #000' }}>
              Link
            </th>
          </tr>
        </thead>

        <tbody>
          {rows.map((row, rank) => (
            <tr key={`${row.targetId}-${row.Index}`}>
              <td style={{ padding: '8px' }}>
                #{rank + 1}
              </td>
              <td style={{ minWidth: 200 }}>
                <div
                  style={{
                    background: '#eee',
                    height: 8,
                    borderRadius: 4,
                    overflow: 'hidden'
                  }}
                >
                  <div
                    style={{
                      width: `${row.Distance * 100}%`,
                      height: '100%',
                      background: '#4caf50'
                    }}
                  />
                </div>
                <div>{row.Distance.toFixed(3)}</div>
              </td>
              <td style={{ padding: '8px' }}>
                  <LinkedThumbnailTitle title={row.title} targetId={row.targetId} startTime={row.StartTime}></LinkedThumbnailTitle>
              </td>
              <td style={{ padding: '8px', borderBottom: '1px solid #ddd' }}>
                <a
                  href={`https://www.youtube.com/watch?v=${row.targetId}&t=${row.StartTime}`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Link - {FormatMMSS(row.StartTime)}s
                </a>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default SemanticSearchResult;
import FormatMMSS from '../../format'
import LinkedThumbnailTitle from "../components/linkedThumbnailTitle";

function KeywordSearchResult({ results }) {

  const rows = Object.entries(results).flatMap(([id, content]) => {
    return content.data.map(item => ({
      targetId: id,
      title: content.title,
      ...item
    }));
  }).sort((a, b) => b.MatchCount - a.MatchCount);

  return (
    <div style={{ padding: 20 }}>

      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={{ textAlign: 'center', padding: '8px', borderBottom: '2px solid #000' }}>
              Rank
            </th>
            <th style={{ textAlign: 'center', padding: '8px', borderBottom: '2px solid #000' }}>
              Matches
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
            <tr key={`${row.targetId}-${row.StartTime}`}>
              <td style={{ padding: '8px' }}>
                #{rank + 1}
              </td>
              <td>
                <div>{row.MatchCount}</div>
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

export default KeywordSearchResult;
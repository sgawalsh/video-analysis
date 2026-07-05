import FormatMMSS from '../../format'
import LinkedThumbnailTitle from "./components/linkedThumbnailTitle";

function KeywordSearchResult({ results }) {

  const allMatches = Object.entries(results)
    .flatMap(([targetId, matches]) =>
      matches.map(m => ({ ...m, targetId }))
    )
    .sort((a, b) => b.MatchCount - a.MatchCount);

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
          {allMatches.map((match, rank) => (
            <tr key={`${match.targetId}-${match.StartTime}`}>
              <td style={{ padding: '8px' }}>
                #{rank + 1}
              </td>
              <td>
                <div>{match.MatchCount}</div>
              </td>
              <td style={{ padding: '8px' }}>
                  <LinkedThumbnailTitle targetId={match.targetId} startTime={match.StartTime}></LinkedThumbnailTitle>
              </td>
              <td style={{ padding: '8px', borderBottom: '1px solid #ddd' }}>
                <a
                  href={`https://www.youtube.com/watch?v=${match.targetId}&t=${match.StartTime}`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Link - {FormatMMSS(match.StartTime)}s
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
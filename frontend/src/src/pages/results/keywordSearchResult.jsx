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
    
    <main className="container results-page">
      {rows.length > 0 ? 
        <div className="results-table-wrapper">
          <table className="results-table">
            <thead>
              <tr>
                <th scope="col">Rank</th>
                <th scope="col">Matches</th>
                <th scope="col">Video</th>
                <th scope="col">Link</th>
              </tr>
            </thead>

            <tbody>
              {rows.map((row, rank) => (
                <tr key={`${row.targetId}-${row.StartTime}`}>
                  <td className="result-rank">
                    #{rank + 1}
                  </td>
                  <td className="match-score-cell">
                    <div>{row.MatchCount}</div>
                  </td>
                  <td className="result-video">
                      <LinkedThumbnailTitle
                        title={row.title}
                        targetId={row.targetId}
                        startTime={row.StartTime}>
                      </LinkedThumbnailTitle>
                  </td>
                  <td className="result-link">
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
        :<></>}
    </main>
  );
}

export default KeywordSearchResult;
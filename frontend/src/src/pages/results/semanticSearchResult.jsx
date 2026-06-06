function SemanticSearchResult({ results }) {

  const allMatches = Object.entries(results)
    .flatMap(([targetId, matches]) =>
      matches.map(m => ({ ...m, targetId }))
    )
    .sort((a, b) => b.Distance - a.Distance);

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
          {allMatches.map((match, rank) => (
            <tr key={`${match.targetId}-${match.Index}`}>
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
                      width: `${match.Distance * 100}%`,
                      height: '100%',
                      background: '#4caf50'
                    }}
                  />
                </div>
                <div>{match.Distance.toFixed(3)}</div>
              </td>
              <td style={{ padding: '8px' }}>
                <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                  <img
                    src={`https://img.youtube.com/vi/${match.targetId}/mqdefault.jpg`}
                    alt="thumbnail"
                    width={120}
                  />
                </div>
              </td>
              <td style={{ padding: '8px', borderBottom: '1px solid #ddd' }}>
                <a
                  href={`https://www.youtube.com/watch?v=${match.targetId}&t=${match.StartTime}`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Link - {match.StartTime}s
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
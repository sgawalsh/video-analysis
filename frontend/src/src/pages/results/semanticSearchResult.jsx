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
            <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #000' }}>
              Match Strength
            </th>
            <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #000' }}>
              Link
            </th>
          </tr>
        </thead>

        <tbody>
          {allMatches.map((match) => (
            <tr key={`${match.targetId}-${match.Index}`}>
              <td style={{ padding: '8px', borderBottom: '1px solid #ddd' }}>
                {Number(match.Distance).toFixed(3)}
              </td>

              <td style={{ padding: '8px', borderBottom: '1px solid #ddd' }}>
                <a
                  href={`https://www.youtube.com/watch?v=${match.targetId}&t=${match.StartTime}`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {match.StartTime}s
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
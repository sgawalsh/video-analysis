function SemanticSearchResult({ results }) {
  return (
        <div style={{ padding: 20 }}>
            <p>Result: {JSON.stringify(results)}</p>
            {results.map((match, index) => (
                <div key={match.Index}>
                    <strong>{index + 1}) Match Strength:</strong><span> {Number(match.Distance).toPrecision(2)} - </span>
                    <a href={`https://www.youtube.com/watch?v=${match.target_id}&t=${match.StartTime}`} target="_blank" rel="noopener noreferrer">Timestamp</a>
                </div>
            ))}
        </div>
    );
}
export default SemanticSearchResult;
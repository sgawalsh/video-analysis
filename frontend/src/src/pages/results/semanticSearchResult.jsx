import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';

function SemanticSearchResult() {
    const { id } = useParams();

    const [job, setJob] = useState(null);
    const [error, setError] = useState(null);

    useEffect(() => {
        async function fetchResult() {
            try {
                const res = await fetch(`/api/results/semanticSearchResult/${id}`,{ cache: 'no-store' });

                if (!res.ok) {
                    throw new Error(`HTTP ${res.status}`);
                }

                const data = await res.json();
                setJob(data);
            } catch (err) {
                setError(err.message);
            }
        }

        fetchResult();
    }, [id]);

    if (error) return <p>Error: {error}</p>;
    if (!job) return <p>Loading…</p>;

    return (
        <div style={{ padding: 20 }}>
            <p>Result: {JSON.stringify(job.result)}</p>
            {job.result.map((match, index) => (
                <div key={match.Index}>
                    <strong>{index + 1}) Match Strength:</strong><span> {Number(match.Distance).toPrecision(2)} - </span>
                    <a href={`https://www.youtube.com/watch?v=${job.video_url}&t=${match.StartTime}`} target="_blank" rel="noopener noreferrer">Timestamp</a>
                </div>
            ))}
        </div>
    );
}

export default SemanticSearchResult;
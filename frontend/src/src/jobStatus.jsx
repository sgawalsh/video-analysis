import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

const TERMINAL_STATUSES = ['SUCCEEDED', 'FAILED'];

function JobStatus() {
  const { id } = useParams();
  const [job, setJob] = useState(null);

  useEffect(() => {
    let intervalId;

    const fetchJob = async () => {
      const res = await fetch(`/api/jobs/${id}`, { cache: 'no-store' });
      const data = await res.json();
      setJob(data);

      if (TERMINAL_STATUSES.includes(data.status)) {
        clearInterval(intervalId);
      }
    };

    // Initial fetch immediately
    fetchJob();

    // Start polling
    intervalId = setInterval(fetchJob, 2000);

    return () => clearInterval(intervalId);
  }, [id]);

  if (!job) return <p>Loading…</p>;

  return (
    <div style={{ padding: 20 }}>
      <p>Status: {job.status}</p>
      <p>Type: {job.type}</p>
      <p>URL: {job.video_url}</p>
      <p>Payload: {JSON.stringify(job.payload)}</p>
    </div>
  );
}

export default JobStatus;
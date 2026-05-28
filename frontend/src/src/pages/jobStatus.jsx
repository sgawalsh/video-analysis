import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';

function JobStatus() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [counts, setCounts] = useState(null);

  useEffect(() => {
    let intervalId;

    const fetchJob = async () => {
      const res = await fetch(`/api/jobs/${id}`, { cache: 'no-store' });
      const data = await res.json();
      setCounts(data);

      if (Number(data.succeeded) + Number(data.failed) === Number(data.total)) {
        clearInterval(intervalId);
      }
    };

    // Initial fetch immediately
    fetchJob();

    // Start polling
    intervalId = setInterval(fetchJob, 2000);

    return () => clearInterval(intervalId);
  }, [id]);

  if (!counts) return <p>Loading…</p>;

  return (
    <div style={{ padding: 20 }}>
      <h2>Job Status</h2>
      <p>Total Jobs: {counts.total}</p>
      <p>Succeeded: {counts.succeeded}</p>
      <p>Failed: {counts.failed}</p>
      <p>Pending: {counts.pending}</p>
      <p>Running: {counts.running}</p>
    </div>
  );
}

export default JobStatus;
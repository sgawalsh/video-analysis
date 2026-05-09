import { useState } from 'react'
import { useNavigate } from 'react-router-dom';

function JobForm() {
  const navigate = useNavigate();

  const [jobType, setJobType] = useState('SEMANTIC_SEARCH');
  const [videoUrl, setVideoUrl] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [error, setError] = useState(null);
  const YOUTUBE_URL_REGEX = /^(https?:\/\/)?(www\.)?(youtube\.com\/watch\?v=|youtu\.be\/)[A-Za-z0-9_-]+/;

  const submitJob = async (e) => {
    e.preventDefault();
    setError(null);

    if (!YOUTUBE_URL_REGEX.test(videoUrl)) {
      setError('Please enter a valid YouTube URL');
      return;
    }

    try {
      // Build request dynamically
      const requestBody = {
        type: jobType,
        video_url: videoUrl,
      };

      if (jobType === 'SEMANTIC_SEARCH') {
        requestBody.search_term = searchTerm;
      }

      const res = await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody),
      });

      if (!res.ok) {
        const payload = await res.json().catch(() => ({}));
        throw new Error(payload.error || 'Failed to submit job');
      }

      const data = await res.json();

      // SUCCESS → redirect
      navigate(`/jobs/${data.public_id}`);
    } catch (err) {
      // FAILURE → stay here
      setError(err.message);
    }
  };

  return (
    <div style={{ padding: 20 }}>
      <h1>Job Processor</h1>

      <form onSubmit={submitJob}>
        <div style={{ marginBottom: 12 }}>
          <label>
            Job Type:
            <select
              value={jobType}
              onChange={(e) => setJobType(e.target.value)}
              style={{ marginLeft: 10 }}
            >
              <option value="SEMANTIC_SEARCH">
                Semantic Search
              </option>

              <option value="TOPIC_DETECTION">
                Topic Detection
              </option>
            </select>
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
            <input
              value={videoUrl}
              onChange={(e) => setVideoUrl(e.target.value)}
              placeholder="Video URL"
              required
              style={{ width: 400 }}
            />
          </div>

        {/* Conditional Field */}
        {jobType === 'SEMANTIC_SEARCH' && (
          <div style={{ marginBottom: 12 }}>
            <input
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Search term"
              required
              style={{ width: 400 }}
            />
          </div>
        )}


        <button type="submit">Submit Job</button>
      </form>

      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  );
}

export default JobForm;
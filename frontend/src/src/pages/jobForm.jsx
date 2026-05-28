import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

function JobForm() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);

  // Grouped State Object
  const [formData, setFormData] = useState({
    mode: 'single',
    jobType: 'SEMANTIC_SEARCH',
    videoUrl: '',
    searchTerm: '',
    channelName: '',
    startDate: '',
    endDate: '',
  });

  const [error, setError] = useState(null);
  const YOUTUBE_URL_REGEX = /^(https?:\/\/)?(www\.)?(youtube\.com\/watch\?v=|youtu\.be\/)[A-Za-z0-9_-]+/;

  // Generic Change Handler for text, select, and radio inputs
  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const submitJob = async (e) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    const { mode, jobType, videoUrl, searchTerm, channelName, startDate, endDate } = formData;

    try {
      let requestBody = { type: jobType, mode: mode };

      if (mode === 'single') {
        if (!YOUTUBE_URL_REGEX.test(videoUrl)) {
          setError('Please enter a valid YouTube URL');
          setLoading(false);
          return;
        }
        requestBody.videoURL = videoUrl;
      } else {
        if (!channelName){
          setError('Channel name is required');
          setLoading(false);
          return;
        }
        if (!startDate || !endDate){
          setError('Date fields are required');
          setLoading(false);
          return;
        }

        const start = new Date(startDate);
        const end = new Date(endDate);
        if (start > end) {
          setError('Start date cannot be later than the end date');
          setLoading(false);
          return;
        }

        requestBody.channelName = channelName;
        requestBody.startDate = startDate;
        requestBody.endDate = endDate;
      }

      if (jobType === 'SEMANTIC_SEARCH') {
        requestBody.searchTerm = searchTerm;
      }

      const res = await fetch('/api/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody),
      });

      if (!res.ok) {
        const payload = await res.json().catch(() => ({}));
        throw new Error(payload.error || 'Failed to submit job');
      }

      const data = await res.json();
      navigate(`/sessions/${data.public_id}`);
    } catch (err) {
      setError(err.message);
      setLoading(false);
    }
  };

  return (
    <div style={{ padding: 20, textAlign: 'center' }}>
      <h1>Job Processor</h1>

      <form onSubmit={submitJob}>
        <fieldset disabled={loading} style={{ border: 'none', padding: 0, margin: 0 }}>

          <div style={{ marginBottom: 12 }}>
            <label>
              <input
                type="radio"
                name="mode"
                value="single"
                checked={formData.mode === "single"}
                onChange={handleChange}
              />
              Single Video
            </label>

            <label style={{ marginLeft: 16 }}>
              <input
                type="radio"
                name="mode"
                value="channel"
                checked={formData.mode === "channel"}
                onChange={handleChange}
              />
              Channel Search
            </label>
          </div>

          <div style={{ marginBottom: 12 }}>
            <label>
              Job Type:
              <select
                name="jobType"
                value={formData.jobType}
                onChange={handleChange}
                style={{ marginLeft: 10 }}
              >
                <option value="SEMANTIC_SEARCH">Semantic Search</option>
                <option value="TOPIC_DETECTION">Topic Detection</option>
              </select>
            </label>
          </div>

          {/* SINGLE VIDEO INPUT */}
          {formData.mode === "single" && (
            <div style={{ marginBottom: 12 }}>
              <input
                name="videoUrl"
                value={formData.videoUrl}
                onChange={handleChange}
                placeholder="Video URL"
                required
                style={{ width: 400 }}
              />
            </div>
          )}

          {/* CHANNEL INPUTS */}
          {formData.mode === "channel" && (
            <>
              <div style={{ marginBottom: 12 }}>
                <input
                  name="channelName"
                  value={formData.channelName}
                  onChange={handleChange}
                  placeholder="Channel name"
                  required
                  style={{ width: 400 }}
                />
              </div>

              <div style={{ marginBottom: 12, display: "flex", gap: 10, justifyContent: "center" }}>
                <input
                  type="date"
                  name="startDate"
                  value={formData.startDate}
                  onChange={handleChange}
                  max={formData.endDate}
                  required
                  style={{ width: 195 }} 
                />
                <input
                  type="date"
                  name="endDate"
                  value={formData.endDate}
                  onChange={handleChange}
                  min={formData.startDate}
                  required
                  style={{ width: 195 }}
                />
              </div>
            </>
          )}

          {/* Conditional Field */}
          {formData.jobType === 'SEMANTIC_SEARCH' && (
            <div style={{ marginBottom: 12 }}>
              <input
                name="searchTerm"
                value={formData.searchTerm}
                onChange={handleChange}
                placeholder="Search term"
                required
                style={{ width: 400 }}
              />
            </div>
          )}

          <button 
            type="submit" 
            disabled={loading}
            style={{ opacity: loading ? 0.7 : 1, cursor: loading ? 'not-allowed' : 'pointer' }}
          >
            {loading ? 'Processing Job...' : 'Submit Job'}
          </button>
        </fieldset>
      </form>

      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  );
}

export default JobForm;
import { useEffect, useState, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import SessionHeader from './components/sessionHeader';
import SessionResultRouter from './results/sessionResultRouter'

function SessionStatus() {
  const { id } = useParams();
  const [sessionState, setSessionState] = useState(null);
  const latestEmittedAt = useRef(0);
  const navigate = useNavigate();

  useEffect(() => {
    let es;

    async function loadSession() {
      try {
        const res = await fetch(`/api/sessions/${id}`);

        if (res.status === 404) {
          navigate('/*', { replace: true });
          return;
        }

        if (!res.ok) {
          navigate('/error', { replace: true });
          return;
        }
        const data = await res.json();

        setSessionState(data);
        console.log("Initial session data loaded:", data);

        // Initialize EventSource after fetching base data
        es = new EventSource(`/api/sessions/${id}/events`);

        es.addEventListener('status_changed', (e) => {
          const event = JSON.parse(e.data);
          console.log('status_changed fired');
          setSessionState((prev) => {
            if (!prev) return prev;

            // Ignore stale messages arriving out of order
            if (event.emittedAt && event.emittedAt < latestEmittedAt.current) {
              return prev;
            }
            if (event.emittedAt) {
              latestEmittedAt.current = event.emittedAt;
            }

            return {
              ...prev,
              counts: event.counts // Safely updates counts
            };
          });
        });

        es.addEventListener('job_completed', (e) => {
          const event = JSON.parse(e.data);
          console.log('job_completed fired');
          console.log('incoming event: ', event)

          setSessionState((prev) => {
            if (!prev) return prev;

            return {
              ...prev,
              results: {
                ...(prev.results || {}),
                [event.target_id]: {
                  title: event.title,
                  data: event.result
                }
              }
            };
          });
        });

        es.addEventListener('job_failed', (e) => {
          const event = JSON.parse(e.data);
          console.log('job_failed fired');
          setSessionState((prev) => {
            if (!prev) return prev;

            return {
              ...prev,
              errorMessages:[
                ...(prev.errorMessages || []),
                {
                  target_id: event.target_id,
                  title: event.title,
                  message: event.error_message,
                },
              ]
            };
          });
        });

        es.addEventListener('channel_search_succeeded', (e) => {
          const event = JSON.parse(e.data);
          console.log('channel_search_succeeded fired');
          setSessionState((prev) => {
            if (!prev) return prev;

            return {
              ...prev,
              channelSearchInfo: {
                status: 'SUCCEEDED',
                lastError: null
              }
            };
          });
        });

        es.addEventListener('channel_search_failed', (e) => {
          const event = JSON.parse(e.data);
          console.log(event)
          console.log('channel_search_failed fired');
          setSessionState((prev) => {
            if (!prev) return prev;

            return {
              ...prev,
              channelSearchInfo: {
                status: 'FAILED',
                lastError: event.error_message
              }
            };
          });
        });

        es.onerror = (err) => {
          console.error("EventSource encountered a connection error:", err);
        };

      } catch (err) {
        console.error("Failed to initialize session view:", err);
      }
    }

    loadSession();

    return () => {
      es?.close();
    };
  }, [id]);

  if (!sessionState) return <main className="container status-message"><p>Loading…</p></main>;

if (['PENDING', 'RUNNING'].includes(sessionState.channelSearchInfo.status)) {
  return (
    <main className="container status-message">
      <p>Channel search is active...</p>
    </main>
  );
  } else if (sessionState.channelSearchInfo.status === 'FAILED') {
    return (
      <main className="container status-message">
        <h3>Channel search failed.</h3>
        {sessionState.channelSearchInfo.lastError && (
          <p>Error Message: {sessionState.channelSearchInfo.lastError}</p>
        )}
      </main>
    );
  }

  if (sessionState.counts.total == 0) {
    return (
      <main className="container status-message">
        <p>No jobs were found for this session. Please check your input parameters.</p>
      </main>
    );
  }

  return (
    <>
      <SessionHeader counts = {sessionState.counts} errors={sessionState.errorMessages}/>
      <SessionResultRouter jobType={sessionState.type} results={sessionState.results } />
    </>
  );
}

export default SessionStatus;
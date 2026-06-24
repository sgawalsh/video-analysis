import { useEffect, useState, useRef } from 'react';
import { useParams } from 'react-router-dom';
import SessionHeader from './sessionHeader';
import SessionResultRouter from './results/sessionResultRouter'

function SessionStatus() {
  const { id } = useParams();
  const [sessionState, setSessionState] = useState(null);
  const latestEmittedAt = useRef(0);

  useEffect(() => {
    let es;

    async function loadSession() {
      try {
        const res = await fetch(`/api/sessions/${id}`);
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

            const targetId = event.target_id;

            const incoming = (event.result ?? []).map(item => ({
              ...item,
              target_id: targetId,
            }));
          console.log('incoming: ', incoming)

            return {
              ...prev,
              results: {
                ...(prev.results || {}),
                [targetId]: [
                  ...((prev.results || {})[targetId] || []),
                  ...incoming
                ]
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

  if (!sessionState) return <p>Loading…</p>;

  return (
    <>
      <SessionHeader counts = {sessionState.counts} />
      <SessionResultRouter jobType={sessionState.type} results={sessionState.results} />
    </>
  );
}

export default SessionStatus;
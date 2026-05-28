import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import SessionHeader from './sessionHeader';
import SessionResultRouter from './results/sessionResultRouter'

function SessionStatus() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [sessionState, setSessionState] = useState(null);

  useEffect(() => {
    let es;

    async function loadSession() {
      const res = await fetch(`/api/jobs/${id}`);
      const data = await res.json();

      setSessionState(data);

      es = new EventSource(
        `/api/sessions/${id}/events`
      );
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
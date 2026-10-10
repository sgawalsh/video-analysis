import VideoTitle from "./videoTitle";
import { useState } from "react";

function SessionHeader({ counts, errors }) {
  const [showErrors, setShowErrors] = useState(false);

  return (
    <main className="container session-page">
      <header className="session-header">
        <h2>Job Status</h2>
      </header>


      <section className="status-panel">
        <table className="status-table">
          <thead>
            <tr>
              <th scope="col">Total Jobs</th>
              <th scope="col">Succeeded</th>
              <th scope="col">Failed</th>
              <th scope="col">Running</th>
              <th scope="col">Pending</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>{counts.total}</td>
              <td>
                <span className={counts.succeeded > 0 ? "status-value status-succeeded" : ""}>
                  {counts.succeeded}
                </span>
              </td>
              <td>
                <span className="failed-count">
                  <span className={counts.failed > 0 ? "status-value status-failed" : ""}>
                    {counts.failed}
                  </span>

                  {counts.failed > 0 && (
                    <button
                      type="button"
                      className="error-trigger"
                      onClick={() => setShowErrors(true)}
                      aria-label={`View ${counts.failed} failed jobs`}
                      title="View failed jobs"
                    >
                      ⚠
                    </button>
                  )}
                </span>
              </td>
              <td>
                <span className={counts.running > 0 ? "status-value status-running" : ""}>
                  {counts.running}
                </span>
              </td>
              <td>
                <span className={counts.pending > 0 ? "status-value status-pending" : ""}>
                  {counts.pending}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </section>

      {showErrors && (
        <dialog open className="errors-dialog">
          <article>
            <header className="errors-dialog-header">
              <div>
                <h3>Failed Jobs</h3>
                <p>{errors.length} job{errors.length !== 1 ? "s" : ""} with errors</p>
              </div>

              <button
                type="button"
                className="secondary outline close-button"
                onClick={() => setShowErrors(false)}
                aria-label="Close failed jobs"
              >
                &times;
              </button>
            </header>

            {errors.length > 0 ? (
              <ul className="error-list">
                {errors.map((e) => (
                  <li key={e.target_id}>
                    <a
                      href={`https://www.youtube.com/watch?v=${e.target_id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {e.target_id}
                    </a>
                    <p className="error-video-title">
                      <VideoTitle targetId={e.target_id} />
                    </p>
                    <p className="error-message">{e.message}</p>
                  </li>
                ))}
              </ul>
            ) : (
              <p>No error details are available.</p>
            )}
          </article>
        </dialog>
      )}
    </main>
  );
}

export default SessionHeader;
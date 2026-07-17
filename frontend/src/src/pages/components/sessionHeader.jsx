import VideoTitle from "./videoTitle";
import { useState } from 'react';

function SessionHeader({ counts, errors }) {
  const [showErrors, setShowErrors] = useState(false);
  return (
      <div style={{ padding: 20 }}>
        <h2>Job Status</h2>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th>Total Jobs</th>
                <th>Succeeded</th>
                <th>Failed</th>
                <th>Running</th>
                <th>Pending</th>
              </tr>
            </thead>
            <tbody> 
              <tr>
                <td>{counts.total}</td>
                <td>{counts.succeeded}</td>
                <td style={{ position: "relative" }} title={errors.map(e => `${e.target_id}: ${e.message}`).join('\n')}>
                  {counts.failed}

                  {counts.failed > 0 && (
                    <>
                      <span
                        onClick={() => setShowErrors((s) => !s)}
                        style={{ cursor: "pointer", marginLeft: 6, color: "red" }}
                      >
                        ⚠
                      </span>

                      {showErrors && errors.length > 0 && (
                        <div
                          style={{
                            position: "absolute",
                            top: "100%",
                            left: 0,
                            marginTop: 4,
                            width: "max-content",
                            maxWidth: 400,
                            background: "#fff",
                            border: "1px solid #ccc",
                            borderRadius: 6,
                            boxShadow: "0 4px 12px rgba(0,0,0,.15)",
                            padding: 12,
                            zIndex: 100,
                            textAlign: "left",
                          }}
                        >
                          <div style={{ 
                            display: 'flex', 
                            justifyContent: 'space-between', 
                            alignItems: 'center',
                            gap: 16,
                            paddingBottom: 4
                          }}>
                            <strong>Failed Jobs</strong>
                            <button
                              onClick={() => setShowErrors(false)}
                              style={{
                                background: 'none',
                                border: 'none',
                                fontSize: 18,
                                cursor: 'pointer',
                                padding: '0 4px',
                                color: '#999',
                                lineHeight: 1
                              }}
                              title="Close popup"
                            >
                              &times;
                            </button>
                          </div>

                          {errors.map((e) => (
                            <div
                              key={e.target_id}
                              style={{
                                marginTop: 10,
                                paddingTop: 10,
                                borderTop: "1px solid #eee",
                              }}
                            >
                              <div>
                                <span>
                                  <strong><VideoTitle targetId={e.target_id} /></strong> - {e.message}
                                </span>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </>
                  )}
                </td>
                <td>{counts.running}</td>
                <td>{counts.pending}</td>
              </tr>
          </tbody>
        </table>
      </div>
  )
}
export default SessionHeader;
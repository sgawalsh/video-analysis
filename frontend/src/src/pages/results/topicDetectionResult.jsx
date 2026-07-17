import FormatMMSS from '../../format'
import LinkedThumbnailTitle from "../components/linkedThumbnailTitle";

function TopicDetectionResult({ results }) {
  const rows = Object.entries(results || {})
      .map(([targetId, topics]) => ({
        targetId,
        topics
      }))
      .sort((a, b) => a.targetId.localeCompare(b.targetId));

  return (
    <div style={{ padding: 20 }}>
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr>
            <th style={{ padding: 8, borderBottom: "2px solid #000" }}>
              Video
            </th>
            <th style={{ padding: 8, borderBottom: "2px solid #000" }}>
              Chapters
            </th>
          </tr>
        </thead>

        <tbody>
          {rows.map((row) => (
            <tr key={row.targetId}>
              {/* LEFT: video info */}
              <td style={{ padding: 8, verticalAlign: "top", width: 220 }}>
                  <LinkedThumbnailTitle targetId={row.targetId}></LinkedThumbnailTitle>
              </td>

              {/* RIGHT: chapters */}
              <td style={{ padding: 8, verticalAlign: "top" }}>
                {row.topics.map((topic, i) => (
                  <div
                    key={i}
                    style={{
                      marginBottom: 8,
                      padding: "6px 8px",
                      background: "#f5f5f5",
                      borderRadius: 4,
                    }}
                  >
                    <a
                      href={`https://www.youtube.com/watch?v=${row.targetId}&t=${topic.StartTime}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={{ textDecoration: "none" }}
                    >
                      {topic.Text.replace(/^"|"$/g, "")}
                      {" "}
                      <span style={{ color: "#666" }}>
                        ({FormatMMSS(topic.StartTime)}s)
                      </span>
                    </a>
                  </div>
                ))}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default TopicDetectionResult;
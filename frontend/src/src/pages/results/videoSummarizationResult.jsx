import ReactMarkdown from "react-markdown";
import LinkedThumbnailTitle from "./components/linkedThumbnailTitle";

function VideoSummarizationResult({ results }) {
  // Convert object {"l-YHm...": [...]} into an array of [key, value] pairs
  const videoEntries = Object.entries(results || {}).map(([targetId, summary]) => ({
        targetId,
        summary
      }));

  return (
    <div style={{ padding: 20 }}>
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr>
            <th style={{ padding: 8, borderBottom: "2px solid #000" }}>
              Video
            </th>
            <th style={{ padding: 8, borderBottom: "2px solid #000" }}>
              Summary
            </th>
          </tr>
        </thead>

        <tbody>
          {videoEntries.map((row) => {

            return (
              <tr key={row.targetId}>
                {/* LEFT: video info */}
                <td style={{ padding: 8, verticalAlign: "top", width: 220 }}>
                  <LinkedThumbnailTitle targetId={row.targetId}></LinkedThumbnailTitle>
                </td>

                {/* RIGHT: summary */}
                <td style={{ padding: 8, verticalAlign: "top" }}>
                  <div style={{ marginBottom: 4, textAlign: "left" }}>
                    <ReactMarkdown>
                      {row.summary[0].Text || { Text: "" }.replace(/^"|"$/g, "")}
                    </ReactMarkdown>
                  </div>
                  
                  <a
                    href={`https://www.youtube.com/watch?v=${row.targetId}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    style={{ textDecoration: "none" }}
                  >
                    <span style={{ color: "#0066cc" }}>
                      Link
                    </span>
                  </a>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export default VideoSummarizationResult;

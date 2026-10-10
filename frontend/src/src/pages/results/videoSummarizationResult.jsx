import ReactMarkdown from "react-markdown";
import LinkedThumbnailTitle from "../components/linkedThumbnailTitle";

function VideoSummarizationResult({ results }) {
const rows = Object.entries(results || {}).map(([id, content]) => ({
targetId: id,
title: content.title,
summary: content.data?.[0]?.Text || "",
}));

  return (
    <main className="container results-page">
      <div className="results-table-wrapper">
        <table className="results-table summary-results-table">
          <thead>
            <tr>
              <th scope="col">Video</th>
              <th scope="col">Summary</th>
            </tr>
          </thead>

          <tbody>
            {rows.map((row) => (
              <tr key={row.targetId}>
                <td className="result-video">
                  <LinkedThumbnailTitle
                    title={row.title}
                    targetId={row.targetId}
                  />
                </td>

                <td className="summary-cell">
                  <div className="summary-content">
                    <ReactMarkdown>{row.summary}</ReactMarkdown>
                  </div>

                  <div className="summary-footer">
                    <a
                      href={`https://www.youtube.com/watch?v=${row.targetId}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      Link
                    </a>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </main>

  );
}

export default VideoSummarizationResult;
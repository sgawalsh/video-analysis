import FormatMMSS from '../../format'
import LinkedThumbnailTitle from "../components/linkedThumbnailTitle";

function TopicDetectionResult({ results }) {

  const rows = Object.entries(results || {}).flatMap(([id, content]) =>
    ({
      targetId: id,
      title: content.title,
      topics: content.data
    })
  )

  return (
    <main className="container results-page">
      {rows.length > 0 ? 
        <div className="results-table-wrapper">
          <table className="results-table topic-results-table">
            <thead>
              <tr>
                <th scope="col">Video</th>
                <th scope="col">Detected Chapters</th>
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

                  <td className="topic-list-cell">
                    <div className="topic-list">
                      {row.topics.map((topic, i) => (
                        <a
                          key={`${row.targetId}-${topic.StartTime}-${i}`}
                          className="topic-item"
                          href={`https://www.youtube.com/watch?v=${row.targetId}&t=${topic.StartTime}`}
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          <span className="topic-timestamp">
                            {FormatMMSS(topic.StartTime)}
                          </span>

                          <span className="topic-text">
                            {topic.Text.replace(/^"|"$/g, "")}
                          </span>
                        </a>
                      ))}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        :<></>}
    </main>
  );
}

export default TopicDetectionResult;
function SessionHeader({ counts }) {
  return (
      <div style={{ padding: 20 }}>
        <h2>Job Status</h2>
        <table>
          <tbody>
            <tr>
              <td>Total Jobs: {counts.total}</td>
              <td>Succeeded: {counts.succeeded}</td>
              <td>Failed: {counts.failed}</td>
              <td>Pending: {counts.pending}</td>
              <td>Running: {counts.running}</td>
            </tr>
          </tbody>
        </table>
    </div>
  )
}
export default SessionHeader;